package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexim/server/internal/model"
)

// ============ 通话记录（管理端） ============

type CallQuery struct {
	Keyword string
	Type    string // "" | audio | video
	Result  string // "" | answered | missed | force | active
	Page    int
	Size    int
}

type CallVO struct {
	ID         int64   `json:"id"`
	CallID     string  `json:"callId"`
	ConvID     string  `json:"convId"`
	FromUID    string  `json:"fromUid"`
	FromName   string  `json:"fromName"`
	PeerUID    string  `json:"peerUid"`
	PeerName   string  `json:"peerName"`
	CallType   string  `json:"callType"`
	TypeLabel  string  `json:"typeLabel"`
	Status     int     `json:"status"`
	Result     string  `json:"result"`
	ResultText string  `json:"resultText"`
	Duration   int     `json:"duration"`
	RoomID     string  `json:"roomId"`
	Mode       string  `json:"mode"`
	Members    int     `json:"members"`
	Recorded   int     `json:"recorded"`
	ForceEnded int     `json:"forceEnded"`
	MOS        float64 `json:"mos"`
	PacketLoss float64 `json:"packetLoss"`
	RTT        int     `json:"rtt"`
	Jitter     int     `json:"jitter"`
	Quality    string  `json:"quality"`
	CreatedAt  int64   `json:"createdAt"`
}

func (a *AdminService) callRows(q CallQuery) []CallVO {
	db := a.S.DB.Model(&model.Call{})
	switch q.Type {
	case "audio", "video":
		db = db.Where("call_type = ?", q.Type)
	}
	switch q.Result {
	case "answered":
		db = db.Where("status = 3 AND force_ended = 0")
	case "missed":
		db = db.Where("status = 4")
	case "force":
		db = db.Where("force_ended = 1")
	case "active":
		db = db.Where("status IN (1,2)")
	}
	var calls []model.Call
	db.Order("id desc").Limit(5000).Find(&calls)

	names := map[string]string{}
	var users []model.User
	a.S.DB.Select("uid", "nickname").Find(&users)
	for _, u := range users {
		names[u.UID] = u.Nickname
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]CallVO, 0, len(calls))
	for _, c := range calls {
		fromName := names[c.CallerUID]
		if fromName == "" {
			fromName = c.CallerUID
		}
		peerName := names[c.CalleeUID]
		if peerName == "" {
			peerName = c.CalleeUID
		}
		if kw != "" {
			hay := strings.ToLower(c.CallID + " " + c.RoomID + " " + fromName + " " + peerName)
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		vo := CallVO{
			ID: c.ID, CallID: c.CallID, ConvID: c.ConvID,
			FromUID: c.CallerUID, FromName: fromName, PeerUID: c.CalleeUID, PeerName: peerName,
			CallType: c.CallType, Status: c.Status, Duration: c.Duration,
			RoomID: c.RoomID, Mode: c.Mode, Members: c.Members, Recorded: c.Recorded,
			ForceEnded: c.ForceEnded, MOS: c.MOS, PacketLoss: c.PacketLoss,
			RTT: c.RTT, Jitter: c.Jitter, CreatedAt: c.CreatedAt,
		}
		if vo.RoomID == "" {
			vo.RoomID = c.CallID
		}
		if vo.Mode == "" {
			vo.Mode = "P2P"
		}
		if vo.Members == 0 {
			vo.Members = 2
		}
		if c.CallType == "video" {
			vo.TypeLabel = "视频"
		} else {
			vo.TypeLabel = "语音"
		}
		switch {
		case c.ForceEnded == 1:
			vo.Result, vo.ResultText = "force", "强制挂断"
		case c.Status == 3:
			vo.Result, vo.ResultText = "answered", "已接通"
		case c.Status == 4:
			vo.Result, vo.ResultText = "missed", "未接听"
		default:
			vo.Result, vo.ResultText = "active", "通话中"
		}
		vo.Quality = qualityLabel(c)
		out = append(out, vo)
	}
	return out
}

func qualityLabel(c model.Call) string {
	if c.Status == 4 {
		return "未接通"
	}
	if c.ForceEnded == 1 {
		return "强制挂断"
	}
	if c.MOS > 0 {
		if c.MOS >= 4 {
			return "良好"
		}
		if c.MOS >= 3.5 {
			return "一般"
		}
		return "较差"
	}
	if c.Status == 1 || c.Status == 2 {
		return "通话中"
	}
	return "未采样"
}

// Calls 管理端通话记录：筛选 + 分页
func (a *AdminService) Calls(q CallQuery) map[string]interface{} {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 || q.Size > 200 {
		q.Size = 20
	}
	all := a.callRows(q)
	total := len(all)
	start := (q.Page - 1) * q.Size
	if start > total {
		start = total
	}
	end := start + q.Size
	if end > total {
		end = total
	}
	return map[string]interface{}{"total": total, "list": all[start:end], "page": q.Page, "size": q.Size}
}

// ExportCalls 导出全部筛选结果
func (a *AdminService) ExportCalls(q CallQuery) []CallVO {
	q.Page, q.Size = 1, 100000
	return a.callRows(q)
}

// ============ RTC 质量监控 ============

// RTCQuality 近 24 小时通话质量聚合
func (a *AdminService) RTCQuality() map[string]interface{} {
	since := model.Now() - 24*3600*1000
	var calls []model.Call
	a.S.DB.Where("created_at >= ?", since).Order("id desc").Find(&calls)

	total := len(calls)
	answered, missed, active, force, sampled := 0, 0, 0, 0, 0
	durationSum, mosSum, lossSum, rttSum, jitterSum := 0, 0.0, 0.0, 0, 0
	buckets := []int{0, 0, 0, 0, 0} // 5-4.5, 4.5-4, 4-3.5, 3.5-3, <3
	for _, c := range calls {
		switch {
		case c.ForceEnded == 1:
			force++
		case c.Status == 3:
			answered++
		case c.Status == 4:
			missed++
		default:
			active++
		}
		durationSum += c.Duration
		if c.MOS > 0 {
			sampled++
			mosSum += c.MOS
			lossSum += c.PacketLoss
			rttSum += c.RTT
			jitterSum += c.Jitter
			switch {
			case c.MOS >= 4.5:
				buckets[0]++
			case c.MOS >= 4.0:
				buckets[1]++
			case c.MOS >= 3.5:
				buckets[2]++
			case c.MOS >= 3.0:
				buckets[3]++
			default:
				buckets[4]++
			}
		}
	}
	avgDuration, answerRate, avgMOS, avgLoss, avgRTT, avgJitter := 0, 0.0, 0.0, 0.0, 0, 0
	if total > 0 {
		avgDuration = durationSum / total
		if total-missed > 0 {
			answerRate = float64(answered) * 100 / float64(total-missed)
		}
	}
	if sampled > 0 {
		avgMOS = mosSum / float64(sampled)
		avgLoss = lossSum / float64(sampled)
		avgRTT = rttSum / sampled
		avgJitter = jitterSum / sampled
	}

	// 近 12 小时通话量趋势
	now := time.Now()
	trend := make([]map[string]interface{}, 0, 12)
	for i := 11; i >= 0; i-- {
		start := now.Add(-time.Duration(i) * time.Hour).Truncate(time.Hour).UnixMilli()
		end := start + 3600*1000
		n := 0
		for _, c := range calls {
			if c.CreatedAt >= start && c.CreatedAt < end {
				n++
			}
		}
		trend = append(trend, map[string]interface{}{
			"label": time.UnixMilli(start).Format("15:04"), "value": n,
		})
	}

	// 告警：质量差 / 强制挂断
	alarms := []map[string]interface{}{}
	for _, c := range calls {
		level, typ, detail := "", "", ""
		switch {
		case c.MOS > 0 && c.MOS < 3.5:
			level, typ = "high", "MOS 过低"
			detail = fmt.Sprintf("MOS %.2f，丢包 %.1f%%，RTT %dms", c.MOS, c.PacketLoss, c.RTT)
		case c.PacketLoss >= 5:
			level, typ = "mid", "丢包率过高"
			detail = fmt.Sprintf("丢包 %.1f%%，RTT %dms", c.PacketLoss, c.RTT)
		case c.ForceEnded == 1:
			level, typ = "low", "强制挂断"
			detail = "管理员强制结束了该通话"
		}
		if typ == "" {
			continue
		}
		alarms = append(alarms, map[string]interface{}{
			"callId": c.CallID, "level": level, "type": typ, "detail": detail,
			"createdAt": c.CreatedAt,
		})
		if len(alarms) >= 20 {
			break
		}
	}

	return map[string]interface{}{
		"total": total, "active": active, "answered": answered, "missed": missed,
		"forceEnded": force, "avgDuration": avgDuration, "answerRate": answerRate,
		"sampled": sampled, "avgMos": avgMOS, "avgLoss": avgLoss,
		"avgRtt": avgRTT, "avgJitter": avgJitter,
		"mosBuckets": []map[string]interface{}{
			{"label": "5.0-4.5", "value": buckets[0]},
			{"label": "4.5-4.0", "value": buckets[1]},
			{"label": "4.0-3.5", "value": buckets[2]},
			{"label": "3.5-3.0", "value": buckets[3]},
			{"label": "<3.0", "value": buckets[4]},
		},
		"trend": trend, "alarms": alarms,
	}
}

// ============ 群组（管理端） ============

type GroupVO struct {
	model.Group
	OwnerName string `json:"ownerName"`
	MsgCount  int64  `json:"msgCount"`
}

func (a *AdminService) AdminGroups(keyword, status string) []GroupVO {
	db := a.S.DB.Model(&model.Group{})
	if status == "1" || status == "2" || status == "3" {
		db = db.Where("status = ?", status)
	}
	var groups []model.Group
	db.Order("id desc").Find(&groups)

	names := map[string]string{}
	var users []model.User
	a.S.DB.Select("uid", "nickname").Find(&users)
	for _, u := range users {
		names[u.UID] = u.Nickname
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	out := make([]GroupVO, 0, len(groups))
	for _, g := range groups {
		owner := names[g.OwnerUID]
		if owner == "" {
			owner = g.OwnerUID
		}
		if kw != "" && !strings.Contains(strings.ToLower(g.Name+" "+g.GroupID+" "+owner), kw) {
			continue
		}
		var maxSeq int64
		a.S.DB.Model(&model.Conversation{}).Where("conv_id = ?", "conv_"+g.GroupID).
			Select("COALESCE(MAX(max_seq),0)").Scan(&maxSeq)
		out = append(out, GroupVO{Group: g, OwnerName: owner, MsgCount: maxSeq})
	}
	return out
}

// SetGroupStatus 冻结（2）/ 恢复（1）
func (a *AdminService) SetGroupStatus(groupID string, status int) error {
	if status != 1 && status != 2 {
		return fmt.Errorf("状态值非法")
	}
	return a.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("status", status).Error
}

// AdminDissolveGroup 管理端解散群（status=3，历史成员与消息保留供审计）
func (a *AdminService) AdminDissolveGroup(groupID string) error {
	var g model.Group
	if err := a.S.DB.Where("group_id = ?", groupID).First(&g).Error; err != nil {
		return fmt.Errorf("群不存在")
	}
	return a.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("status", 3).Error
}

// ============ 系统设置 ============

var defaultSettings = map[string]string{
	"retentionDays": "180", "fileLimitMB": "100", "groupMemberLimit": "2000",
	"revokeWindowMin": "2", "heartbeatSec": "30", "signUrlMinutes": "15",
	"mfaRequired": "true", "watermark": "true", "ipWhitelist": "",
	"notifyChannels": "站内信,邮件",
}

func (a *AdminService) GetSettings() map[string]string {
	out := map[string]string{}
	for k, v := range defaultSettings {
		out[k] = v
	}
	var rows []model.SystemSetting
	a.S.DB.Find(&rows)
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out
}

func (a *AdminService) SaveSettings(values map[string]interface{}) map[string]string {
	for k, v := range values {
		if k == "" {
			continue
		}
		val := strings.TrimSpace(fmt.Sprint(v))
		var row model.SystemSetting
		// key 是 MySQL 保留字，必须显式加反引号
		if a.S.DB.Where("`key` = ?", k).First(&row).Error == nil {
			a.S.DB.Model(&row).Updates(map[string]interface{}{"value": val, "updated_at": model.Now()})
		} else {
			a.S.DB.Create(&model.SystemSetting{Key: k, Value: val, UpdatedAt: model.Now()})
		}
	}
	return a.GetSettings()
}

// ============ 登录日志 ============

type LoginLogQuery struct {
	Keyword string
	Result  string
	Risk    string
	Page    int
	Size    int
}

func (a *AdminService) loginLogRows(q LoginLogQuery) []model.LoginLog {
	db := a.S.DB.Model(&model.LoginLog{})
	if q.Result == "success" || q.Result == "fail" {
		db = db.Where("result = ?", q.Result)
	}
	if q.Risk != "" {
		db = db.Where("risk = ?", q.Risk)
	}
	kw := strings.TrimSpace(q.Keyword)
	if kw != "" {
		like := "%" + kw + "%"
		db = db.Where("(account LIKE ? OR ip LIKE ? OR device_id LIKE ?)", like, like, like)
	}
	var rows []model.LoginLog
	db.Order("id desc").Limit(5000).Find(&rows)
	return rows
}

func (a *AdminService) LoginLogs(q LoginLogQuery) map[string]interface{} {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 || q.Size > 200 {
		q.Size = 20
	}
	all := a.loginLogRows(q)
	total := len(all)
	start := (q.Page - 1) * q.Size
	if start > total {
		start = total
	}
	end := start + q.Size
	if end > total {
		end = total
	}
	return map[string]interface{}{"total": total, "list": all[start:end], "page": q.Page, "size": q.Size}
}

func (a *AdminService) ExportLoginLogs(q LoginLogQuery) []model.LoginLog {
	return a.loginLogRows(q)
}

// RecordLogin 写入登录日志，并按近 10 分钟失败次数标记风险
func (a *AdminService) RecordLogin(uid, account, ip, platform, deviceID string, ok bool) {
	result := "success"
	if !ok {
		result = "fail"
	}
	risk := ""
	var fails int64
	a.S.DB.Model(&model.LoginLog{}).
		Where("ip = ? AND result = 'fail' AND created_at >= ?", ip, model.Now()-10*60*1000).
		Count(&fails)
	if fails >= 5 {
		risk = "异常登录"
	}
	a.S.DB.Create(&model.LoginLog{
		UID: uid, Account: account, IP: ip, Platform: platform, DeviceID: deviceID,
		Result: result, Risk: risk, CreatedAt: model.Now(),
	})
}

// BanIP 封禁来源 IP 指定分钟数（登录入口即时生效）
func (a *AdminService) BanIP(ip string, minutes int) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return fmt.Errorf("IP 不能为空")
	}
	if minutes <= 0 {
		minutes = 30
	}
	return a.S.RDB.Set(a.S.Ctx, "ip:ban:"+ip, "1", time.Duration(minutes)*time.Minute).Err()
}

func (a *AdminService) IsIPBanned(ip string) bool {
	if ip == "" {
		return false
	}
	n, _ := a.S.RDB.Exists(a.S.Ctx, "ip:ban:"+ip).Result()
	return n > 0
}

// ============ 敏感词 ============

func (a *AdminService) DeleteSensitiveWord(word string) error {
	return a.S.RDB.SRem(a.S.Ctx, "sensitive:words:v1", word).Err()
}

func (a *AdminService) BulkSensitiveWords(words []string) error {
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			if err := a.S.RDB.SAdd(a.S.Ctx, "sensitive:words:v1", w).Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *AdminService) SensitiveStrategy() string {
	v, err := a.S.RDB.Get(a.S.Ctx, "sensitive:strategy").Result()
	if err != nil || (v != "block" && v != "mask" && v != "alert") {
		return "block"
	}
	return v
}

func (a *AdminService) SetSensitiveStrategy(s string) error {
	if s != "block" && s != "mask" && s != "alert" {
		return fmt.Errorf("策略值非法")
	}
	return a.S.RDB.Set(a.S.Ctx, "sensitive:strategy", s, 0).Err()
}

func (a *AdminService) SensitiveHits(page, size int) map[string]interface{} {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	var total int64
	a.S.DB.Model(&model.SensitiveHit{}).Count(&total)
	var list []model.SensitiveHit
	a.S.DB.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list)
	return map[string]interface{}{"total": total, "list": list}
}

func (a *AdminService) ClearSensitiveHits() error {
	return a.S.DB.Where("1 = 1").Delete(&model.SensitiveHit{}).Error
}

// ============ 审计导出 ============

// AuditQuery 操作日志筛选条件
type AuditQuery struct {
	Module  string
	Keyword string
}

// AuditLogs 操作日志分页（支持模块前缀与关键词）
func (a *AdminService) AuditLogs(q AuditQuery, page, size int) map[string]interface{} {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	db := a.S.DB.Model(&model.AuditLog{})
	if q.Module != "" {
		db = db.Where("action LIKE ?", q.Module+"%")
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("(uid LIKE ? OR action LIKE ? OR target LIKE ? OR detail LIKE ?)", like, like, like, like)
	}
	var total int64
	db.Count(&total)
	var list []model.AuditLog
	db.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list)
	return map[string]interface{}{"total": total, "list": list}
}

// MessageAuditFiltered 消息审计分页（关键词/类型/状态）
func (a *AdminService) MessageAuditFiltered(kw string, msgType, status, page, size int) map[string]interface{} {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	all, _ := a.S.Msg.Page(0, 3000)
	kw = strings.ToLower(strings.TrimSpace(kw))
	out := make([]model.Message, 0, len(all))
	for _, m := range all {
		if msgType > 0 && m.Type != msgType {
			continue
		}
		if status > 0 && m.Status != status {
			continue
		}
		if kw != "" {
			hay := strings.ToLower(m.MsgID + " " + m.ConvID + " " + m.FromUID + " " + m.ToID + " " + m.Content)
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		out = append(out, m)
	}
	total := len(out)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return map[string]interface{}{"total": total, "list": out[start:end]}
}

func (a *AdminService) ExportAudit(q AuditQuery) []model.AuditLog {
	db := a.S.DB.Model(&model.AuditLog{})
	if q.Module != "" {
		db = db.Where("action LIKE ?", q.Module+"%")
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("(uid LIKE ? OR action LIKE ? OR target LIKE ? OR detail LIKE ?)", like, like, like, like)
	}
	var rows []model.AuditLog
	db.Order("id desc").Limit(10000).Find(&rows)
	return rows
}

// ExportMessages 消息审计导出（最多 5000 条）
func (a *AdminService) ExportMessages() []model.Message {
	list, err := a.S.Msg.Page(0, 5000)
	if err != nil {
		return []model.Message{}
	}
	return list
}

// ExportMessagesFiltered 按条件导出消息审计（最多 5000 条）
func (a *AdminService) ExportMessagesFiltered(kw string, msgType, status int) []model.Message {
	all, _ := a.S.Msg.Page(0, 5000)
	kw = strings.ToLower(strings.TrimSpace(kw))
	out := make([]model.Message, 0, len(all))
	for _, m := range all {
		if msgType > 0 && m.Type != msgType {
			continue
		}
		if status > 0 && m.Status != status {
			continue
		}
		if kw != "" {
			hay := strings.ToLower(m.MsgID + " " + m.ConvID + " " + m.FromUID + " " + m.ToID + " " + m.Content)
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

func (a *AdminService) ExportGroups(keyword, status string) []GroupVO {
	rows := a.AdminGroups(keyword, status)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	return rows
}

// CallStatusText 供导出复用的状态文案
func CallStatusText(v CallVO) string {
	if v.Status == 3 && v.ForceEnded == 0 {
		return "已接通"
	}
	if v.Status == 4 {
		return "未接听"
	}
	if v.ForceEnded == 1 {
		return "强制挂断"
	}
	return "通话中"
}

// DurationText 秒转 mm:ss
func DurationText(sec int) string {
	if sec <= 0 {
		return "00:00"
	}
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

// FormatTimestamp 毫秒时间戳转本地时间
func FormatTimestamp(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
}

// ParseIntDefault 便捷解析
func ParseIntDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
