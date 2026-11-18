package api

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"nexim/server/internal/model"
	"nexim/server/internal/pkg"

	"github.com/gin-gonic/gin"
)

// ============ 公告 ============
func (rt *Router) publicNotices(c *gin.Context) {
	pkg.OK(c, rt.Admin.Notices())
}

// ============ 消息搜索/转发 ============
func (rt *Router) messageSearch(c *gin.Context) {
	var req struct {
		Keyword string `json:"keyword" binding:"required"`
		ConvID  string `json:"convId"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	list, err := rt.S.Msg.Search(req.Keyword, req.ConvID, 50)
	if err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, list)
}

func (rt *Router) messageForward(c *gin.Context) {
	var req struct {
		MsgID   string   `json:"msgId" binding:"required"`
		ConvIDs []string `json:"convIds" binding:"required"`
		Note    string   `json:"note"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	src, err := rt.S.Msg.GetByMsgID(req.MsgID)
	if err != nil {
		pkg.Fail(c, 404, 4000, "消息不存在")
		return
	}
	me := uid(c)
	content := "[转发] " + src.Content
	if req.Note != "" {
		content = req.Note + " · " + content
	}
	ok := 0
	for _, convID := range req.ConvIDs {
		// 解析会话对端
		var conv model.Conversation
		if rt.S.DB.Where("conv_id = ? AND owner_uid = ?", convID, me).First(&conv).Error != nil {
			continue
		}
		clientID := "fwd_" + src.MsgID + "_" + convID
		if _, err := rt.Msg.Send(me, convID, "single", conv.PeerID, "text", content, "", clientID); err == nil {
			ok++
		}
	}
	pkg.OK(c, gin.H{"forwarded": ok})
}

// ============ 群聊 ============
func (rt *Router) groups(c *gin.Context) {
	pkg.OK(c, rt.Group.MyGroups(uid(c)))
}
func (rt *Router) groupDetail(c *gin.Context) {
	g := rt.Group.Detail(c.Param("id"))
	if g == nil {
		pkg.Fail(c, 404, 4000, "群不存在")
		return
	}
	pkg.OK(c, g)
}
func (rt *Router) groupMembers(c *gin.Context) {
	pkg.OK(c, rt.Group.Members(c.Param("id")))
}
func (rt *Router) groupCreate(c *gin.Context) {
	var req struct {
		Name    string   `json:"name" binding:"required"`
		Members []string `json:"members"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	g, err := rt.Group.Create(uid(c), req.Name, req.Members)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "group:create", g.GroupID, req.Name)
	pkg.OK(c, g)
}
func (rt *Router) groupInvite(c *gin.Context) {
	var req struct {
		Members []string `json:"members" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	_ = rt.Group.Invite(c.Param("id"), req.Members)
	pkg.OK(c, nil)
}
func (rt *Router) groupRemoveMember(c *gin.Context) {
	_ = rt.Group.RemoveMember(c.Param("id"), c.Param("uid"))
	pkg.OK(c, nil)
}
func (rt *Router) groupDissolve(c *gin.Context) {
	_ = rt.Group.Dissolve(c.Param("id"))
	pkg.OK(c, nil)
}

// ============ 文件 ============
func (rt *Router) files(c *gin.Context) {
	pkg.OK(c, rt.File.List(uid(c)))
}
func (rt *Router) fileUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		pkg.Fail(c, 400, 1001, "缺少文件")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		pkg.Fail(c, 400, 1001, "读取文件失败")
		return
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	rec, err := rt.File.Upload(uid(c), fileHeader.Filename, fileHeader.Size, fileHeader.Header.Get("Content-Type"), c.PostForm("convId"), data)
	if err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, rec)
}
func (rt *Router) fileDownload(c *gin.Context) {
	// 匿名路由：优先 query token（<a> 直接下载），其次 Authorization: Bearer
	if c.GetString("uid") == "" {
		token := c.Query("token")
		if token == "" {
			token = strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		}
		if claims, e := pkg.ParseToken(rt.C.Auth.JWTSecret, token); e == nil && !rt.Auth.CheckBlacklist(claims.ID) {
			c.Set("uid", claims.UID)
		} else {
			pkg.Fail(c, 401, 2001, "未授权")
			return
		}
	}
	rec, obj, err := rt.File.Open(c.Param("id"))
	if err != nil {
		pkg.Fail(c, 404, 4000, "文件不存在")
		return
	}
	defer obj.Body.Close()
	ct := rec.MimeType
	if ct == "" {
		ct = obj.ContentType
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	headers := map[string]string{"Content-Disposition": contentDisposition(rec.Name)}
	if obj.Size < 0 {
		// 对象未返回长度（分块传输）：读全量后回写，避免 Content-Length 缺失导致截断
		data, rerr := io.ReadAll(obj.Body)
		if rerr != nil {
			pkg.Fail(c, 500, 9000, "读取文件失败")
			return
		}
		c.Data(http.StatusOK, ct, data)
		return
	}
	c.DataFromReader(http.StatusOK, obj.Size, ct, obj.Body, headers)
}

// contentDisposition 生成兼容中文文件名的下载头（RFC 5987）
func contentDisposition(name string) string {
	if name == "" {
		name = "download"
	}
	ascii := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)
	return `attachment; filename="` + ascii + `"; filename*=UTF-8''` + url.PathEscape(name)
}
func (rt *Router) fileDelete(c *gin.Context) {
	_ = rt.File.Delete(c.Param("id"), uid(c))
	pkg.OK(c, nil)
}

// ============ 管理后台 ============
func (rt *Router) adminDashboard(c *gin.Context) {
	pkg.OK(c, rt.Admin.Dashboard())
}
func (rt *Router) adminUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	pkg.OK(c, rt.Admin.UserList(page, size))
}
func (rt *Router) adminCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		DeptID   int64  `json:"deptId"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	u, err := rt.Admin.CreateUser(req.Username, req.Password, req.Nickname, req.DeptID)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "user:create", u.UID, req.Username)
	pkg.OK(c, u)
}
func (rt *Router) adminUpdateUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req map[string]interface{}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	_ = rt.Admin.UpdateUser(id, req)
	rt.Admin.Audit(uid(c), "user:edit", strconv.FormatInt(id, 10), "")
	pkg.OK(c, nil)
}
func (rt *Router) adminDeleteUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_ = rt.Admin.DeleteUser(id)
	rt.Admin.Audit(uid(c), "user:disable", strconv.FormatInt(id, 10), "")
	pkg.OK(c, nil)
}
func (rt *Router) adminResetPwd(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	pwd, err := rt.Admin.ResetPassword(id)
	if err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "user:reset-pwd", strconv.FormatInt(id, 10), "")
	pkg.OK(c, gin.H{"password": pwd})
}
func (rt *Router) adminRoles(c *gin.Context) {
	pkg.OK(c, rt.Admin.Roles())
}
func (rt *Router) adminGrantRole(c *gin.Context) {
	var req struct {
		UID    string `json:"uid" binding:"required"`
		RoleID int64  `json:"roleId" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	_ = rt.Admin.GrantRole(req.UID, req.RoleID)
	pkg.OK(c, nil)
}
func (rt *Router) adminAudit(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	pkg.OK(c, rt.Admin.AuditList(page, size))
}
func (rt *Router) adminMessageAudit(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	pkg.OK(c, rt.Admin.MessageAudit(page, size))
}
func (rt *Router) adminNotices(c *gin.Context) {
	pkg.OK(c, rt.Admin.Notices())
}
func (rt *Router) adminCreateNotice(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Author  string `json:"author"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if req.Author == "" {
		req.Author = "系统管理员"
	}
	n, err := rt.Admin.CreateNotice(req.Title, req.Content, req.Author)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	pkg.OK(c, n)
}
func (rt *Router) adminUpdateNotice(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req map[string]interface{}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	_ = rt.Admin.UpdateNotice(id, req)
	pkg.OK(c, nil)
}
func (rt *Router) adminSensitiveWords(c *gin.Context) {
	pkg.OK(c, rt.Admin.SensitiveWords())
}
func (rt *Router) adminAddSensitiveWord(c *gin.Context) {
	var req struct {
		Word string `json:"word" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	_ = rt.Admin.AddSensitiveWord(req.Word)
	pkg.OK(c, nil)
}
