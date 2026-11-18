package api

import (
	"net/http"
	"strconv"
	"strings"

	"nexim/server/internal/config"
	"nexim/server/internal/gateway"
	"nexim/server/internal/model"
	"nexim/server/internal/pkg"
	"nexim/server/internal/service"
	"nexim/server/internal/store"
	"nexim/server/webassets"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Router struct {
	S       *store.Store
	C       *config.Config
	Auth    *service.AuthService
	Msg     *service.MessageService
	Contact *service.ContactService
	Group   *service.GroupService
	File    *service.FileService
	Admin   *service.AdminService
	Call    *service.CallService
	Hub     *gateway.Hub
	Up      websocket.Upgrader
}

func NewRouter(s *store.Store, c *config.Config, auth *service.AuthService, msg *service.MessageService,
	contact *service.ContactService, group *service.GroupService, file *service.FileService,
	admin *service.AdminService, call *service.CallService, hub *gateway.Hub) *Router {
	return &Router{
		S: s, C: c, Auth: auth, Msg: msg, Contact: contact, Group: group, File: file, Admin: admin, Call: call, Hub: hub,
		Up: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (rt *Router) Engine() *gin.Engine {
	gin.SetMode(rt.C.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery(), traceMiddleware())

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
			auth.GET("/users", rt.users)
			auth.GET("/users/:id/profile", rt.userProfile)

			auth.GET("/friends", rt.friends)
			auth.POST("/friends/apply", rt.friendApply)
			auth.GET("/friends/applies", rt.friendApplies)
			auth.PUT("/friends/applies/:id", rt.friendHandle)

			auth.GET("/depts", rt.depts)
			auth.GET("/depts/tree", rt.deptTree)
			auth.GET("/depts/:id/members", rt.deptMembers)

			auth.GET("/conversations", rt.conversations)
			auth.PATCH("/conversations/:id/settings", rt.convSettings)
			auth.POST("/conversations/read-all", rt.readAll)
			auth.GET("/conversations/:id/messages", rt.messages)
			auth.POST("/messages/read", rt.markRead)
			auth.POST("/messages/send", rt.sendMessage)
			auth.POST("/messages/revoke", rt.revoke)

			// 实时音视频
			auth.GET("/rtc/config", rt.rtcConfig)
			auth.GET("/calls", rt.callRecords)
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

			// 管理后台（简化鉴权：登录用户可访问，前端按角色控制）
			adm := auth.Group("/admin")
			{
				adm.GET("/dashboard", rt.adminDashboard)
				adm.GET("/users", rt.adminUsers)
				adm.POST("/users", rt.adminCreateUser)
				adm.PUT("/users/:id", rt.adminUpdateUser)
				adm.DELETE("/users/:id", rt.adminDeleteUser)
				adm.POST("/users/:id/reset-password", rt.adminResetPwd)
				adm.GET("/roles", rt.adminRoles)
				adm.POST("/roles/grant", rt.adminGrantRole)
				adm.GET("/audit", rt.adminAudit)
				adm.GET("/audit/messages", rt.adminMessageAudit)
				adm.GET("/notices", rt.adminNotices)
				adm.POST("/notices", rt.adminCreateNotice)
				adm.PUT("/notices/:id", rt.adminUpdateNotice)
				adm.GET("/sensitive-words", rt.adminSensitiveWords)
				adm.POST("/sensitive-words", rt.adminAddSensitiveWord)
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
		c.Set("uid", claims.UID)
		c.Set("claims", claims)
		c.Next()
	}
}

func uid(c *gin.Context) string { return c.GetString("uid") }

func (rt *Router) WSHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	claims, err := pkg.ParseToken(rt.C.Auth.JWTSecret, token)
	if err != nil || rt.Auth.CheckBlacklist(claims.ID) {
		http.Error(w, "unauthorized", 401)
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
	data, err := rt.Auth.Login(req.Account, req.Password, req.DeviceID, req.Platform)
	if err != nil {
		pkg.Fail(c, 401, 2000, err.Error())
		return
	}
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
	pkg.OK(c, rt.Contact.Users(c.Query("q")))
}
func (rt *Router) users(c *gin.Context) {
	pkg.OK(c, rt.Contact.Users(""))
}
func (rt *Router) userProfile(c *gin.Context) {
	u := rt.Contact.Users(c.Param("id"))
	if len(u) > 0 {
		pkg.OK(c, u[0])
	} else {
		pkg.OK(c, nil)
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
	pkg.OK(c, rt.Contact.Depts())
}
func (rt *Router) deptMembers(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	pkg.OK(c, rt.Contact.DeptMembers(id))
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
	pkg.OK(c, rt.Msg.ListMessages(c.Param("id"), 200))
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
	rt.Hub.PushMessage(me, msg.ConvID, req.ConvType, toUID, req.ClientMsgID,
		msg.MsgID, req.Type, msg.Content, msg.Seq, msg.CreatedAt)
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
