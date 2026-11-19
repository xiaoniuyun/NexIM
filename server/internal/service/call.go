package service

import (
	"nexim/server/internal/model"
	"nexim/server/internal/store"
)

// CallService 1对1音视频通话记录（信令链路本身走 WebSocket / JetStream 转发）
type CallService struct{ S *store.Store }

func NewCall(s *store.Store) *CallService { return &CallService{S: s} }

// Start 呼叫发起时建记录
func (c *CallService) Start(callID, convID, callerUID, calleeUID, callType string) {
	if callID == "" {
		return
	}
	var call model.Call
	if err := c.S.DB.Where("call_id = ?", callID).First(&call).Error; err == nil {
		return
	}
	c.S.DB.Create(&model.Call{
		CallID: callID, ConvID: convID, CallerUID: callerUID, CalleeUID: calleeUID,
		CallType: callType, Status: 1, CreatedAt: model.Now(),
		RoomID: callID, Mode: "P2P", Members: 2,
	})
}

// Answer 被叫应答：接听 -> 通话中；拒绝 -> 拒接
func (c *CallService) Answer(callID string, accepted bool) {
	if callID == "" {
		return
	}
	now := model.Now()
	if accepted {
		c.S.DB.Model(&model.Call{}).Where("call_id = ? AND status = 1", callID).
			Updates(map[string]interface{}{"status": 2, "answered_at": now})
		return
	}
	c.S.DB.Model(&model.Call{}).Where("call_id = ? AND status IN (1,2)", callID).
		Updates(map[string]interface{}{"status": 4, "ended_at": now})
}

// End 结束通话并计算通话时长（秒）
func (c *CallService) End(callID string) {
	if callID == "" {
		return
	}
	var call model.Call
	if c.S.DB.Where("call_id = ?", callID).First(&call).Error != nil {
		return
	}
	if call.Status == 3 || call.Status == 4 {
		return
	}
	now := model.Now()
	duration := 0
	if call.AnsweredAt > 0 {
		duration = int((now - call.AnsweredAt) / 1000)
	}
	c.S.DB.Model(&model.Call{}).Where("call_id = ?", callID).
		Updates(map[string]interface{}{"status": 3, "ended_at": now, "duration": duration})
}

// ForceEnd 管理端强制挂断：标记强制结束并计算时长
func (c *CallService) ForceEnd(callID string) error {
	var call model.Call
	if err := c.S.DB.Where("call_id = ?", callID).First(&call).Error; err != nil {
		return err
	}
	now := model.Now()
	duration := call.Duration
	if call.AnsweredAt > 0 && call.Status != 3 && call.Status != 4 {
		duration = int((now - call.AnsweredAt) / 1000)
	}
	return c.S.DB.Model(&model.Call{}).Where("call_id = ?", callID).
		Updates(map[string]interface{}{
			"status": 3, "ended_at": now, "duration": duration, "force_ended": 1,
		}).Error
}

// ReportQuality 保存客户端上报的通话质量采样
func (c *CallService) ReportQuality(callID string, mos, loss float64, rtt, jitter int) error {
	return c.S.DB.Model(&model.Call{}).Where("call_id = ?", callID).
		Updates(map[string]interface{}{
			"mos": mos, "packet_loss": loss, "rtt": rtt, "jitter": jitter,
		}).Error
}

// Records 当前用户的通话记录（主叫或被叫），按时间倒序
func (c *CallService) Records(uid string, limit int) []map[string]interface{} {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var calls []model.Call
	c.S.DB.Where("caller_uid = ? OR callee_uid = ?", uid, uid).
		Order("id desc").Limit(limit).Find(&calls)
	out := make([]map[string]interface{}, 0, len(calls))
	for _, call := range calls {
		peer := call.CalleeUID
		outgoing := call.CallerUID == uid
		if !outgoing {
			peer = call.CallerUID
		}
		name := peer
		var u model.User
		if c.S.DB.Where("uid = ?", peer).First(&u).Error == nil && u.Nickname != "" {
			name = u.Nickname
		}
		out = append(out, map[string]interface{}{
			"callId": call.CallID, "convId": call.ConvID, "peerId": peer, "peerName": name,
			"callType": call.CallType, "status": call.Status, "outgoing": outgoing,
			"createdAt": call.CreatedAt, "answeredAt": call.AnsweredAt,
			"endedAt": call.EndedAt, "duration": call.Duration,
		})
	}
	return out
}
