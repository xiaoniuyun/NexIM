package model

import "time"

type User struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UID          string `gorm:"size:32;uniqueIndex" json:"uid"`
	Username     string `gorm:"size:64;uniqueIndex" json:"username"`
	PasswordHash string `gorm:"size:128" json:"-"`
	Nickname     string `gorm:"size:64" json:"nickname"`
	Avatar       string `gorm:"size:256" json:"avatar"`
	DeptID       int64  `gorm:"default:0" json:"deptId"`
	Phone        string `gorm:"size:32" json:"phone"`
	Title        string `gorm:"size:64" json:"title"`
	Status       int    `gorm:"default:1" json:"status"`
	LastLoginAt  int64  `gorm:"default:0" json:"lastLoginAt"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}

type Device struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UID         string `gorm:"size:32;index:idx_device_uid" json:"uid"`
	DeviceID    string `gorm:"size:64;uniqueIndex:uk_device" json:"deviceId"`
	Platform    string `gorm:"size:16" json:"platform"`
	DeviceName  string `gorm:"size:64" json:"deviceName"`
	PushChannel string `gorm:"size:24" json:"pushChannel"`
	PushToken   string `gorm:"size:256" json:"pushToken"`
	LastActive  int64  `json:"lastActive"`
	CreatedAt   int64  `json:"createdAt"`
}

type Conversation struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ConvID   string `gorm:"size:64;uniqueIndex:uk_conv_owner" json:"convId"`
	ConvType int    `json:"convType"` // 1单聊 2群聊 3系统
	// 同一条消息会为会话双方各写一行会话记录，唯一键必须是 (conv_id, owner_uid)
	OwnerUID   string `gorm:"size:32;uniqueIndex:uk_conv_owner;index:idx_owner_peer" json:"ownerUid"`
	PeerID     string `gorm:"size:64;index:idx_owner_peer" json:"peerId"`
	MaxSeq     int64  `gorm:"default:0" json:"maxSeq"`
	ReadSeq    int64  `gorm:"default:0" json:"readSeq"`
	Unread     int    `gorm:"default:0" json:"unread"`
	Pinned     int    `gorm:"default:0" json:"pinned"`
	Muted      int    `gorm:"default:0" json:"muted"`
	LastMsgID  string `gorm:"size:64" json:"lastMsgId"`
	LastMsgAt  int64  `json:"lastMsgAt"`
	LastMsg    string `gorm:"type:text" json:"lastMsg"`
	PeerName   string `gorm:"size:64;default:''" json:"peerName"`
	PeerAvatar string `gorm:"size:256;default:''" json:"peerAvatar"`
}

type Message struct {
	ID    int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	MsgID string `gorm:"size:64;uniqueIndex" json:"msgId"`
	// seq 只在单个会话内自增，唯一键必须是 (conv_id, seq)，不能是 seq 单列
	ConvID      string `gorm:"size:64;uniqueIndex:uk_conv_seq" json:"convId"`
	Seq         int64  `gorm:"uniqueIndex:uk_conv_seq" json:"seq"`
	FromUID     string `gorm:"size:32" json:"fromUid"`
	ToID        string `gorm:"size:64" json:"toId"`
	Type        int    `json:"type"`
	Content     string `gorm:"type:text" json:"content"`
	Extra       string `gorm:"type:text" json:"extra"`
	ClientMsgID string `gorm:"size:64;uniqueIndex:uk_client" json:"clientMsgId"`
	Status      int    `gorm:"default:1" json:"status"` // 1正常 2撤回 3删除
	CreatedAt   int64  `json:"createdAt"`
}

// Call 1对1音视频通话记录
type Call struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	CallID     string `gorm:"size:64;uniqueIndex" json:"callId"`
	ConvID     string `gorm:"size:64;index" json:"convId"`
	CallerUID  string `gorm:"size:32;index" json:"callerUid"`
	CalleeUID  string `gorm:"size:32;index" json:"calleeUid"`
	CallType   string `gorm:"size:16" json:"callType"` // audio | video
	Status     int    `gorm:"default:1" json:"status"` // 1呼叫中 2通话中 3已结束 4未接/拒接
	CreatedAt  int64  `json:"createdAt"`
	AnsweredAt int64  `json:"answeredAt"`
	EndedAt    int64  `json:"endedAt"`
	Duration   int    `json:"duration"` // 秒
	RoomID     string `gorm:"size:64" json:"roomId"`
	Mode       string `gorm:"size:16;default:'P2P'" json:"mode"`
	Members    int    `gorm:"default:2" json:"members"`
	Recorded   int    `gorm:"default:0" json:"recorded"`
	ForceEnded int    `gorm:"default:0" json:"forceEnded"`
	// 客户端上报的质量采样（MOS 1~5，丢包率 0~100，RTT/Jitter 毫秒）
	MOS        float64 `gorm:"default:0" json:"mos"`
	PacketLoss float64 `gorm:"default:0" json:"packetLoss"`
	RTT        int     `gorm:"default:0" json:"rtt"`
	Jitter     int     `gorm:"default:0" json:"jitter"`
}

type Friend struct {
	UID       string `gorm:"size:32;primaryKey" json:"uid"`
	FriendUID string `gorm:"size:32;primaryKey" json:"friendUid"`
	Remark    string `gorm:"size:64" json:"remark"`
	Status    int    `gorm:"default:1" json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

type FriendApply struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	FromUID   string `gorm:"size:32" json:"fromUid"`
	ToUID     string `gorm:"size:32;index" json:"toUid"`
	Message   string `gorm:"type:text" json:"message"`
	Status    int    `gorm:"default:0" json:"status"` // 0待处理 1同意 2拒绝
	CreatedAt int64  `json:"createdAt"`
}

type Dept struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID  int64  `gorm:"default:0" json:"parentId"`
	Name      string `gorm:"size:64" json:"name"`
	Path      string `gorm:"size:128;index" json:"path"`
	Sort      int    `gorm:"default:0" json:"sort"`
	LeaderUID string `gorm:"size:32" json:"leaderUid"`
	Status    int    `gorm:"default:1" json:"status"`
}

type AuditLog struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UID       string `gorm:"size:32;index" json:"uid"`
	Action    string `gorm:"size:64" json:"action"`
	Target    string `gorm:"size:128" json:"target"`
	Detail    string `gorm:"type:text" json:"detail"`
	CreatedAt int64  `json:"createdAt"`
}

type Group struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID     string `gorm:"size:64;uniqueIndex" json:"groupId"`
	Name        string `gorm:"size:64" json:"name"`
	OwnerUID    string `gorm:"size:32" json:"ownerUid"`
	Avatar      string `gorm:"size:256" json:"avatar"`
	Notice      string `gorm:"type:text" json:"notice"`
	MemberCount int    `gorm:"default:0" json:"memberCount"`
	Status      int    `gorm:"default:1" json:"status"`
	CreatedAt   int64  `json:"createdAt"`
}

type GroupMember struct {
	GroupID    string `gorm:"size:64;primaryKey" json:"groupId"`
	UID        string `gorm:"size:32;primaryKey" json:"uid"`
	Role       string `gorm:"size:16" json:"role"` // owner/admin/member
	MutedUntil int64  `json:"mutedUntil"`
	JoinedAt   int64  `json:"joinedAt"`
}

type FileRecord struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	FileID    string `gorm:"size:64;uniqueIndex" json:"fileId"`
	UID       string `gorm:"size:32;index" json:"uid"`
	Name      string `gorm:"size:256" json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `gorm:"size:64" json:"mimeType"`
	Path      string `gorm:"size:512" json:"-"`
	ConvID    string `gorm:"size:64" json:"convId"`
	Status    int    `gorm:"default:1" json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

type Notice struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Title     string `gorm:"size:128" json:"title"`
	Content   string `gorm:"type:text" json:"content"`
	Author    string `gorm:"size:64" json:"author"`
	Scope     string `gorm:"size:32;default:'全员'" json:"scope"`
	Pinned    int    `gorm:"default:0" json:"pinned"`
	Status    int    `gorm:"default:1" json:"status"` // 1已发布 2下线
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type Role struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string `gorm:"size:64;uniqueIndex" json:"code"`
	Name        string `gorm:"size:64" json:"name"`
	Permissions string `gorm:"type:text" json:"permissions"` // JSON 数组
	DataScope   string `gorm:"size:32" json:"dataScope"`
	Version     int    `gorm:"default:1" json:"version"`
}

type UserRole struct {
	UID    string `gorm:"size:32;primaryKey" json:"uid"`
	RoleID int64  `gorm:"primaryKey" json:"roleId"`
}

// Favorite 消息收藏：对消息内容做快照，消息撤回/删除后仍可查看
type Favorite struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UID       string `gorm:"size:32;uniqueIndex:uk_favorite" json:"uid"`
	MsgID     string `gorm:"size:64;uniqueIndex:uk_favorite" json:"msgId"`
	ConvID    string `gorm:"size:64;index" json:"convId"`
	MsgType   int    `json:"msgType"`
	Content   string `gorm:"type:text" json:"content"`
	Extra     string `gorm:"type:text" json:"extra"`
	FromUID   string `gorm:"size:32" json:"fromUid"`
	FromName  string `gorm:"size:64" json:"fromName"`
	CreatedAt int64  `json:"createdAt"`
}

// SensitiveHit 敏感词命中记录
type SensitiveHit struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UID       string `gorm:"size:32;index" json:"uid"`
	ConvID    string `gorm:"size:64" json:"convId"`
	Word      string `gorm:"size:64" json:"word"`
	Action    string `gorm:"size:16" json:"action"` // block | mask | alert
	Content   string `gorm:"type:text" json:"content"`
	CreatedAt int64  `json:"createdAt"`
}

// SystemSetting 管理端系统设置（键值对）
type SystemSetting struct {
	Key       string `gorm:"size:64;primaryKey" json:"key"`
	Value     string `gorm:"type:text" json:"value"`
	UpdatedAt int64  `json:"updatedAt"`
}

// LoginLog 登录日志
type LoginLog struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UID       string `gorm:"size:32;index" json:"uid"`
	Account   string `gorm:"size:64;index" json:"account"`
	IP        string `gorm:"size:64" json:"ip"`
	Platform  string `gorm:"size:16" json:"platform"`
	DeviceID  string `gorm:"size:64" json:"deviceId"`
	Result    string `gorm:"size:16" json:"result"` // success | fail
	Risk      string `gorm:"size:32" json:"risk"`
	CreatedAt int64  `gorm:"index" json:"createdAt"`
}

// Blacklist 用户黑名单：拉黑后不再收到对方单聊消息
type Blacklist struct {
	UID        string `gorm:"size:32;primaryKey" json:"uid"`
	BlockedUID string `gorm:"size:32;primaryKey" json:"blockedUid"`
	CreatedAt  int64  `json:"createdAt"`
}

func (User) TableName() string         { return "sys_user" }
func (Device) TableName() string       { return "sys_device" }
func (Conversation) TableName() string { return "im_conversation" }
func (Message) TableName() string      { return "im_message_0000" }
func (Call) TableName() string         { return "im_call" }
func (Friend) TableName() string       { return "im_friend" }
func (FriendApply) TableName() string  { return "im_friend_apply" }
func (Dept) TableName() string         { return "sys_dept" }
func (AuditLog) TableName() string     { return "sys_audit_log" }
func (Group) TableName() string        { return "im_group" }
func (GroupMember) TableName() string  { return "im_group_member" }
func (FileRecord) TableName() string   { return "im_file" }
func (Notice) TableName() string       { return "sys_notice" }
func (Role) TableName() string         { return "sys_role" }
func (UserRole) TableName() string     { return "sys_user_role" }
func (Favorite) TableName() string     { return "im_favorite" }
func (SensitiveHit) TableName() string { return "sys_sensitive_hit" }
func (SystemSetting) TableName() string { return "sys_setting" }
func (LoginLog) TableName() string     { return "sys_login_log" }
func (Blacklist) TableName() string    { return "im_blacklist" }

func Now() int64 { return time.Now().UnixMilli() }
