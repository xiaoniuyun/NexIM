package store

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"nexim/server/internal/config"
	"nexim/server/internal/model"

	"github.com/gocql/gocql"
)

// ScyllaDB 消息库：消息按会话分区、seq 聚类（PRIMARY KEY ((conv_id), seq)），
// 另建 msg_id 反查表（撤回/幂等回查）与按月分区的审计时间线表。

const (
	scyllaTableMessage   = "im_message"
	scyllaTableByID      = "im_message_by_id"
	scyllaTableTimeline  = "im_message_timeline"
	scyllaAuditMonthSpan = 2 // 审计查询覆盖最近 N 个自然月
)

// OpenScylla 连接 ScyllaDB 并完成 keyspace / 表结构初始化
func OpenScylla(cfg config.DB) (*gocql.Session, error) {
	hosts := cfg.Scylla.Hosts
	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1:9042"}
	}
	keyspace := cfg.Scylla.Keyspace
	if keyspace == "" {
		keyspace = "nexim"
	}
	if !validKeyspace(keyspace) {
		return nil, fmt.Errorf("invalid scylla keyspace %q", keyspace)
	}
	rf := cfg.Scylla.ReplicationFactor
	if rf <= 0 {
		rf = 1
	}
	timeout := 5 * time.Second
	if d, err := time.ParseDuration(cfg.Scylla.Timeout); err == nil && d > 0 {
		timeout = d
	}

	cluster := gocql.NewCluster(hosts...)
	cluster.Timeout = timeout
	cluster.ConnectTimeout = timeout
	cluster.Consistency = gocql.Quorum
	if cfg.Scylla.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.Scylla.Username,
			Password: cfg.Scylla.Password,
		}
	}

	boot, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("scylla connect: %w", err)
	}
	ddl := fmt.Sprintf(`CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class':'SimpleStrategy','replication_factor':%d}`, keyspace, rf)
	if err := boot.Query(ddl).Exec(); err != nil {
		boot.Close()
		return nil, fmt.Errorf("scylla create keyspace: %w", err)
	}
	boot.Close()

	cluster.Keyspace = keyspace
	sess, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("scylla connect keyspace: %w", err)
	}
	if err := initScyllaSchema(sess); err != nil {
		sess.Close()
		return nil, err
	}
	return sess, nil
}

func validKeyspace(s string) bool {
	if s == "" || len(s) > 48 {
		return false
	}
	for _, c := range s {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
			return false
		}
	}
	return true
}

func initScyllaSchema(sess *gocql.Session) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ` + scyllaTableMessage + ` (
			conv_id text, seq bigint, msg_id text, from_uid text, to_id text, msg_type int,
			content text, extra text, client_msg_id text, status int, created_at bigint,
			PRIMARY KEY ((conv_id), seq)
		) WITH CLUSTERING ORDER BY (seq DESC)`,
		`CREATE TABLE IF NOT EXISTS ` + scyllaTableByID + ` (
			msg_id text PRIMARY KEY, conv_id text, seq bigint, from_uid text, to_id text, msg_type int,
			content text, extra text, client_msg_id text, status int, created_at bigint
		)`,
		`CREATE TABLE IF NOT EXISTS ` + scyllaTableTimeline + ` (
			bucket text, created_at bigint, msg_id text, conv_id text, seq bigint, from_uid text,
			to_id text, msg_type int, content text, extra text, client_msg_id text, status int,
			PRIMARY KEY ((bucket), created_at, msg_id)
		) WITH CLUSTERING ORDER BY (created_at DESC, msg_id ASC)`,
	}
	for _, s := range stmts {
		if err := sess.Query(s).Exec(); err != nil {
			return fmt.Errorf("scylla schema: %w", err)
		}
	}
	return nil
}

// ============ ScyllaDB 实现（standard 形态消息库） ============

type scyllaMessageRepo struct{ sess *gocql.Session }

func NewScyllaMessageRepo(sess *gocql.Session) MessageRepo { return &scyllaMessageRepo{sess: sess} }

func (r *scyllaMessageRepo) Driver() string { return "scylla" }

func bucketOf(ts int64) string { return time.UnixMilli(ts).Format("200601") }

func recentBuckets(n int) []string {
	out := make([]string, 0, n)
	t := time.Now()
	for i := 0; i < n; i++ {
		out = append(out, t.AddDate(0, -i, 0).Format("200601"))
	}
	return out
}

func (r *scyllaMessageRepo) Create(m *model.Message) error {
	batch := r.sess.NewBatch(gocql.LoggedBatch)
	batch.Query(`INSERT INTO `+scyllaTableMessage+`
		(conv_id, seq, msg_id, from_uid, to_id, msg_type, content, extra, client_msg_id, status, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		m.ConvID, m.Seq, m.MsgID, m.FromUID, m.ToID, m.Type, m.Content, m.Extra, m.ClientMsgID, m.Status, m.CreatedAt)
	batch.Query(`INSERT INTO `+scyllaTableByID+`
		(msg_id, conv_id, seq, from_uid, to_id, msg_type, content, extra, client_msg_id, status, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		m.MsgID, m.ConvID, m.Seq, m.FromUID, m.ToID, m.Type, m.Content, m.Extra, m.ClientMsgID, m.Status, m.CreatedAt)
	batch.Query(`INSERT INTO `+scyllaTableTimeline+`
		(bucket, created_at, msg_id, conv_id, seq, from_uid, to_id, msg_type, content, extra, client_msg_id, status)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		bucketOf(m.CreatedAt), m.CreatedAt, m.MsgID, m.ConvID, m.Seq, m.FromUID, m.ToID, m.Type,
		m.Content, m.Extra, m.ClientMsgID, m.Status)
	return r.sess.ExecuteBatch(batch)
}

func (r *scyllaMessageRepo) GetByMsgID(msgID string) (*model.Message, error) {
	var m model.Message
	err := r.sess.Query(`SELECT msg_id, conv_id, seq, from_uid, to_id, msg_type, content, extra,
		client_msg_id, status, created_at FROM `+scyllaTableByID+` WHERE msg_id = ?`, msgID).
		Scan(&m.MsgID, &m.ConvID, &m.Seq, &m.FromUID, &m.ToID, &m.Type, &m.Content, &m.Extra,
			&m.ClientMsgID, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *scyllaMessageRepo) List(convID string, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 200
	}
	// 非主键列不能出现在 WHERE（Scylla 需 ALLOW FILTERING），status=2 的撤回消息也要展示
	iter := r.sess.Query(`SELECT msg_id, conv_id, seq, from_uid, to_id, msg_type, content, extra,
		client_msg_id, status, created_at FROM `+scyllaTableMessage+`
		WHERE conv_id = ? LIMIT ?`, convID, limit).Iter()
	msgs := make([]model.Message, 0, limit)
	var m model.Message
	for iter.Scan(&m.MsgID, &m.ConvID, &m.Seq, &m.FromUID, &m.ToID, &m.Type, &m.Content, &m.Extra,
		&m.ClientMsgID, &m.Status, &m.CreatedAt) {
		msgs = append(msgs, m)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	// 主表按 seq 倒序存储，返回时转为正序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (r *scyllaMessageRepo) UpdateStatus(msgID string, status int) error {
	m, err := r.GetByMsgID(msgID)
	if err != nil {
		return err
	}
	batch := r.sess.NewBatch(gocql.LoggedBatch)
	batch.Query(`UPDATE `+scyllaTableMessage+` SET status = ? WHERE conv_id = ? AND seq = ?`,
		status, m.ConvID, m.Seq)
	batch.Query(`UPDATE `+scyllaTableByID+` SET status = ? WHERE msg_id = ?`, status, msgID)
	batch.Query(`UPDATE `+scyllaTableTimeline+` SET status = ? WHERE bucket = ? AND created_at = ? AND msg_id = ?`,
		status, bucketOf(m.CreatedAt), m.CreatedAt, msgID)
	return r.sess.ExecuteBatch(batch)
}

func (r *scyllaMessageRepo) Search(keyword, convID string, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	var candidates []model.Message
	if convID != "" {
		list, err := r.List(convID, 500)
		if err != nil {
			return nil, err
		}
		candidates = list
	} else {
		// 无会话条件时，在最近 N 个月的审计时间线中做内存过滤（Scylla 不支持 LIKE）
		for _, b := range recentBuckets(scyllaAuditMonthSpan) {
			list, err := r.timelinePage(b, 2000)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, list...)
		}
	}
	out := make([]model.Message, 0, limit)
	for _, m := range candidates {
		if m.Status == 1 && strings.Contains(m.Content, keyword) {
			out = append(out, m)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *scyllaMessageRepo) Count() (int64, error) {
	var total int64
	for _, b := range recentBuckets(scyllaAuditMonthSpan) {
		var n int64
		if err := r.sess.Query(`SELECT count(*) FROM `+scyllaTableTimeline+` WHERE bucket = ?`, b).Scan(&n); err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (r *scyllaMessageRepo) Page(offset, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 20
	}
	var all []model.Message
	for _, b := range recentBuckets(scyllaAuditMonthSpan) {
		list, err := r.timelinePage(b, offset+limit)
		if err != nil {
			return nil, err
		}
		all = append(all, list...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt == all[j].CreatedAt {
			return all[i].MsgID > all[j].MsgID
		}
		return all[i].CreatedAt > all[j].CreatedAt
	})
	if offset >= len(all) {
		return []model.Message{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

// MaxSeq 取会话内最大 seq（聚簇列 seq 倒序，LIMIT 1 即最大值）
func (r *scyllaMessageRepo) MaxSeq(convID string) (int64, error) {
	var seq int64
	err := r.sess.Query(`SELECT seq FROM `+scyllaTableMessage+` WHERE conv_id = ? LIMIT 1`, convID).Scan(&seq)
	if err == gocql.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return seq, nil
}

func (r *scyllaMessageRepo) timelinePage(bucket string, limit int) ([]model.Message, error) {
	iter := r.sess.Query(`SELECT msg_id, conv_id, seq, from_uid, to_id, msg_type, content, extra,
		client_msg_id, status, created_at FROM `+scyllaTableTimeline+` WHERE bucket = ? LIMIT ?`, bucket, limit).Iter()
	list := make([]model.Message, 0, limit)
	var m model.Message
	for iter.Scan(&m.MsgID, &m.ConvID, &m.Seq, &m.FromUID, &m.ToID, &m.Type, &m.Content, &m.Extra,
		&m.ClientMsgID, &m.Status, &m.CreatedAt) {
		list = append(list, m)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return list, nil
}
