package service

import (
	"fmt"

	"nexim/server/internal/model"
	"nexim/server/internal/store"

	"gorm.io/gorm"
)

type GroupService struct{ S *store.Store }

func NewGroup(s *store.Store) *GroupService { return &GroupService{S: s} }

func (g *GroupService) MyGroups(uid string) []model.Group {
	var ids []string
	g.S.DB.Model(&model.GroupMember{}).Where("uid = ?", uid).Pluck("group_id", &ids)
	if len(ids) == 0 {
		return []model.Group{}
	}
	var groups []model.Group
	g.S.DB.Where("group_id IN ? AND status = 1", ids).Find(&groups)
	return groups
}

func (g *GroupService) Detail(groupID string) *model.Group {
	var gr model.Group
	if g.S.DB.Where("group_id = ?", groupID).First(&gr).Error != nil {
		return nil
	}
	return &gr
}

func (g *GroupService) Members(groupID string) []map[string]interface{} {
	var ms []model.GroupMember
	g.S.DB.Where("group_id = ?", groupID).Find(&ms)
	out := []map[string]interface{}{}
	for _, m := range ms {
		var u model.User
		if g.S.DB.Where("uid = ?", m.UID).First(&u).Error == nil {
			out = append(out, map[string]interface{}{
				"uid": u.UID, "nickname": u.Nickname, "avatar": u.Avatar, "role": m.Role,
			})
		}
	}
	return out
}

func (g *GroupService) Create(uid, name string, memberUIDs []string) (*model.Group, error) {
	if name == "" {
		return nil, fmt.Errorf("群名称不能为空")
	}
	now := model.Now()
	groupID := fmt.Sprintf("g_%d", now)
	grp := model.Group{GroupID: groupID, Name: name, OwnerUID: uid, Avatar: "#4f6ef7", MemberCount: len(memberUIDs) + 1, Status: 1, CreatedAt: now}
	if err := g.S.DB.Create(&grp).Error; err != nil {
		return nil, err
	}
	g.S.DB.Create(&model.GroupMember{GroupID: groupID, UID: uid, Role: "owner", JoinedAt: now})
	for _, muid := range memberUIDs {
		g.S.DB.Create(&model.GroupMember{GroupID: groupID, UID: muid, Role: "member", JoinedAt: now})
	}
	return &grp, nil
}

func (g *GroupService) Invite(groupID string, uids []string) error {
	now := model.Now()
	for _, uid := range uids {
		var cnt int64
		g.S.DB.Model(&model.GroupMember{}).Where("group_id = ? AND uid = ?", groupID, uid).Count(&cnt)
		if cnt == 0 {
			g.S.DB.Create(&model.GroupMember{GroupID: groupID, UID: uid, Role: "member", JoinedAt: now})
			g.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("member_count", gorm.Expr("member_count + 1"))
		}
	}
	return nil
}

func (g *GroupService) RemoveMember(groupID, uid string) error {
	if err := g.S.DB.Where("group_id = ? AND uid = ?", groupID, uid).Delete(&model.GroupMember{}).Error; err != nil {
		return err
	}
	g.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("member_count", gorm.Expr("member_count - 1"))
	return nil
}

func (g *GroupService) Dissolve(groupID string) error {
	return g.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("status", 2).Error
}
