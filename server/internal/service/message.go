package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"nexim/server/internal/model"
	"nexim/server/internal/store"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MessageService struct {
	S *store.Store
}

func NewMessage(s *store.Store) *MessageService { return &MessageService{S: s} }

// seqLua 原子分配会话内自增 seq：
// Redis 中的游标若落后于库内 MAX(seq)（Redis 重启/过期/清空），先抬到库内水位再自增，
// 避免 seq 从 1 重新开始造成的 (conv_id, seq) 唯一键冲突。
var seqLua = redis.NewScript(`
local cur = tonumber(redis.call('GET', KEYS[1]) or '0')
local base = tonumber(ARGV[1])
if cur < base then cur = base end
cur = cur + 1
redis.call('SET', KEYS[1], cur)
return cur
`)

// Send 发送消息：幂等 -> seq 自增 -> 写扩散会话 -> 落库
func (m *MessageService) Send(fromUID, convID, convType, toID, msgType, content, extra, clientMsgID string) (*model.Message, error) {
	if clientMsgID == "" {
		return nil, fmt.Errorf("clientMsgId 不能为空")
	}
	// 幂等
	idemKey := "idem:" + clientMsgID
	val, err := m.S.RDB.Get(m.S.Ctx, idemKey).Result()
	if err == nil && val != "" {
		if exist, gerr := m.S.Msg.GetByMsgID(val); gerr == nil {
			return exist, nil
		}
	}

	// 敏感词策略：拦截 / 替换 / 仅告警
	content, err = m.applySensitive(fromUID, convID, content)
	if err != nil {
		return nil, err
	}

	// 单聊未携带 toUid（历史客户端或会话 peer 缺失）时从 convId 解析对端，保证会话与推送目标完整
	if toID == "" && convTypeCode(convType) == 1 {
		toID = peerFromConvID(convID, fromUID)
	}

	seq, err := m.nextSeq(convID)
	if err != nil {
		return nil, err
	}
	msgID := fmt.Sprintf("m_%d_%d", time.Now().UnixMilli(), seq)
	now := model.Now()
	msg := model.Message{
		MsgID: msgID, ConvID: convID, Seq: seq, FromUID: fromUID, ToID: toID,
		Type: convMsgType(msgType), Content: content, Extra: extra, ClientMsgID: clientMsgID,
		Status: 1, CreatedAt: now,
	}
	if err := m.S.Msg.Create(&msg); err != nil {
		return nil, err
	}
	m.S.RDB.Set(m.S.Ctx, idemKey, msgID, 24*time.Hour)

	m.upsertConversation(convID, convTypeCode(convType), fromUID, toID, msgID, content, seq, now)
	return &msg, nil
}

func (m *MessageService) nextSeq(convID string) (int64, error) {
	// 以消息库（MySQL 或 ScyllaDB）中的最大 seq 为水位，Redis 游标落后时自动抬升
	base, err := m.S.Msg.MaxSeq(convID)
	if err != nil {
		return 0, err
	}
	return seqLua.Run(m.S.Ctx, m.S.RDB, []string{"conv:seq:" + convID}, base).Int64()
}

func (m *MessageService) upsertConversation(convID string, convType int, fromUID, toID, msgID, content string, seq, now int64) {
	peers := map[string]string{}
	owners := []string{fromUID}

	if convType == 2 {
		gid := groupIDOf(convID, toID)
		var members []string
		m.S.DB.Model(&model.GroupMember{}).Where("group_id = ?", gid).Pluck("uid", &members)
		for _, muid := range members {
			if muid != fromUID {
				owners = append(owners, muid)
			}
		}
		for _, owner := range owners {
			peers[owner] = gid
		}
	} else if toID != "" && toID != fromUID {
		owners = append(owners, toID)
		peers[fromUID] = toID
		peers[toID] = fromUID
	} else {
		peers[fromUID] = toID
	}

	for _, owner := range owners {
		peer := peers[owner]
		var c model.Conversation
		err := m.S.DB.Where("conv_id = ? AND owner_uid = ?", convID, owner).First(&c).Error
		if err == gorm.ErrRecordNotFound {
			unread := 0
			if owner != fromUID {
				unread = 1
			}
			c = model.Conversation{
				ConvID: convID, ConvType: convType, OwnerUID: owner, PeerID: peer,
				MaxSeq: seq, ReadSeq: 0, Unread: unread, LastMsgID: msgID, LastMsgAt: now, LastMsg: content,
			}
			if cerr := m.S.DB.Create(&c).Error; cerr != nil {
				// 并发写入时另一请求已建好该行，退化为更新
				m.S.DB.Model(&model.Conversation{}).Where("conv_id = ? AND owner_uid = ?", convID, owner).
					Updates(map[string]interface{}{"last_msg_id": msgID, "last_msg": content, "last_msg_at": now, "max_seq": seq})
			}
		} else if err == nil {
			updates := map[string]interface{}{"last_msg_id": msgID, "last_msg": content, "last_msg_at": now, "max_seq": seq}
			if c.PeerID == "" && peer != "" {
				updates["peer_id"] = peer
			}
			if owner != fromUID {
				updates["unread"] = gorm.Expr("unread + 1")
			}
			m.S.DB.Model(&c).Updates(updates)
		}
	}
}

// PeerOf 客户端未携带 toUid 时，从会话记录反查对端，避免推送目标丢失
func (m *MessageService) PeerOf(convID, ownerUID string) string {
	var c model.Conversation
	if m.S.DB.Where("conv_id = ? AND owner_uid = ?", convID, ownerUID).First(&c).Error != nil {
		return ""
	}
	return c.PeerID
}

// PushTargets 返回消息需要实时推送的其他用户：单聊为对端，群聊为除发送者外的全部群成员
func (m *MessageService) PushTargets(convID, convType, fromUID, toID string) []string {
	if convTypeCode(convType) == 2 {
		gid := groupIDOf(convID, toID)
		var members []string
		m.S.DB.Model(&model.GroupMember{}).Where("group_id = ?", gid).Pluck("uid", &members)
		out := make([]string, 0, len(members))
		for _, muid := range members {
			if muid != fromUID {
				out = append(out, muid)
			}
		}
		return out
	}
	// 单聊：接收方已拉黑发送方时不再实时推送
	if toID != "" && toID != fromUID {
		if m.IsBlocked(toID, fromUID) {
			return nil
		}
		return []string{toID}
	}
	return nil
}

func (m *MessageService) ListMessages(uid, convID string, limit int) []model.Message {
	msgs, err := m.S.Msg.List(convID, limit)
	if err != nil {
		m.S.Log.Warn("list messages failed", zap.Error(err), zap.String("conv", convID))
		return []model.Message{}
	}
	if m.isSingleConv(uid, convID) {
		blocked := m.BlockedMap(uid)
		if len(blocked) > 0 {
			out := msgs[:0]
			for _, msg := range msgs {
				if blocked[msg.FromUID] {
					continue
				}
				out = append(out, msg)
			}
			msgs = out
		}
	}
	return msgs
}

func (m *MessageService) isSingleConv(uid, convID string) bool {
	var c model.Conversation
	if m.S.DB.Where("conv_id = ? AND owner_uid = ?", convID, uid).First(&c).Error != nil {
		return false
	}
	return c.ConvType == 1
}

func (m *MessageService) Conversations(uid string) []model.Conversation {
	var convs []model.Conversation
	m.S.DB.Where("owner_uid = ?", uid).Order("pinned desc, last_msg_at desc").Find(&convs)
	blocked := m.BlockedMap(uid)
	out := convs[:0]
	for i := range convs {
		// 历史会话 peer_id 缺失时从 convId 解析并回填，避免会话列表出现空对端
		if convs[i].ConvType == 1 && convs[i].PeerID == "" {
			if p := peerFromConvID(convs[i].ConvID, convs[i].OwnerUID); p != "" {
				convs[i].PeerID = p
				m.S.DB.Model(&model.Conversation{}).
					Where("conv_id = ? AND owner_uid = ?", convs[i].ConvID, convs[i].OwnerUID).
					Update("peer_id", p)
			}
		}
		if convs[i].ConvType == 2 {
			var g model.Group
			if m.S.DB.Where("group_id = ?", convs[i].PeerID).First(&g).Error == nil {
				convs[i].PeerName = g.Name
				convs[i].PeerAvatar = g.Avatar
			}
			out = append(out, convs[i])
			continue
		}
		if convs[i].PeerID == "" {
			continue
		}
		if blocked[convs[i].PeerID] {
			continue
		}
		var u model.User
		if m.S.DB.Where("uid = ?", convs[i].PeerID).First(&u).Error == nil {
			convs[i].PeerName = u.Nickname
			convs[i].PeerAvatar = u.Avatar
		}
		out = append(out, convs[i])
	}
	return out
}

func (m *MessageService) MarkRead(uid, convID string) error {
	return m.S.DB.Model(&model.Conversation{}).
		Where("conv_id = ? AND owner_uid = ?", convID, uid).
		Updates(map[string]interface{}{"unread": 0, "read_seq": gorm.Expr("max_seq")}).Error
}

func (m *MessageService) ReadAll(uid string) error {
	return m.S.DB.Model(&model.Conversation{}).Where("owner_uid = ?", uid).
		Updates(map[string]interface{}{"unread": 0, "read_seq": gorm.Expr("max_seq")}).Error
}

func (m *MessageService) Revoke(uid, msgID string) error {
	msg, err := m.S.Msg.GetByMsgID(msgID)
	if err != nil {
		return err
	}
	if msg.FromUID != uid {
		return fmt.Errorf("只能撤回自己的消息")
	}
	return m.S.Msg.UpdateStatus(msgID, 2)
}

func (m *MessageService) SetConversationSetting(uid, convID string, pinned, muted *int) error {
	updates := map[string]interface{}{}
	if pinned != nil {
		updates["pinned"] = *pinned
	}
	if muted != nil {
		updates["muted"] = *muted
	}
	if len(updates) == 0 {
		return nil
	}
	return m.S.DB.Model(&model.Conversation{}).Where("conv_id = ? AND owner_uid = ?", convID, uid).Updates(updates).Error
}

// groupIDOf 解析群会话对应的群 ID：优先用客户端传入的 toUid（群 ID），否则从 conv_<groupID> 推导
func groupIDOf(convID, toID string) string {
	if toID != "" && !strings.HasPrefix(toID, "conv_") {
		return toID
	}
	return strings.TrimPrefix(convID, "conv_")
}

// convUIDPattern 匹配会话 ID 中的用户 ID（形如 u_1001 / u_1795066626962）
var convUIDPattern = regexp.MustCompile(`u_[A-Za-z0-9]+`)

// peerFromConvID 从单聊会话 ID（c_<uidA>_<uidB> 或 conv_<uidA>_<uidB>）解析对端；
// 自聊会话（两端相同）返回自身，无法解析时返回空串
func peerFromConvID(convID, ownerUID string) string {
	ids := convUIDPattern.FindAllString(convID, -1)
	for _, id := range ids {
		if id != ownerUID {
			return id
		}
	}
	if len(ids) > 0 {
		return ownerUID
	}
	return ""
}

func convTypeCode(s string) int {
	switch s {
	case "group", "2":
		return 2
	case "system", "3":
		return 3
	default:
		return 1
	}
}

func convMsgType(s string) int {
	switch s {
	case "image", "2":
		return 2
	case "voice", "4":
		return 4
	case "file", "5":
		return 5
	case "video", "6":
		return 6
	case "system", "3":
		return 3
	default:
		return 1
	}
}

// DeleteConversation 删除某用户的会话记录（消息本身保留，收到新消息时会话会重新建立）
func (m *MessageService) DeleteConversation(uid, convID string) error {
	return m.S.DB.Where("conv_id = ? AND owner_uid = ?", convID, uid).Delete(&model.Conversation{}).Error
}

// applySensitive 对文本做敏感词检测，按 Redis 中的策略（block/mask/alert）处理并记录命中
func (m *MessageService) applySensitive(uid, convID, content string) (string, error) {
	if content == "" {
		return content, nil
	}
	words, err := m.S.RDB.SMembers(m.S.Ctx, "sensitive:words:v1").Result()
	if err != nil || len(words) == 0 {
		return content, nil
	}
	lower := strings.ToLower(content)
	matched := ""
	for _, w := range words {
		if w != "" && strings.Contains(lower, strings.ToLower(w)) {
			matched = w
			break
		}
	}
	if matched == "" {
		return content, nil
	}
	strategy, _ := m.S.RDB.Get(m.S.Ctx, "sensitive:strategy").Result()
	if strategy != "mask" && strategy != "alert" {
		strategy = "block"
	}
	hit := model.SensitiveHit{
		UID: uid, ConvID: convID, Word: matched, Action: strategy,
		Content: content, CreatedAt: model.Now(),
	}
	if err := m.S.DB.Create(&hit).Error; err != nil {
		m.S.Log.Warn("save sensitive hit failed", zap.Error(err))
	}
	if strategy == "block" {
		return "", fmt.Errorf("内容包含敏感词「%s」，已被拦截", matched)
	}
	if strategy == "mask" {
		for _, w := range words {
			if w == "" {
				continue
			}
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(w))
			content = re.ReplaceAllStringFunc(content, func(s string) string {
				return strings.Repeat("*", utf8.RuneCountInString(s))
			})
		}
	}
	return content, nil
}
