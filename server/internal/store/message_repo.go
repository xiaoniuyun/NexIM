package store

import (
	"nexim/server/internal/model"

	"gorm.io/gorm"
)

// MessageRepo 消息存储抽象：lite 形态落 MySQL，standard 形态落 ScyllaDB
type MessageRepo interface {
	Create(m *model.Message) error
	GetByMsgID(msgID string) (*model.Message, error)
	List(convID string, limit int) ([]model.Message, error)
	UpdateStatus(msgID string, status int) error
	Search(keyword, convID string, limit int) ([]model.Message, error)
	Count() (int64, error)
	Page(offset, limit int) ([]model.Message, error)
	MaxSeq(convID string) (int64, error)
	Driver() string
}

// ============ MySQL 实现（lite 形态 / 兼容既有数据） ============

type mysqlMessageRepo struct{ db *gorm.DB }

func NewMySQLMessageRepo(db *gorm.DB) MessageRepo { return &mysqlMessageRepo{db: db} }

func (r *mysqlMessageRepo) Driver() string { return "mysql" }

func (r *mysqlMessageRepo) Create(m *model.Message) error { return r.db.Create(m).Error }

func (r *mysqlMessageRepo) GetByMsgID(msgID string) (*model.Message, error) {
	var m model.Message
	if err := r.db.Where("msg_id = ?", msgID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *mysqlMessageRepo) List(convID string, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 200
	}
	var msgs []model.Message
	if err := r.db.Where("conv_id = ? AND status IN (1,2)", convID).
		Order("seq desc").Limit(limit).Find(&msgs).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (r *mysqlMessageRepo) UpdateStatus(msgID string, status int) error {
	return r.db.Model(&model.Message{}).Where("msg_id = ?", msgID).Update("status", status).Error
}

func (r *mysqlMessageRepo) Search(keyword, convID string, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	q := r.db.Model(&model.Message{}).Where("content LIKE ? AND status = 1", "%"+keyword+"%")
	if convID != "" {
		q = q.Where("conv_id = ?", convID)
	}
	var list []model.Message
	if err := q.Order("id desc").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *mysqlMessageRepo) Count() (int64, error) {
	var total int64
	err := r.db.Model(&model.Message{}).Count(&total).Error
	return total, err
}

func (r *mysqlMessageRepo) Page(offset, limit int) ([]model.Message, error) {
	var list []model.Message
	if err := r.db.Order("id desc").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *mysqlMessageRepo) MaxSeq(convID string) (int64, error) {
	var seq int64
	row := r.db.Model(&model.Message{}).Where("conv_id = ?", convID).
		Select("COALESCE(MAX(seq),0)").Row()
	if err := row.Scan(&seq); err != nil {
		return 0, err
	}
	return seq, nil
}
