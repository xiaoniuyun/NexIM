package service

import (
	"nexim/server/internal/model"
	"nexim/server/internal/store"
)

type BlacklistService struct {
	S *store.Store
}

func NewBlacklist(s *store.Store) *BlacklistService { return &BlacklistService{S: s} }

// BlockedMap 返回某用户拉黑的 UID 集合
func (m *MessageService) BlockedMap(uid string) map[string]bool {
	out := map[string]bool{}
	var rows []model.Blacklist
	m.S.DB.Where("uid = ?", uid).Find(&rows)
	for _, r := range rows {
		out[r.BlockedUID] = true
	}
	return out
}

// IsBlocked 判断 owner 是否已拉黑 other
func (m *MessageService) IsBlocked(owner, other string) bool {
	if owner == "" || other == "" {
		return false
	}
	var n int64
	m.S.DB.Model(&model.Blacklist{}).Where("uid = ? AND blocked_uid = ?", owner, other).Count(&n)
	return n > 0
}

// List 黑名单列表（带昵称）
func (b *BlacklistService) List(uid string) []map[string]interface{} {
	var rows []model.Blacklist
	b.S.DB.Where("uid = ?", uid).Order("created_at desc").Find(&rows)
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		item := map[string]interface{}{
			"uid": r.BlockedUID, "createdAt": r.CreatedAt,
		}
		var u model.User
		if b.S.DB.Where("uid = ?", r.BlockedUID).First(&u).Error == nil {
			item["nickname"] = u.Nickname
			item["username"] = u.Username
			item["avatar"] = u.Avatar
		}
		out = append(out, item)
	}
	return out
}

func (b *BlacklistService) Add(uid, blockedUID string) error {
	if uid == blockedUID {
		return nil
	}
	var n int64
	b.S.DB.Model(&model.Blacklist{}).Where("uid = ? AND blocked_uid = ?", uid, blockedUID).Count(&n)
	if n > 0 {
		return nil
	}
	return b.S.DB.Create(&model.Blacklist{
		UID: uid, BlockedUID: blockedUID, CreatedAt: model.Now(),
	}).Error
}

func (b *BlacklistService) Remove(uid, blockedUID string) error {
	return b.S.DB.Where("uid = ? AND blocked_uid = ?", uid, blockedUID).Delete(&model.Blacklist{}).Error
}
