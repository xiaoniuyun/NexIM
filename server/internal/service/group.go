package service

import (
	"fmt"

	"nexim/server/internal/model"
	"nexim/server/internal/store"

	"gorm.io/gorm"
)

type GroupService struct{ S *store.Store }

func NewGroup(s *store.Store) *GroupService { return &GroupService{S: s} }

// 公司全员群：所有启用用户默认加入，新用户创建/注册后自动入群
const (
	CompanyGroupID   = "g_company"
	CompanyGroupName = "公司全员群"
)

// EnsureCompanyGroup 幂等创建公司全员群，并把组织架构内的启用用户纳入（启动时调用）
// 外部联系人（user_type = 1 或未分配部门）不属于公司全员群
func (g *GroupService) EnsureCompanyGroup() (*model.Group, error) {
	now := model.Now()
	var grp model.Group
	if err := g.S.DB.Where("group_id = ?", CompanyGroupID).First(&grp).Error; err != nil {
		grp = model.Group{
			GroupID: CompanyGroupID, Name: CompanyGroupName, OwnerUID: g.companyOwner(),
			Avatar: "#2f7df6", Notice: "公司全员群：组织架构内的成员自动加入，可在此与全公司同事交流",
			Status: 1, CreatedAt: now,
		}
		if err := g.S.DB.Create(&grp).Error; err != nil {
			return nil, err
		}
	}
	var uids []string
	g.S.DB.Model(&model.User{}).Where("status = 1 AND user_type = 0 AND dept_id > 0").Order("id").Pluck("uid", &uids)
	for _, uid := range uids {
		g.joinCompanyMember(uid, now)
	}
	// 清理历史上被误加入公司群的外部联系人及其公司群会话
	var extUIDs []string
	g.S.DB.Model(&model.User{}).Where("user_type = 1 OR dept_id = 0").Pluck("uid", &extUIDs)
	if len(extUIDs) > 0 {
		g.S.DB.Where("group_id = ? AND uid IN ?", CompanyGroupID, extUIDs).Delete(&model.GroupMember{})
		g.S.DB.Where("conv_id = ? AND owner_uid IN ?", "conv_"+CompanyGroupID, extUIDs).Delete(&model.Conversation{})
	}
	// 存量用户补会话，登录后即可在会话列表看到公司群
	g.ensureConversations(CompanyGroupID, grp.Name, uids, now)
	g.syncCompanyMemberCount()
	g.S.DB.Where("group_id = ?", CompanyGroupID).First(&grp)
	return &grp, nil
}

// JoinCompanyGroup 新用户加入公司全员群（幂等），仅后台创建用户（加入组织架构）后调用
func (g *GroupService) JoinCompanyGroup(uid string) error {
	if uid == "" {
		return nil
	}
	var grp model.Group
	if err := g.S.DB.Where("group_id = ?", CompanyGroupID).First(&grp).Error; err != nil {
		created, err := g.EnsureCompanyGroup()
		if err != nil {
			return err
		}
		grp = *created
	}
	now := model.Now()
	if g.joinCompanyMember(uid, now) {
		g.syncCompanyMemberCount()
	}
	g.ensureConversations(CompanyGroupID, grp.Name, []string{uid}, now)
	return nil
}

// joinCompanyMember 写入群成员（已存在则跳过），返回是否新增
func (g *GroupService) joinCompanyMember(uid string, now int64) bool {
	var cnt int64
	g.S.DB.Model(&model.GroupMember{}).Where("group_id = ? AND uid = ?", CompanyGroupID, uid).Count(&cnt)
	if cnt > 0 {
		return false
	}
	role := "member"
	if uid == g.companyOwner() {
		role = "owner"
	}
	return g.S.DB.Create(&model.GroupMember{GroupID: CompanyGroupID, UID: uid, Role: role, JoinedAt: now}).Error == nil
}

// syncCompanyMemberCount 以群成员表为准刷新人数
func (g *GroupService) syncCompanyMemberCount() {
	var cnt int64
	g.S.DB.Model(&model.GroupMember{}).Where("group_id = ?", CompanyGroupID).Count(&cnt)
	g.S.DB.Model(&model.Group{}).Where("group_id = ?", CompanyGroupID).Update("member_count", cnt)
}

// companyOwner 公司群群主：优先取超级管理员，其次最早创建的用户
func (g *GroupService) companyOwner() string {
	var uid string
	g.S.DB.Table("sys_user_role AS ur").Joins("JOIN sys_role AS r ON r.id = ur.role_id").
		Where("r.code = ?", "SUPER_ADMIN").Order("ur.uid").Limit(1).Pluck("ur.uid", &uid)
	if uid != "" {
		return uid
	}
	var u model.User
	if g.S.DB.Order("id").First(&u).Error == nil {
		return u.UID
	}
	return ""
}

func (g *GroupService) MyGroups(uid string) []model.Group {
	var ids []string
	g.S.DB.Model(&model.GroupMember{}).Where("uid = ?", uid).Pluck("group_id", &ids)
	if len(ids) == 0 {
		return []model.Group{}
	}
	// 外部联系人不可见公司全员群（内部组织架构群）
	if g.isExternal(uid) {
		filtered := ids[:0]
		for _, id := range ids {
			if id != CompanyGroupID {
				filtered = append(filtered, id)
			}
		}
		ids = filtered
		if len(ids) == 0 {
			return []model.Group{}
		}
	}
	var groups []model.Group
	g.S.DB.Where("group_id IN ? AND status = 1", ids).Find(&groups)
	return groups
}

// isExternal 用户是否为外部联系人
func (g *GroupService) isExternal(uid string) bool {
	var u model.User
	if g.S.DB.Select("user_type, dept_id").Where("uid = ?", uid).First(&u).Error != nil {
		return false
	}
	return IsExternalUser(&u)
}

func (g *GroupService) Detail(groupID string) *model.Group {
	var gr model.Group
	if g.S.DB.Where("group_id = ?", groupID).First(&gr).Error != nil {
		return nil
	}
	return &gr
}

// RoleOf 返回成员在群内的角色：owner / member，非成员返回空串
func (g *GroupService) RoleOf(groupID, uid string) string {
	var m model.GroupMember
	if g.S.DB.Where("group_id = ? AND uid = ?", groupID, uid).First(&m).Error != nil {
		return ""
	}
	return m.Role
}

// IsMember 是否为群成员
func (g *GroupService) IsMember(groupID, uid string) bool { return g.RoleOf(groupID, uid) != "" }

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
	// 建群即写入所有成员的会话记录，成员登录后无需等待第一条消息即可在会话列表看到群聊
	g.ensureConversations(groupID, name, append([]string{uid}, memberUIDs...), now)
	return &grp, nil
}

func (g *GroupService) ensureConversations(groupID, name string, uids []string, now int64) {
	convID := "conv_" + groupID
	seen := map[string]bool{}
	for _, muid := range uids {
		if muid == "" || seen[muid] {
			continue
		}
		seen[muid] = true
		var cnt int64
		g.S.DB.Model(&model.Conversation{}).Where("conv_id = ? AND owner_uid = ?", convID, muid).Count(&cnt)
		if cnt > 0 {
			continue
		}
		g.S.DB.Create(&model.Conversation{
			ConvID: convID, ConvType: 2, OwnerUID: muid, PeerID: groupID,
			PeerName: name, LastMsg: "群聊已创建", LastMsgAt: now,
		})
	}
}

func (g *GroupService) Invite(groupID string, uids []string) error {
	now := model.Now()
	var grp model.Group
	g.S.DB.Where("group_id = ?", groupID).First(&grp)
	for _, uid := range uids {
		var cnt int64
		g.S.DB.Model(&model.GroupMember{}).Where("group_id = ? AND uid = ?", groupID, uid).Count(&cnt)
		if cnt == 0 {
			g.S.DB.Create(&model.GroupMember{GroupID: groupID, UID: uid, Role: "member", JoinedAt: now})
			g.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("member_count", gorm.Expr("member_count + 1"))
		}
	}
	g.ensureConversations(groupID, grp.Name, uids, now)
	return nil
}

func (g *GroupService) RemoveMember(groupID, uid string) error {
	if err := g.S.DB.Where("group_id = ? AND uid = ?", groupID, uid).Delete(&model.GroupMember{}).Error; err != nil {
		return err
	}
	g.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("member_count", gorm.Expr("member_count - 1"))
	g.S.DB.Where("conv_id = ? AND owner_uid = ?", "conv_"+groupID, uid).Delete(&model.Conversation{})
	return nil
}

func (g *GroupService) Dissolve(groupID string) error {
	if err := g.S.DB.Model(&model.Group{}).Where("group_id = ?", groupID).Update("status", 3).Error; err != nil {
		return err
	}
	g.S.DB.Where("conv_id = ?", "conv_"+groupID).Delete(&model.Conversation{})
	return nil
}
