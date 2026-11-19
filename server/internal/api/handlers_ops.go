package api

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"nexim/server/internal/gateway"
	"nexim/server/internal/model"
	"nexim/server/internal/pkg"
	"nexim/server/internal/service"

	"github.com/gin-gonic/gin"
)

// ============ 会话 / 收藏 / 黑名单 ============

func (rt *Router) conversationDelete(c *gin.Context) {
	if err := rt.Msg.DeleteConversation(uid(c), c.Param("id")); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, nil)
}

func (rt *Router) favorites(c *gin.Context) {
	pkg.OK(c, rt.Fav.List(uid(c)))
}

func (rt *Router) favoriteAdd(c *gin.Context) {
	var req struct {
		MsgID string `json:"msgId" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	fav, err := rt.Fav.Add(uid(c), req.MsgID)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	pkg.OK(c, fav)
}

func (rt *Router) favoriteDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := rt.Fav.Remove(uid(c), id); err != nil {
		pkg.Fail(c, 404, 4000, err.Error())
		return
	}
	pkg.OK(c, nil)
}

func (rt *Router) blacklist(c *gin.Context) {
	pkg.OK(c, rt.Blacklist.List(uid(c)))
}

func (rt *Router) blacklistAdd(c *gin.Context) {
	var req struct {
		UID string `json:"uid" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if err := rt.Blacklist.Add(uid(c), req.UID); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, nil)
}

func (rt *Router) blacklistRemove(c *gin.Context) {
	if err := rt.Blacklist.Remove(uid(c), c.Param("uid")); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, nil)
}

// ============ 通话质量上报 ============

func (rt *Router) callQualityReport(c *gin.Context) {
	var req struct {
		MOS        float64 `json:"mos"`
		PacketLoss float64 `json:"packetLoss"`
		RTT        int     `json:"rtt"`
		Jitter     int     `json:"jitter"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	var call model.Call
	if rt.S.DB.Where("call_id = ?", c.Param("id")).First(&call).Error != nil {
		pkg.Fail(c, 404, 4000, "通话不存在")
		return
	}
	me := uid(c)
	if call.CallerUID != me && call.CalleeUID != me {
		pkg.Fail(c, 403, 3001, "无权上报该通话")
		return
	}
	if req.MOS < 1 || req.MOS > 5 {
		req.MOS = 0
	}
	_ = rt.Call.ReportQuality(c.Param("id"), req.MOS, req.PacketLoss, req.RTT, req.Jitter)
	pkg.OK(c, nil)
}

// ============ 管理端：通话记录 ============

func callQueryFrom(c *gin.Context) service.CallQuery {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	return service.CallQuery{
		Keyword: c.Query("kw"), Type: c.Query("type"), Result: c.Query("result"),
		Page: page, Size: size,
	}
}

func (rt *Router) adminCalls(c *gin.Context) {
	pkg.OK(c, rt.Admin.Calls(callQueryFrom(c)))
}

func (rt *Router) adminCallsExport(c *gin.Context) {
	rows := rt.Admin.ExportCalls(callQueryFrom(c))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="nexim-calls.csv"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"通话ID", "房间", "发起人", "接收人", "类型", "模式", "人数", "结果", "时长(秒)", "质量", "MOS", "丢包(%)", "RTT(ms)", "录制", "开始时间"})
	for _, v := range rows {
		rec := "否"
		if v.Recorded == 1 {
			rec = "是"
		}
		_ = w.Write([]string{
			v.CallID, v.RoomID, v.FromName, v.PeerName, v.TypeLabel, v.Mode,
			strconv.Itoa(v.Members), service.CallStatusText(v), strconv.Itoa(v.Duration),
			v.Quality, strconv.FormatFloat(v.MOS, 'f', 2, 64),
			strconv.FormatFloat(v.PacketLoss, 'f', 1, 64), strconv.Itoa(v.RTT), rec,
			service.FormatTimestamp(v.CreatedAt),
		})
	}
	w.Flush()
	rt.Admin.Audit(uid(c), "rtc:record:export", "", strconv.Itoa(len(rows)))
}

func (rt *Router) adminEndCall(c *gin.Context) {
	callID := c.Param("id")
	if err := rt.Call.ForceEnd(callID); err != nil {
		pkg.Fail(c, 404, 4000, "通话不存在")
		return
	}
	var call model.Call
	_ = rt.S.DB.Where("call_id = ?", callID).First(&call).Error
	payload, _ := json.Marshal(map[string]interface{}{
		"callId": callID, "reason": "管理员强制挂断",
	})
	for _, target := range []string{call.CallerUID, call.CalleeUID} {
		if target != "" {
			rt.Hub.PushToUser(target, &gateway.Frame{
				Cmd: "CALL_FORCE_END_PUSH", Payload: payload, ServerTime: model.Now(),
			})
		}
	}
	rt.Admin.Audit(uid(c), "rtc:record:manage", callID, "force-end")
	pkg.OK(c, nil)
}

// ============ 管理端：RTC 质量 ============

func (rt *Router) adminRTCQuality(c *gin.Context) {
	pkg.OK(c, rt.Admin.RTCQuality())
}

// ============ 管理端：群组 ============

func (rt *Router) adminGroups(c *gin.Context) {
	list := rt.Admin.AdminGroups(c.Query("kw"), c.Query("status"))
	pkg.OK(c, list)
}

func (rt *Router) adminGroupsExport(c *gin.Context) {
	rows := rt.Admin.ExportGroups(c.Query("kw"), c.Query("status"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="nexim-groups.csv"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"群ID", "群名称", "群主", "成员数", "消息量", "状态", "创建时间"})
	for _, g := range rows {
		state := "正常"
		switch g.Status {
		case 2:
			state = "已冻结"
		case 3:
			state = "已解散"
		}
		_ = w.Write([]string{g.GroupID, g.Name, g.OwnerName, strconv.Itoa(g.MemberCount),
			strconv.FormatInt(g.MsgCount, 10), state, service.FormatTimestamp(g.CreatedAt)})
	}
	w.Flush()
	rt.Admin.Audit(uid(c), "group:export", "", strconv.Itoa(len(rows)))
}

func (rt *Router) adminGroupStatus(c *gin.Context) {
	var req struct {
		Status int `json:"status"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if err := rt.Admin.SetGroupStatus(c.Param("id"), req.Status); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	action := "group:freeze"
	if req.Status == 1 {
		action = "group:unfreeze"
	}
	rt.Admin.Audit(uid(c), action, c.Param("id"), "")
	pkg.OK(c, nil)
}

func (rt *Router) adminGroupDissolve(c *gin.Context) {
	if err := rt.Admin.AdminDissolveGroup(c.Param("id")); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "group:dissolve", c.Param("id"), "")
	pkg.OK(c, nil)
}

// ============ 管理端：系统设置 ============

func (rt *Router) adminGetSettings(c *gin.Context) {
	pkg.OK(c, rt.Admin.GetSettings())
}

func (rt *Router) adminSaveSettings(c *gin.Context) {
	var req map[string]interface{}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if inner, ok := req["settings"].(map[string]interface{}); ok {
		req = inner
	}
	saved := rt.Admin.SaveSettings(req)
	rt.Admin.Audit(uid(c), "setting:config", "", "")
	pkg.OK(c, saved)
}

// ============ 管理端：登录日志 ============

func loginLogQueryFrom(c *gin.Context) service.LoginLogQuery {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	return service.LoginLogQuery{
		Keyword: c.Query("kw"), Result: c.Query("result"), Risk: c.Query("risk"),
		Page: page, Size: size,
	}
}

func (rt *Router) adminLoginLogs(c *gin.Context) {
	pkg.OK(c, rt.Admin.LoginLogs(loginLogQueryFrom(c)))
}

func (rt *Router) adminLoginLogsExport(c *gin.Context) {
	rows := rt.Admin.ExportLoginLogs(loginLogQueryFrom(c))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="nexim-login-logs.csv"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"账号", "IP", "平台", "设备", "结果", "风险", "时间"})
	for _, r := range rows {
		result := "成功"
		if r.Result == "fail" {
			result = "失败"
		}
		risk := r.Risk
		if risk == "" {
			risk = "—"
		}
		_ = w.Write([]string{r.Account, r.IP, r.Platform, r.DeviceID, result, risk, service.FormatTimestamp(r.CreatedAt)})
	}
	w.Flush()
	rt.Admin.Audit(uid(c), "log:login:export", "", strconv.Itoa(len(rows)))
}

func (rt *Router) adminBanIP(c *gin.Context) {
	var req struct {
		IP      string `json:"ip" binding:"required"`
		Minutes int    `json:"minutes"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if err := rt.Admin.BanIP(req.IP, req.Minutes); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "log:login:ban", req.IP, strconv.Itoa(req.Minutes))
	pkg.OK(c, nil)
}

// ============ 管理端：敏感词 ============

func (rt *Router) adminDeleteSensitiveWord(c *gin.Context) {
	word := c.Param("word")
	if err := rt.Admin.DeleteSensitiveWord(word); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "sensitive:delete", word, "")
	pkg.OK(c, nil)
}

func (rt *Router) adminBulkSensitiveWords(c *gin.Context) {
	var req struct {
		Text  string   `json:"text"`
		Words []string `json:"words"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	words := req.Words
	if len(words) == 0 && req.Text != "" {
		for _, line := range strings.FieldsFunc(req.Text, func(r rune) bool {
			return r == '\n' || r == '\r' || r == ',' || r == '，' || r == ';' || r == '；'
		}) {
			words = append(words, line)
		}
	}
	if err := rt.Admin.BulkSensitiveWords(words); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "sensitive:import", "", strconv.Itoa(len(words)))
	pkg.OK(c, gin.H{"imported": len(words)})
}

func (rt *Router) adminSensitiveStrategy(c *gin.Context) {
	pkg.OK(c, gin.H{"strategy": rt.Admin.SensitiveStrategy()})
}

func (rt *Router) adminSetSensitiveStrategy(c *gin.Context) {
	var req struct {
		Strategy string `json:"strategy" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if err := rt.Admin.SetSensitiveStrategy(req.Strategy); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "sensitive:strategy", req.Strategy, "")
	pkg.OK(c, nil)
}

func (rt *Router) adminSensitiveHits(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	pkg.OK(c, rt.Admin.SensitiveHits(page, size))
}

func (rt *Router) adminClearSensitiveHits(c *gin.Context) {
	if err := rt.Admin.ClearSensitiveHits(); err != nil {
		pkg.Fail(c, 500, 9000, err.Error())
		return
	}
	pkg.OK(c, nil)
}

// ============ 管理端：审计导出 ============

func auditQueryFrom(c *gin.Context) service.AuditQuery {
	return service.AuditQuery{Module: c.Query("module"), Keyword: c.Query("kw")}
}

func (rt *Router) adminAuditExport(c *gin.Context) {
	rows := rt.Admin.ExportAudit(auditQueryFrom(c))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="nexim-audit-logs.csv"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"时间", "操作人", "动作", "对象", "详情"})
	for _, r := range rows {
		_ = w.Write([]string{service.FormatTimestamp(r.CreatedAt), r.UID, r.Action, r.Target, r.Detail})
	}
	w.Flush()
	rt.Admin.Audit(uid(c), "log:export", "", strconv.Itoa(len(rows)))
}

func (rt *Router) adminMessagesExport(c *gin.Context) {
	mt, _ := strconv.Atoi(c.Query("type"))
	st, _ := strconv.Atoi(c.Query("status"))
	rows := rt.Admin.ExportMessagesFiltered(c.Query("kw"), mt, st)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="nexim-message-audit.csv"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"时间", "消息ID", "会话", "发送人", "接收方", "类型", "内容", "状态"})
	for _, m := range rows {
		_ = w.Write([]string{
			service.FormatTimestamp(m.CreatedAt), m.MsgID, m.ConvID, m.FromUID, m.ToID,
			msgTypeName(m.Type), m.Content, msgStatusName(m.Status),
		})
	}
	w.Flush()
	rt.Admin.Audit(uid(c), "audit:message:export", "", strconv.Itoa(len(rows)))
}

func msgTypeName(t int) string {
	switch t {
	case 2:
		return "图片"
	case 3:
		return "系统"
	case 4:
		return "语音"
	case 5:
		return "文件"
	case 6:
		return "视频"
	default:
		return "文本"
	}
}

func msgStatusName(s int) string {
	switch s {
	case 2:
		return "已撤回"
	case 3:
		return "已删除"
	default:
		return "正常"
	}
}
