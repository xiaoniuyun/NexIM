package gateway

import (
	"encoding/json"
	"fmt"

	"nexim/server/internal/model"
	"nexim/server/internal/mq"

	"go.uber.org/zap"
)

type SendReq struct {
	ClientMsgID string `json:"clientMsgId"`
	ConvID      string `json:"convId"`
	ConvType    string `json:"convType"`
	ToUID       string `json:"toUid"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Extra       string `json:"extra"`
}

type SendAck struct {
	ClientMsgID string `json:"clientMsgId"`
	MsgID       string `json:"msgId"`
	Seq         int64  `json:"seq"`
	SendTime    int64  `json:"sendTime"`
}

type PushPayload struct {
	MsgID       string `json:"msgId"`
	ClientMsgID string `json:"clientMsgId"`
	ConvID      string `json:"convId"`
	ConvType    string `json:"convType"`
	Seq         int64  `json:"seq"`
	FromUID     string `json:"fromUid"`
	FromName    string `json:"fromName"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Extra       string `json:"extra,omitempty"`
	SendTime    int64  `json:"sendTime"`
}

func (h *Hub) handle(c *Client, f *Frame) {
	switch f.Cmd {
	case "HEARTBEAT":
		c.send <- &Frame{Cmd: "HEARTBEAT_ACK", SeqNo: f.SeqNo, ServerTime: model.Now()}
	case "AUTH":
		// token 已在 HTTP 握手阶段校验，此处确认注册
		c.send <- &Frame{Cmd: "AUTH_ACK", SeqNo: f.SeqNo, Code: 0, Payload: mustJSON(map[string]interface{}{
			"uid": c.UID, "deviceId": c.DeviceID, "serverTime": model.Now(),
		})}
	case "SEND":
		var req SendReq
		if len(f.Payload) == 0 || json.Unmarshal(f.Payload, &req) != nil {
			c.send <- &Frame{Cmd: "SEND_ACK", SeqNo: f.SeqNo, Code: 1001, Message: "帧格式错误"}
			return
		}
		// 客户端未携带 toUid（例如会话记录里 peerId 缺失）时从会话表反查对端，避免推送目标丢失
		toUID := req.ToUID
		if toUID == "" {
			toUID = h.Msg.PeerOf(req.ConvID, c.UID)
		}
		msg, err := h.Msg.Send(c.UID, req.ConvID, req.ConvType, toUID, req.Type, req.Content, req.Extra, req.ClientMsgID)
		if err != nil {
			h.Log.Warn("send message failed", zap.Error(err))
			c.send <- &Frame{Cmd: "SEND_ACK", SeqNo: f.SeqNo, Code: 4000, Message: err.Error()}
			return
		}
		// Send 内部可能从 convId 补齐对端，推送以落库后的 ToID 为准
		if msg.ToID != "" {
			toUID = msg.ToID
		}
		c.send <- &Frame{Cmd: "SEND_ACK", SeqNo: f.SeqNo, Code: 0, Payload: mustJSON(SendAck{
			ClientMsgID: req.ClientMsgID, MsgID: msg.MsgID, Seq: msg.Seq, SendTime: msg.CreatedAt,
		})}
		h.PushMessage(c.UID, msg.ConvID, req.ConvType, toUID, req.ClientMsgID,
			msg.MsgID, req.Type, msg.Content, msg.Extra, msg.Seq, msg.CreatedAt)
	case "PUSH_ACK":
		// 送达确认，记录即可
	case "SYNC":
		var req struct {
			SinceSeq int64  `json:"sinceSeq"`
			ConvID   string `json:"convId"`
		}
		if json.Unmarshal(f.Payload, &req) != nil {
			return
		}
		msgs := h.Msg.ListMessages(c.UID, req.ConvID, 200)
		c.send <- &Frame{Cmd: "SYNC_ACK", SeqNo: f.SeqNo, Code: 0, Payload: mustJSON(map[string]interface{}{
			"convId": req.ConvID, "messages": msgs,
		})}
	case "READ":
		var req struct {
			ConvID string `json:"convId"`
		}
		if json.Unmarshal(f.Payload, &req) != nil {
			return
		}
		_ = h.Msg.MarkRead(c.UID, req.ConvID)
		c.send <- &Frame{Cmd: "READ_ACK", SeqNo: f.SeqNo, Code: 0}
		h.NotifyRead(c.UID, req.ConvID)
	case "REVOKE":
		var req struct {
			MsgID string `json:"msgId"`
		}
		if json.Unmarshal(f.Payload, &req) != nil {
			return
		}
		if err := h.Msg.Revoke(c.UID, req.MsgID); err != nil {
			c.send <- &Frame{Cmd: "REVOKE_ACK", SeqNo: f.SeqNo, Code: 4006, Message: err.Error()}
			return
		}
		c.send <- &Frame{Cmd: "REVOKE_ACK", SeqNo: f.SeqNo, Code: 0}
		h.BroadcastRevoke(req.MsgID)
	case "CALL_INVITE":
		var req struct {
			CallID   string `json:"callId"`
			ConvID   string `json:"convId"`
			ToUID    string `json:"toUid"`
			CallType string `json:"callType"`
			SDP      string `json:"sdp"`
		}
		if json.Unmarshal(f.Payload, &req) != nil || req.CallID == "" || req.ToUID == "" || req.ToUID == c.UID {
			c.send <- &Frame{Cmd: "CALL_ACK", SeqNo: f.SeqNo, Code: 1001, Message: "呼叫参数错误"}
			return
		}
		if !h.IsOnline(req.ToUID) {
			c.send <- &Frame{Cmd: "CALL_ACK", SeqNo: f.SeqNo, Code: 4041, Message: "对方不在线"}
			return
		}
		if h.Calls != nil {
			h.Calls.Start(req.CallID, req.ConvID, c.UID, req.ToUID, req.CallType)
		}
		h.Signal(req.ToUID, "CALL_INVITE_PUSH", map[string]interface{}{
			"callId": req.CallID, "convId": req.ConvID, "callType": req.CallType,
			"fromUid": c.UID, "fromName": h.DisplayName(c.UID), "sdp": req.SDP,
		})
		c.send <- &Frame{Cmd: "CALL_ACK", SeqNo: f.SeqNo, Code: 0, Payload: mustJSON(map[string]interface{}{
			"callId": req.CallID,
		})}
	case "CALL_ANSWER":
		var req struct {
			CallID string `json:"callId"`
			ToUID  string `json:"toUid"`
			Accept bool   `json:"accept"`
			SDP    string `json:"sdp"`
		}
		if json.Unmarshal(f.Payload, &req) != nil || req.CallID == "" || req.ToUID == "" {
			return
		}
		if h.Calls != nil {
			h.Calls.Answer(req.CallID, req.Accept)
		}
		h.Signal(req.ToUID, "CALL_ANSWER_PUSH", map[string]interface{}{
			"callId": req.CallID, "fromUid": c.UID, "accept": req.Accept, "sdp": req.SDP,
		})
	case "CALL_ICE":
		var req struct {
			CallID    string          `json:"callId"`
			ToUID     string          `json:"toUid"`
			Candidate json.RawMessage `json:"candidate"`
		}
		if json.Unmarshal(f.Payload, &req) != nil || req.CallID == "" || req.ToUID == "" || len(req.Candidate) == 0 {
			return
		}
		h.Signal(req.ToUID, "CALL_ICE_PUSH", map[string]interface{}{
			"callId": req.CallID, "fromUid": c.UID, "candidate": req.Candidate,
		})
	case "CALL_HANGUP":
		var req struct {
			CallID string `json:"callId"`
			ToUID  string `json:"toUid"`
		}
		if json.Unmarshal(f.Payload, &req) != nil || req.CallID == "" || req.ToUID == "" {
			return
		}
		if h.Calls != nil {
			h.Calls.End(req.CallID)
		}
		h.Signal(req.ToUID, "CALL_HANGUP_PUSH", map[string]interface{}{
			"callId": req.CallID, "fromUid": c.UID,
		})
	case "TYPING":
		var req struct {
			ConvID string `json:"convId"`
			ToUID  string `json:"toUid"`
		}
		if json.Unmarshal(f.Payload, &req) != nil {
			return
		}
		if req.ToUID != "" {
			h.PushToUser(req.ToUID, &Frame{Cmd: "PRESENCE_PUSH", Payload: mustJSON(map[string]interface{}{
				"kind": "typing", "convId": req.ConvID, "uid": c.UID,
			})})
		}
	default:
		c.send <- &Frame{Cmd: f.Cmd + "_ACK", SeqNo: f.SeqNo, Code: 1002, Message: fmt.Sprintf("不支持的命令字: %s", f.Cmd)}
	}
}

// DisplayName 返回用户展示名，找不到时退回 uid
func (h *Hub) DisplayName(uid string) string {
	var u model.User
	if h.S.DB.Where("uid = ?", uid).First(&u).Error == nil && u.Nickname != "" {
		return u.Nickname
	}
	return uid
}

// PushMessage 把已落库的消息实时推送给对端/群成员，并同步给发送者的其它设备。
// WebSocket 与 HTTP 两条发送通道共用；standard 形态同时发布 JetStream 事件，
// 由集群内各节点消费后向本节点在线用户投递。
func (h *Hub) PushMessage(fromUID, convID, convType, toUID, clientMsgID, msgID, msgType, content, extra string, seq, createdAt int64) {
	// 本节点在线用户立即投递
	h.deliver(fromUID, convID, convType, toUID, clientMsgID, msgID, msgType, content, extra, seq, createdAt)
	// 发布事件给其它网关节点（单节点部署时事件仍会持久化到 JetStream）
	if h.MQ != nil {
		h.MQ.PublishMessage(mq.MessageEvent{
			NodeID: h.NodeID, MsgID: msgID, ClientMsgID: clientMsgID,
			ConvID: convID, ConvType: convType, FromUID: fromUID, ToUID: toUID,
			Type: msgType, Content: content, Extra: extra, Seq: seq, SendTime: createdAt,
		})
	}
}

// deliver 仅做本节点在线投递，不产生 MQ 事件（避免跨节点循环扩散）
func (h *Hub) deliver(fromUID, convID, convType, toUID, clientMsgID, msgID, msgType, content, extra string, seq, createdAt int64) {
	push := &Frame{Cmd: "PUSH", Payload: mustJSON(PushPayload{
		MsgID: msgID, ClientMsgID: clientMsgID, ConvID: convID, ConvType: convType,
		Seq: seq, FromUID: fromUID, FromName: h.DisplayName(fromUID), Type: msgType,
		Content: content, Extra: extra, SendTime: createdAt,
	})}
	// 单聊推给对端，群聊推给全部群成员
	for _, target := range h.Msg.PushTargets(convID, convType, fromUID, toUID) {
		h.PushToUser(target, push)
	}
	// 同步给自己其它设备（本端已乐观上屏，由前端按 clientMsgId 去重）
	h.PushToUser(fromUID, push)
}

// NotifyRead 向单聊对端推送「已读」通知；WebSocket 与 HTTP 两条通道共用
func (h *Hub) NotifyRead(uid, convID string) {
	if convID == "" {
		return
	}
	var conv model.Conversation
	if h.S.DB.Where("conv_id = ? AND owner_uid = ?", convID, uid).First(&conv).Error != nil || conv.PeerID == "" {
		return
	}
	h.PushToUser(conv.PeerID, &Frame{Cmd: "NOTICE_PUSH", Payload: mustJSON(map[string]interface{}{
		"kind": "read", "convId": convID, "uid": uid,
	})})
}

// BroadcastRevoke 向消息对端及会话相关用户推送撤回通知；WebSocket 与 HTTP 两条通道共用
func (h *Hub) BroadcastRevoke(msgID string) {
	var msg model.Message
	if h.S.DB.Where("msg_id = ?", msgID).First(&msg).Error != nil {
		return
	}
	frame := &Frame{Cmd: "REVOKE_PUSH", Payload: mustJSON(map[string]interface{}{
		"msgId": msgID, "convId": msg.ConvID, "status": 2,
	})}
	if msg.ToID != "" {
		h.PushToUser(msg.ToID, frame)
	}
	var convs []model.Conversation
	h.S.DB.Where("conv_id = ?", msg.ConvID).Find(&convs)
	for _, cv := range convs {
		h.PushToUser(cv.OwnerUID, frame)
	}
}

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
