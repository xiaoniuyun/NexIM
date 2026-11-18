// Package mq 提供消息队列接入。standard 形态使用 NATS JetStream：
// 消息落库后发布 im.message.created 事件，集群内各节点消费事件后向本节点在线用户投递，
// 从而实现多网关节点的实时消息扩散与事件持久化（可回放、可审计）。
package mq

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// SubjectMessage 消息创建事件主题
const SubjectMessage = "im.message.created"

// SubjectCall 通话信令主题（跨网关节点投递）
const SubjectCall = "im.call.signal"

// CallSignal 通话信令事件：投递给目标用户所在节点后再下发 WebSocket
type CallSignal struct {
	NodeID    string          `json:"nodeId"`
	TargetUID string          `json:"targetUid"`
	Cmd       string          `json:"cmd"`
	Payload   json.RawMessage `json:"payload"`
}

// MessageEvent 消息事件：字段与 WebSocket PUSH 负载保持一致，便于各节点直接投递
type MessageEvent struct {
	NodeID      string `json:"nodeId"`
	MsgID       string `json:"msgId"`
	ClientMsgID string `json:"clientMsgId"`
	ConvID      string `json:"convId"`
	ConvType    string `json:"convType"`
	FromUID     string `json:"fromUid"`
	ToUID       string `json:"toUid"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Seq         int64  `json:"seq"`
	SendTime    int64  `json:"sendTime"`
}

// NATS JetStream 客户端
type NATS struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	stream string
	log    *zap.Logger
}

// NewNATS 连接 NATS 并确保 JetStream 流存在
func NewNATS(url, stream, retention string, log *zap.Logger) (*NATS, error) {
	if stream == "" {
		stream = "IM_EVENT"
	}
	nc, err := nats.Connect(url,
		nats.Name("nexim-server"),
		nats.Timeout(5*time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats jetstream: %w", err)
	}
	n := &NATS{conn: nc, js: js, stream: stream, log: log}
	if err := n.ensureStream(retention); err != nil {
		nc.Close()
		return nil, err
	}
	return n, nil
}

func (n *NATS) ensureStream(retention string) error {
	maxAge, err := time.ParseDuration(retention)
	if err != nil || maxAge <= 0 {
		maxAge = 7 * 24 * time.Hour
	}
	cfg := &nats.StreamConfig{
		Name:       n.stream,
		Subjects:   []string{"im.>"},
		Retention:  nats.LimitsPolicy,
		MaxAge:     maxAge,
		Storage:    nats.FileStorage,
		Replicas:   1,
		Duplicates: 2 * time.Minute,
	}
	info, err := n.js.StreamInfo(n.stream)
	if err == nil {
		// 已存在：保留副本数，只同步主题与保留策略
		cfg.Replicas = info.Config.Replicas
		_, err = n.js.UpdateStream(cfg)
		if err != nil {
			return fmt.Errorf("nats update stream %s: %w", n.stream, err)
		}
		return nil
	}
	if errors.Is(err, nats.ErrStreamNotFound) {
		if _, err = n.js.AddStream(cfg); err != nil {
			return fmt.Errorf("nats add stream %s: %w", n.stream, err)
		}
		return nil
	}
	return fmt.Errorf("nats stream info %s: %w", n.stream, err)
}

// PublishMessage 发布消息事件；以 msgId 作为去重键，重复提交不会产生重复事件
func (n *NATS) PublishMessage(ev MessageEvent) {
	if n == nil || n.js == nil {
		return
	}
	b, err := json.Marshal(ev)
	if err != nil {
		n.log.Warn("marshal mq event failed", zap.Error(err))
		return
	}
	opts := []nats.PubOpt{}
	if ev.MsgID != "" {
		opts = append(opts, nats.MsgId(ev.MsgID))
	}
	if _, err := n.js.Publish(SubjectMessage, b, opts...); err != nil {
		n.log.Warn("nats publish message event failed", zap.Error(err), zap.String("subject", SubjectMessage))
	}
}

// ConsumeMessages 以节点独立 durable 消费消息事件；只消费启动后的新事件
func (n *NATS) ConsumeMessages(durable string, handler func(MessageEvent)) error {
	opts := []nats.SubOpt{
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.BindStream(n.stream),
		nats.MaxAckPending(1024),
	}
	if _, err := n.js.ConsumerInfo(n.stream, durable); err != nil {
		opts = append(opts, nats.Durable(durable), nats.DeliverNew())
	} else {
		opts = append(opts, nats.Bind(n.stream, durable))
	}
	_, err := n.js.Subscribe(SubjectMessage, func(m *nats.Msg) {
		var ev MessageEvent
		if e := json.Unmarshal(m.Data, &ev); e != nil {
			n.log.Warn("bad mq event", zap.Error(e))
			_ = m.Term()
			return
		}
		handler(ev)
		_ = m.Ack()
	}, opts...)
	if err != nil {
		return fmt.Errorf("nats subscribe %s: %w", SubjectMessage, err)
	}
	return nil
}

// PublishCall 发布通话信令事件（目标用户不在本节点时由对应节点消费投递）
func (n *NATS) PublishCall(ev CallSignal) {
	if n == nil || n.js == nil {
		return
	}
	b, err := json.Marshal(ev)
	if err != nil {
		n.log.Warn("marshal call signal failed", zap.Error(err))
		return
	}
	if _, err := n.js.Publish(SubjectCall, b); err != nil {
		n.log.Warn("nats publish call signal failed", zap.Error(err), zap.String("cmd", ev.Cmd))
	}
}

// ConsumeCalls 以节点独立 durable 消费通话信令
func (n *NATS) ConsumeCalls(durable string, handler func(CallSignal)) error {
	opts := []nats.SubOpt{
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.BindStream(n.stream),
		nats.MaxAckPending(1024),
	}
	if _, err := n.js.ConsumerInfo(n.stream, durable); err != nil {
		opts = append(opts, nats.Durable(durable), nats.DeliverNew())
	} else {
		opts = append(opts, nats.Bind(n.stream, durable))
	}
	_, err := n.js.Subscribe(SubjectCall, func(m *nats.Msg) {
		var ev CallSignal
		if e := json.Unmarshal(m.Data, &ev); e != nil {
			n.log.Warn("bad call signal", zap.Error(e))
			_ = m.Term()
			return
		}
		handler(ev)
		_ = m.Ack()
	}, opts...)
	if err != nil {
		return fmt.Errorf("nats subscribe %s: %w", SubjectCall, err)
	}
	return nil
}

// Close 优雅关闭连接
func (n *NATS) Close() {
	if n == nil || n.conn == nil {
		return
	}
	_ = n.conn.Drain()
	n.conn.Close()
}
