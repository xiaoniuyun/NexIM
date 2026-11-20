package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"nexim/server/internal/config"
	"nexim/server/internal/gateway"
	"nexim/server/internal/model"
	"nexim/server/internal/pkg"
	"nexim/server/internal/service"
	"nexim/server/internal/store"
	"nexim/server/webassets"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Router struct {
	S         *store.Store
	C         *config.Config
	Auth      *service.AuthService
	Msg       *service.MessageService
	Contact   *service.ContactService
	Group     *service.GroupService
	File      *service.FileService
	Admin     *service.AdminService
	Call      *service.CallService
	Fav       *service.FavoriteService
	Blacklist *service.BlacklistService
	Hub       *gateway.Hub
	Up        websocket.Upgrader
}

func NewRouter(s *store.Store, c *config.Config, auth *service.AuthService, msg *service.MessageService,
	contact *service.ContactService, group *service.GroupService, file *service.FileService,
	admin *service.AdminService, call *service.CallService, hub *gateway.Hub) *Router {
	return &Router{
		S: s, C: c, Auth: auth, Msg: msg, Contact: contact, Group: group, File: file, Admin: admin, Call: call,
		Fav: service.NewFavorite(s), Blacklist: service.NewBlacklist(s), Hub: hub,
		Up: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (rt *Router) Engine() *gin.Engine {
	gin.SetMode(rt.C.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery(), traceMiddleware(), corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok", "profile": rt.C.Deployment.Profile,
			"mode": rt.C.Deployment.Mode, "services": rt.C.Deployment.Services,
			"mq": gin.H{
				"driver": rt.C.MQ.Driver, "url": rt.C.MQ.URL,
				"stream": rt.C.MQ.Stream, "retention": rt.C.MQ.Retention,
			},
			"messageDB": gin.H{
				"driver": rt.C.DB.MessageDriver, "keyspace": rt.C.DB.Scylla.Keyspace,
				"hosts": rt.C.DB.Scylla.Hosts,
			},
			"summary": config.Summary(),
		})
	})

	api := r.Group("/api/v1")
	{
		// 认证（匿名）
		api.POST("/auth/login", rt.login)
		api.POST("/auth/register", rt.register)
		api.POST("/auth/refresh", rt.refresh)
		api.POST("/auth/sms-code", rt.smsCode)
		api.POST("/auth/password/forgot", rt.forgot)
		api.GET("/notices", rt.publicNotices)
		// 默认头像：用户未上传头像时由前端兜底展示
		api.GET("/assets/default-avatar.svg", rt.defaultAvatar)
		// 文件下载：匿名入口，handler 内用 Bearer 或 query token 校验
		api.GET("/files/:id/download", rt.fileDownload)

		// 以下需鉴权
		auth := api.Group("")
		auth.Use(rt.authMiddleware())
		{
			auth.POST("/auth/logout", rt.logout)
			auth.GET("/auth/permissions", rt.permissions)
			auth.GET("/auth/devices", rt.devices)
			auth.POST("/auth/devices/bind", rt.bindDevice)
			auth.DELETE("/auth/devices/:deviceId", rt.deleteDevice)

			auth.GET("/users/search", rt.userSearch)
			auth.GET("/users/me", rt.myProfile)
			auth.PUT("/users/me", rt.updateMyProfile)
			auth.GET("/users", rt.users)
			auth.GET("/users/:id/profile", rt.userProfile)

			auth.GET("/friends", rt.friends)
			auth.POST("/friends/apply", rt.friendApply)
			auth.GET("/friends/applies", rt.friendApplies)
			auth.PUT("/friends/applies/:id", rt.friendHandle)

			auth.GET("/depts", rt.requireInternal(), rt.depts)
			auth.GET("/depts/tree", rt.requireInternal(), rt.deptTree)
			auth.GET("/depts/:id/members", rt.requireInternal(), rt.deptMembers)

			auth.GET("/conversations", rt.conversations)
			auth.DELETE("/conversations/:id", rt.conversationDelete)
			auth.PATCH("/conversations/:id/settings", rt.convSettings)
			auth.POST("/conversations/read-all", rt.readAll)
			auth.GET("/conversations/:id/messages", rt.messages)
			auth.POST("/messages/read", rt.markRead)
			auth.POST("/messages/send", rt.sendMessage)
			auth.POST("/messages/revoke", rt.revoke)

			// 收藏与黑名单
			auth.GET("/favorites", rt.favorites)
			auth.POST("/favorites", rt.favoriteAdd)
			auth.DELETE("/favorites/:id", rt.favoriteDelete)
			auth.GET("/contacts/blacklist", rt.blacklist)
			auth.POST("/contacts/blacklist", rt.blacklistAdd)
			auth.DELETE("/contacts/blacklist/:uid", rt.blacklistRemove)

			// 实时音视频
			auth.GET("/rtc/config", rt.rtcConfig)
			auth.GET("/calls", rt.callRecords)
			auth.POST("/calls/:id/quality", rt.callQualityReport)
			auth.POST("/messages/search", rt.messageSearch)
			auth.POST("/messages/forward", rt.messageForward)

			// 群聊
			auth.GET("/groups", rt.groups)
			auth.GET("/groups/:id", rt.groupDetail)
			auth.GET("/groups/:id/members", rt.groupMembers)
			auth.POST("/groups", rt.groupCreate)
			auth.POST("/groups/:id/invite", rt.groupInvite)
			auth.DELETE("/groups/:id/members/:uid", rt.groupRemoveMember)
			auth.DELETE("/groups/:id", rt.groupDissolve)

			// 文件
			auth.GET("/files", rt.files)
			auth.POST("/files/upload", rt.fileUpload)
			auth.DELETE("/files/:id", rt.fileDelete)

			// 管理后台：服务端按权限码校验（与前端菜单/按钮共用同一份权限目录）
			adm := auth.Group("/admin")
			{
				adm.GET("/dashboard", rt.requirePerm("dashboard:view"), rt.adminDashboard)
				adm.GET("/users", rt.requirePerm("user:list"), rt.adminUsers)
				adm.GET("/users-export", rt.requirePerm("user:export"), rt.adminUsersExport)
				adm.POST("/users", rt.requirePerm("user:create"), rt.adminCreateUser)
				adm.PUT("/users/:id", rt.requirePerm("user:edit"), rt.adminUpdateUser)
				adm.DELETE("/users/:id", rt.requirePerm("user:disable"), rt.adminDeleteUser)
				adm.POST("/users/:id/reset-password", rt.requirePerm("user:reset-pwd"), rt.adminResetPwd)
				adm.PUT("/users/:id/roles", rt.requirePerm("rbac:role:grant"), rt.adminSetUserRoles)
				adm.GET("/user-permissions/:id", rt.requirePerm("rbac:perm:preview"), rt.adminUserPermissions)
				adm.GET("/roles", rt.requirePerm("rbac:role:list"), rt.adminRoles)
				adm.GET("/permissions", rt.requirePerm("rbac:role:list"), rt.adminPermissions)
				adm.POST("/roles", rt.requirePerm("rbac:role:create"), rt.adminCreateRole)
				adm.PUT("/roles/:id", rt.requirePerm("rbac:role:edit"), rt.adminUpdateRole)
				adm.DELETE("/roles/:id", rt.requirePerm("rbac:role:delete"), rt.adminDeleteRole)
				adm.POST("/roles/grant", rt.requirePerm("rbac:role:grant"), rt.adminGrantRole)
				adm.GET("/roles/:id/members", rt.requirePerm("rbac:role:list"), rt.adminRoleMembers)
				adm.DELETE("/roles/:id/members/:uid", rt.requirePerm("rbac:role:grant"), rt.adminRoleRemoveMember)
				adm.POST("/depts", rt.requirePerm("dept:create"), rt.adminCreateDept)
				adm.PUT("/depts/:id", rt.requirePerm("dept:edit"), rt.adminUpdateDept)
				adm.DELETE("/depts/:id", rt.requirePerm("dept:delete"), rt.adminDeleteDept)
				adm.POST("/depts/:id/members", rt.requirePerm("dept:edit"), rt.adminDeptAddMembers)
				adm.DELETE("/depts/:id/members/:uid", rt.requirePerm("dept:edit"), rt.adminDeptRemoveMember)
				adm.GET("/audit", rt.requirePerm("log:list"), rt.adminAudit)
				adm.GET("/audit/export", rt.requirePerm("log:export"), rt.adminAuditExport)
				adm.GET("/audit/messages", rt.requirePerm("audit:message:list"), rt.adminMessageAudit)
				adm.GET("/audit/messages/export", rt.requirePerm("audit:message:export"), rt.adminMessagesExport)
				adm.GET("/notices", rt.requirePerm("notice:publish"), rt.adminNotices)
				adm.POST("/notices", rt.requirePerm("notice:publish"), rt.adminCreateNotice)
				adm.PUT("/notices/:id", rt.requirePerm("notice:publish"), rt.adminUpdateNotice)
				adm.GET("/sensitive-words", rt.requirePerm("sensitive:config"), rt.adminSensitiveWords)
				adm.POST("/sensitive-words", rt.requirePerm("sensitive:config"), rt.adminAddSensitiveWord)
				adm.POST("/sensitive-words/bulk", rt.requirePerm("sensitive:config"), rt.adminBulkSensitiveWords)
				adm.DELETE("/sensitive-words/:word", rt.requirePerm("sensitive:config"), rt.adminDeleteSensitiveWord)
				adm.GET("/sensitive/strategy", rt.requirePerm("sensitive:config"), rt.adminSensitiveStrategy)
				adm.PUT("/sensitive/strategy", rt.requirePerm("sensitive:config"), rt.adminSetSensitiveStrategy)
				adm.GET("/sensitive/hits", rt.requirePerm("sensitive:config"), rt.adminSensitiveHits)
				adm.DELETE("/sensitive/hits", rt.requirePerm("sensitive:config"), rt.adminClearSensitiveHits)
				// 通话 / 质量 / 群组 / 设置 / 登录日志
				adm.GET("/calls", rt.requirePerm("rtc:record:list"), rt.adminCalls)
				adm.GET("/calls-export", rt.requirePerm("rtc:record:export"), rt.adminCallsExport)
				adm.POST("/calls/:id/end", rt.requirePerm("rtc:record:manage"), rt.adminEndCall)
				adm.GET("/rtc/quality", rt.requirePerm("rtc:quality:view"), rt.adminRTCQuality)
				adm.GET("/groups", rt.requirePerm("menu:groups"), rt.adminGroups)
				adm.GET("/groups-export", rt.requirePerm("group:export"), rt.adminGroupsExport)
				adm.PUT("/groups/:id/status", rt.requirePerm("group:manage"), rt.adminGroupStatus)
				adm.DELETE("/groups/:id", rt.requirePerm("group:manage"), rt.adminGroupDissolve)
				adm.GET("/settings", rt.requirePerm("menu:settings"), rt.adminGetSettings)
				adm.PUT("/settings", rt.requirePerm("setting:config"), rt.adminSaveSettings)
				adm.GET("/login-logs", rt.requirePerm("log:login:list"), rt.adminLoginLogs)
				adm.GET("/login-logs-export", rt.requirePerm("log:login:list"), rt.adminLoginLogsExport)
				adm.POST("/login-logs/ban", rt.requirePerm("log:login:ban"), rt.adminBanIP)
			}
		}
	}

	// WebSocket（同源，供嵌入的前端使用；独立网关端口仍保留）
	r.GET("/ws", func(c *gin.Context) { rt.WSHandler(c.Writer, c.Request) })

	// 嵌入的前端静态资源（H5 / Admin），单二进制同源部署
	if webassets.Has() {
		r.GET("/h5", func(c *gin.Context) { c.Redirect(http.StatusFound, "/h5/") })
		r.GET("/h5/*filepath", gin.WrapH(webassets.FileServer("h5")))
		r.GET("/admin", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin/") })
		r.GET("/admin/*filepath", gin.WrapH(webassets.FileServer("admin")))
		r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/h5/") })
	}
	return r
}

func traceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("traceId", pkg.NewTrace())
		c.Next()
	}
}

// corsMiddleware：桌面端（app:// 自定义协议）与移动端跨域访问需要；
// 仅使用 Authorization 头鉴权、不依赖 Cookie，因此允许任意来源。
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Requested-With,X-Trace-Id")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (rt *Router) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			pkg.Fail(c, 401, 2001, "缺少访问令牌")
			c.Abort()
			return
		}
		claims, err := pkg.ParseToken(rt.C.Auth.JWTSecret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			pkg.Fail(c, 401, 2001, "令牌无效或过期")
			c.Abort()
			return
		}
		if rt.Auth.CheckBlacklist(claims.ID) {
			pkg.Fail(c, 401, 2002, "令牌已失效")
			c.Abort()
			return
		}
		// 禁用账号即时阻断（无需等待令牌过期）
		var active model.User
		if rt.S.DB.Select("status").Where("uid = ?", claims.UID).First(&active).Error == nil && active.Status != 1 {
			pkg.Fail(c, 401, 2003, "账号已禁用")
			c.Abort()
			return
		}
		c.Set("uid", claims.UID)
		c.Set("claims", claims)
		c.Next()
	}
}

func uid(c *gin.Context) string { return c.GetString("uid") }

// requirePerm 服务端 RBAC 校验：与 /auth/permissions 下发的权限码同源，* 通配放行
func (rt *Router) requirePerm(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !service.HasPerm(rt.S, uid(c), code) {
			pkg.Fail(c, 403, 3001, "无权限执行此操作："+code)
			c.Abort()
			return
		}
		c.Next()
	}
}

// requireInternal 组织架构仅对内部员工开放；外部联系人默认无法查询组织架构相关人员
func (rt *Router) requireInternal() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rt.Contact.IsExternal(uid(c)) {
			pkg.Fail(c, 403, 3003, "外部联系人无法查看组织架构")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (rt *Router) WSHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	claims, err := pkg.ParseToken(rt.C.Auth.JWTSecret, token)
	if err != nil || rt.Auth.CheckBlacklist(claims.ID) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var active model.User
	if rt.S.DB.Select("status").Where("uid = ?", claims.UID).First(&active).Error == nil && active.Status != 1 {
		http.Error(w, "account disabled", 401)
		return
	}
	deviceID := r.URL.Query().Get("device")
	if deviceID == "" {
		deviceID = "web"
	}
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "H5"
	}
	conn, err := rt.Up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	cl := gateway.NewClient(conn, claims.UID, deviceID, platform, claims, rt.Hub)
	rt.Hub.Register(cl)
	go cl.WritePump()
	cl.ReadPump()
}

// ============ 认证处理器 ============
func (rt *Router) login(c *gin.Context) {
	var req struct {
		Account  string `json:"account" binding:"required"`
		Password string `json:"password" binding:"required"`
		DeviceID string `json:"deviceId"`
		Platform string `json:"platform"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	ip := c.ClientIP()
	platform := req.Platform
	if platform == "" {
		platform = "H5"
	}
	if rt.Admin.IsIPBanned(ip) {
		rt.Admin.RecordLogin("", req.Account, ip, platform, req.DeviceID, false)
		pkg.Fail(c, 403, 3002, "该 IP 已被封禁，请联系管理员")
		return
	}
	data, err := rt.Auth.Login(req.Account, req.Password, req.DeviceID, req.Platform)
	if err != nil {
		rt.Admin.RecordLogin("", req.Account, ip, platform, req.DeviceID, false)
		pkg.Fail(c, 401, 2000, err.Error())
		return
	}
	loginUID := fmt.Sprint(data["uid"])
	rt.Admin.RecordLogin(loginUID, req.Account, ip, platform, req.DeviceID, true)
	pkg.OK(c, data)
}

func (rt *Router) register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	data, err := rt.Auth.Register(req.Username, req.Password, req.Nickname)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	// 自助注册同样默认普通用户并加入公司全员群
	if newUID, ok := data["uid"].(string); ok && newUID != "" {
		if err := rt.Admin.EnsureDefaultRole(newUID); err != nil {
			rt.S.Log.Warn("default role assign failed", zap.String("uid", newUID), zap.Error(err))
		}
		if err := rt.Group.JoinCompanyGroup(newUID); err != nil {
			rt.S.Log.Warn("join company group failed", zap.String("uid", newUID), zap.Error(err))
		}
	}
	pkg.OK(c, data)
}

func (rt *Router) refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	data, err := rt.Auth.Refresh(req.RefreshToken)
	if err != nil {
		pkg.Fail(c, 401, 2001, err.Error())
		return
	}
	pkg.OK(c, data)
}

func (rt *Router) smsCode(c *gin.Context) {
	var req struct {
		Target string `json:"target" binding:"required"`
		Scene  string `json:"scene"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if req.Scene == "" {
		req.Scene = "login"
	}
	if err := rt.Auth.SmsCode(req.Target, req.Scene); err != nil {
		pkg.Fail(c, 429, 5000, err.Error())
		return
	}
	pkg.OK(c, gin.H{"demoCode": "123456"})
}

func (rt *Router) forgot(c *gin.Context) {
	pkg.OK(c, gin.H{"message": "演示模式：请使用短信验证码重置"})
}

func (rt *Router) logout(c *gin.Context) {
	cl := c.MustGet("claims").(*pkg.Claims)
	_ = rt.Auth.Logout(cl.ID, cl.ExpiresAt.Unix())
	pkg.OK(c, nil)
}

func (rt *Router) permissions(c *gin.Context) {
	pkg.OK(c, rt.Auth.Permissions(uid(c)))
}

func (rt *Router) devices(c *gin.Context) {
	pkg.OK(c, rt.Auth.Devices(uid(c)))
}

func (rt *Router) bindDevice(c *gin.Context) {
	var req struct {
		DeviceID string `json:"deviceId" binding:"required"`
		Platform string `json:"platform"`
		Channel  string `json:"channel"`
		Token    string `json:"token"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if err := rt.Auth.BindDevice(uid(c), req.DeviceID, req.Platform, req.Channel, req.Token); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, nil)
}

func (rt *Router) deleteDevice(c *gin.Context) {
	deviceID := c.Param("deviceId")
	rt.S.DB.Where("uid = ? AND device_id = ?", uid(c), deviceID).Delete(&model.Device{})
	pkg.OK(c, nil)
}

// ============ 联系人/组织 ============
func (rt *Router) userSearch(c *gin.Context) {
	pkg.OK(c, rt.Contact.Users(c.Query("q"), uid(c)))
}
func (rt *Router) users(c *gin.Context) {
	pkg.OK(c, rt.Contact.Users("", uid(c)))
}
func (rt *Router) userProfile(c *gin.Context) {
	u := rt.Contact.Users(c.Param("id"), uid(c))
	if len(u) > 0 {
		pkg.OK(c, u[0])
	} else {
		pkg.OK(c, nil)
	}
}

// myProfile 返回当前登录用户的完整资料
func (rt *Router) myProfile(c *gin.Context) {
	pkg.OK(c, rt.myProfileData(uid(c)))
}

// updateMyProfile 更新昵称、头像、性别、个性签名等基础资料
func (rt *Router) updateMyProfile(c *gin.Context) {
	var req struct {
		Nickname  string `json:"nickname"`
		Avatar    string `json:"avatar"`
		Signature string `json:"signature"`
		Gender    int    `json:"gender"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Signature = strings.TrimSpace(req.Signature)
	if n := utf8.RuneCountInString(req.Nickname); n < 1 || n > 32 {
		pkg.Fail(c, 400, 1001, "昵称需为 1-32 个字符")
		return
	}
	if utf8.RuneCountInString(req.Signature) > 120 {
		pkg.Fail(c, 400, 1001, "个性签名不能超过 120 个字符")
		return
	}
	if req.Gender < 0 || req.Gender > 2 {
		pkg.Fail(c, 400, 1001, "性别取值不合法")
		return
	}
	if len(req.Avatar) > 256 {
		pkg.Fail(c, 400, 1001, "头像地址过长")
		return
	}
	updates := map[string]interface{}{
		"nickname": req.Nickname, "avatar": req.Avatar,
		"signature": req.Signature, "gender": req.Gender, "updated_at": model.Now(),
	}
	if err := rt.S.DB.Model(&model.User{}).Where("uid = ?", uid(c)).Updates(updates).Error; err != nil {
		pkg.Fail(c, 500, 9000, "保存失败")
		return
	}
	pkg.OK(c, rt.myProfileData(uid(c)))
}

func (rt *Router) myProfileData(uidStr string) map[string]interface{} {
	var u model.User
	if err := rt.S.DB.Where("uid = ?", uidStr).First(&u).Error; err != nil {
		return nil
	}
	deptName := ""
	var d model.Dept
	if rt.S.DB.Where("id = ?", u.DeptID).First(&d).Error == nil {
		deptName = d.Name
	}
	return map[string]interface{}{
		"uid": u.UID, "username": u.Username, "nickname": u.Nickname,
		"avatar": u.Avatar, "signature": u.Signature, "gender": u.Gender,
		"phone": u.Phone, "title": u.Title, "deptId": u.DeptID, "deptName": deptName,
		"userType": u.UserType, "isExternal": service.IsExternalUser(&u),
	}
}
func (rt *Router) friends(c *gin.Context) {
	pkg.OK(c, rt.Contact.Friends(uid(c)))
}
func (rt *Router) friendApply(c *gin.Context) {
	var req struct {
		ToUID   string `json:"toUid" binding:"required"`
		Message string `json:"message"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if err := rt.Contact.ApplyFriend(uid(c), req.ToUID, req.Message); err != nil {
		pkg.Fail(c, 400, 4000, "好友申请失败")
		return
	}
	pkg.OK(c, nil)
}
func (rt *Router) friendApplies(c *gin.Context) {
	pkg.OK(c, rt.Contact.Applies(uid(c)))
}
func (rt *Router) friendHandle(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Accept bool `json:"accept"`
	}
	c.ShouldBindJSON(&req)
	if err := rt.Contact.HandleApply(uid(c), id, req.Accept); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	pkg.OK(c, nil)
}
func (rt *Router) depts(c *gin.Context) {
	pkg.OK(c, rt.Contact.Depts())
}
func (rt *Router) deptTree(c *gin.Context) {
	pkg.OK(c, rt.Admin.DeptTree())
}
func (rt *Router) deptMembers(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	pkg.OK(c, rt.Admin.DeptMembersFor(id, uid(c)))
}

// ============ 会话/消息 ============
func (rt *Router) conversations(c *gin.Context) {
	pkg.OK(c, rt.Msg.Conversations(uid(c)))
}
func (rt *Router) convSettings(c *gin.Context) {
	var req struct {
		Pinned *int `json:"pinned"`
		Muted  *int `json:"muted"`
	}
	c.ShouldBindJSON(&req)
	if err := rt.Msg.SetConversationSetting(uid(c), c.Param("id"), req.Pinned, req.Muted); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, nil)
}
func (rt *Router) readAll(c *gin.Context) {
	_ = rt.Msg.ReadAll(uid(c))
	pkg.OK(c, nil)
}
func (rt *Router) messages(c *gin.Context) {
	pkg.OK(c, rt.Msg.ListMessages(uid(c), c.Param("id"), 200))
}
func (rt *Router) markRead(c *gin.Context) {
	var req struct {
		ConvID string `json:"convId" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	_ = rt.Msg.MarkRead(uid(c), req.ConvID)
	rt.Hub.NotifyRead(uid(c), req.ConvID)
	pkg.OK(c, nil)
}

// sendMessage 是 WebSocket SEND 帧的 HTTP 等价通道。
// WebSocket 不可用（代理未放行 Upgrade、网络抖动）时，前端仍可可靠发消息并触发实时推送；
// 幂等由 clientMsgId 保证，与 WS 通道重复提交不会产生两条消息。
func (rt *Router) sendMessage(c *gin.Context) {
	var req struct {
		ClientMsgID string `json:"clientMsgId" binding:"required"`
		ConvID      string `json:"convId" binding:"required"`
		ConvType    string `json:"convType"`
		ToUID       string `json:"toUid"`
		Type        string `json:"type"`
		Content     string `json:"content"`
		Extra       string `json:"extra"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	me := uid(c)
	toUID := req.ToUID
	if toUID == "" {
		toUID = rt.Msg.PeerOf(req.ConvID, me)
	}
	msg, err := rt.Msg.Send(me, req.ConvID, req.ConvType, toUID, req.Type, req.Content, req.Extra, req.ClientMsgID)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	// Send 内部可能从 convId 补齐对端，推送以落库后的 ToID 为准
	if msg.ToID != "" {
		toUID = msg.ToID
	}
	rt.Hub.PushMessage(me, msg.ConvID, req.ConvType, toUID, req.ClientMsgID,
		msg.MsgID, req.Type, msg.Content, msg.Extra, msg.Seq, msg.CreatedAt)
	pkg.OK(c, gin.H{
		"clientMsgId": req.ClientMsgID, "msgId": msg.MsgID,
		"convId": msg.ConvID, "seq": msg.Seq, "sendTime": msg.CreatedAt,
	})
}

func (rt *Router) revoke(c *gin.Context) {
	var req struct {
		MsgID string `json:"msgId" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if err := rt.Msg.Revoke(uid(c), req.MsgID); err != nil {
		pkg.Fail(c, 400, 4006, err.Error())
		return
	}
	rt.Hub.BroadcastRevoke(req.MsgID)
	pkg.OK(c, nil)
}

// ============ 实时音视频 ============

// rtcConfig 下发 WebRTC ICE Servers（STUN/TURN），前端建立 PeerConnection 时使用
func (rt *Router) rtcConfig(c *gin.Context) {
	ice := make([]gin.H, 0, len(rt.C.RTC.STUNServers)+1)
	for _, u := range rt.C.RTC.STUNServers {
		if u != "" {
			ice = append(ice, gin.H{"urls": []string{u}})
		}
	}
	if rt.C.RTC.TurnURL != "" {
		ice = append(ice, gin.H{
			"urls":     []string{rt.C.RTC.TurnURL},
			"username": rt.C.RTC.TurnUsername, "credential": rt.C.RTC.TurnPassword,
		})
	}
	pkg.OK(c, gin.H{
		"iceServers": ice, "sfuNodes": rt.C.RTC.SFUNodes,
		"clusterEnabled": rt.C.RTC.ClusterEnabled, "coturnAddr": rt.C.RTC.CoturnAddr,
	})
}

// callRecords 当前用户的通话记录
func (rt *Router) callRecords(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	pkg.OK(c, rt.Call.Records(uid(c), limit))
}
