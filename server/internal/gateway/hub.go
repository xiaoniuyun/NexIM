package gateway

import (
	"encoding/json"
	"sync"
	"time"

	"nexim/server/internal/config"
	"nexim/server/internal/model"
	"nexim/server/internal/mq"
	"nexim/server/internal/pkg"
	"nexim/server/internal/service"
	"nexim/server/internal/store"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Frame struct {
	TraceID string `json:"traceId,omitempty"`
	Cmd     string `json:"cmd"`
	SeqNo   uint64 `json:"seqNo,omitempty"`
	// Code 不使用 omitempty：成功时的 code=0 必须显式下发，前端据此判定 ACK 成功
	Code       int             `json:"code"`
	Message    string          `json:"message,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	ServerTime int64           `json:"serverTime,omitempty"`
}

type Client struct {
	Conn     *websocket.Conn
	UID      string
	DeviceID string
	Platform string
	Claims   *pkg.Claims
	send     chan *Frame
	hub      *Hub
	mu       sync.Mutex
}

type Hub struct {
	clients map[string]map[string]*Client // uid -> deviceId -> client
	mu      sync.RWMutex
	S       *store.Store
	C       *config.Config
	Msg     *service.MessageService
	Contact *service.ContactService
	Auth    *service.AuthService
	Calls   *service.CallService
	Log     *zap.Logger
	MQ      *mq.NATS // standard 形态下的 JetStream 事件总线，nil 表示未启用
	NodeID  string   // 本节点标识，用于过滤自己发布的 MQ 事件
}

func NewHub(s *store.Store, c *config.Config, msg *service.MessageService, contact *service.ContactService, auth *service.AuthService, call *service.CallService, log *zap.Logger) *Hub {
	return &Hub{clients: map[string]map[string]*Client{}, S: s, C: c, Msg: msg, Contact: contact, Auth: auth, Calls: call, Log: log}
}

func NewClient(conn *websocket.Conn, uid, deviceID, platform string, claims *pkg.Claims, hub *Hub) *Client {
	return &Client{Conn: conn, UID: uid, DeviceID: deviceID, Platform: platform, Claims: claims, send: make(chan *Frame, 64), hub: hub}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	if h.clients[c.UID] == nil {
		h.clients[c.UID] = map[string]*Client{}
	}
	// 同设备互踢
	if old := h.clients[c.UID][c.DeviceID]; old != nil && old != c {
		old.send <- &Frame{Cmd: "KICK", Code: 2002, Message: "账号在其他设备登录"}
		old.Conn.Close()
	}
	h.clients[c.UID][c.DeviceID] = c
	h.S.RDB.HSet(h.S.Ctx, "route:"+c.UID, c.DeviceID, "local")
	h.S.RDB.Set(h.S.Ctx, "online:"+c.UID, "1", 120*time.Second)
	h.mu.Unlock()
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if devs := h.clients[c.UID]; devs != nil {
		if devs[c.DeviceID] == c {
			delete(devs, c.DeviceID)
			h.S.RDB.HDel(h.S.Ctx, "route:"+c.UID, c.DeviceID)
		}
		if len(devs) == 0 {
			delete(h.clients, c.UID)
			h.S.RDB.Del(h.S.Ctx, "online:"+c.UID)
		}
	}
	h.mu.Unlock()
}

func (h *Hub) PushToUser(uid string, frame *Frame) {
	h.mu.RLock()
	devs := h.clients[uid]
	list := make([]*Client, 0, len(devs))
	for _, c := range devs {
		list = append(list, c)
	}
	h.mu.RUnlock()
	for _, c := range list {
		select {
		case c.send <- frame:
		default:
		}
	}
}

// SetMQ 启用 JetStream 事件总线（standard 形态），并注册跨节点消息 / 通话信令投递
func (h *Hub) SetMQ(n *mq.NATS, nodeID string) {
	h.MQ = n
	h.NodeID = nodeID
	if n == nil {
		return
	}
	if err := n.ConsumeMessages("nexim-node-"+nodeID, h.OnRemoteMessage); err != nil {
		h.Log.Warn("consume message events failed", zap.Error(err))
	}
	if err := n.ConsumeCalls("nexim-call-"+nodeID, h.OnRemoteSignal); err != nil {
		h.Log.Warn("consume call signals failed", zap.Error(err))
	}
}

// IsOnline 用户是否在线（本节点有连接，或其它节点写入了 Redis 在线标记）
func (h *Hub) IsOnline(uid string) bool {
	h.mu.RLock()
	n := len(h.clients[uid])
	h.mu.RUnlock()
	if n > 0 {
		return true
	}
	online, _ := h.S.RDB.Exists(h.S.Ctx, "online:"+uid).Result()
	return online > 0
}

// Signal 向目标用户投递通话信令：本节点直投，其它节点经 JetStream 转发
func (h *Hub) Signal(targetUID, cmd string, payload interface{}) {
	raw := mustJSON(payload)
	h.deliverSignal(targetUID, cmd, raw)
	if h.MQ != nil {
		h.MQ.PublishCall(mq.CallSignal{
			NodeID: h.NodeID, TargetUID: targetUID, Cmd: cmd, Payload: raw,
		})
	}
}

func (h *Hub) deliverSignal(targetUID, cmd string, payload []byte) {
	h.PushToUser(targetUID, &Frame{Cmd: cmd, Payload: payload})
}

// OnRemoteSignal 处理其它节点发布的通话信令，只做本节点投递
func (h *Hub) OnRemoteSignal(ev mq.CallSignal) {
	if ev.NodeID == h.NodeID {
		return
	}
	h.deliverSignal(ev.TargetUID, ev.Cmd, ev.Payload)
}

// OnRemoteMessage 处理其它节点发布的消息事件：只做本节点在线投递，不再回发事件
func (h *Hub) OnRemoteMessage(ev mq.MessageEvent) {
	if ev.NodeID == h.NodeID {
		return
	}
	h.deliver(ev.FromUID, ev.ConvID, ev.ConvType, ev.ToUID, ev.ClientMsgID,
		ev.MsgID, ev.Type, ev.Content, ev.Extra, ev.Seq, ev.SendTime)
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case f, ok := <-c.send:
			if !ok {
				return
			}
			f.ServerTime = model.Now()
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.Conn.WriteJSON(f)
			c.mu.Unlock()
			if err != nil {
				return
			}
		case <-ticker.C:
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			// 浏览器会自动回 Pong，用于刷新读超时；仅靠 HEARTBEAT 文本帧无法保活连接
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.mu.Unlock()
				return
			}
			// HEARTBEAT 帧供客户端做连接假死检测
			err := c.Conn.WriteJSON(&Frame{Cmd: "HEARTBEAT", ServerTime: model.Now()})
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Conn.Close()
	}()
	c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})
	for {
		var f Frame
		if err := c.Conn.ReadJSON(&f); err != nil {
			return
		}
		c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		c.hub.handle(c, &f)
	}
}
