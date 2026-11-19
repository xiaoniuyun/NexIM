package service

import (
	"encoding/json"
	"fmt"

	"nexim/server/internal/model"
	"nexim/server/internal/store"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminService struct{ S *store.Store }

func NewAdmin(s *store.Store) *AdminService { return &AdminService{S: s} }

func (a *AdminService) Dashboard() map[string]interface{} {
	var online int64
	a.S.DB.Model(&model.User{}).Where("status = 1").Count(&online)
	msgCount, _ := a.S.Msg.Count()
	var groups int64
	a.S.DB.Model(&model.Group{}).Where("status = 1").Count(&groups)
	var users int64
	a.S.DB.Model(&model.User{}).Where("status = 1").Count(&users)
	return map[string]interface{}{
		"onlineUsers": online, "todayMessages": msgCount,
		"dau": users, "activeGroups": groups,
		"trend": []map[string]interface{}{
			{"label": "00:00", "value": 120}, {"label": "04:00", "value": 60},
			{"label": "08:00", "value": 320}, {"label": "12:00", "value": 480},
			{"label": "16:00", "value": 410}, {"label": "20:00", "value": 290},
		},
	}
}

func (a *AdminService) UserList(page, size int) map[string]interface{} {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var total int64
	a.S.DB.Model(&model.User{}).Count(&total)
	var users []model.User
	a.S.DB.Order("id").Offset((page - 1) * size).Limit(size).Find(&users)
	return map[string]interface{}{"total": total, "list": users, "page": page, "size": size}
}

func (a *AdminService) CreateUser(username, password, nickname string, deptID int64, phone, title string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("账号和密码不能为空")
	}
	var cnt int64
	a.S.DB.Where("username = ?", username).Count(&cnt)
	if cnt > 0 {
		return nil, fmt.Errorf("账号已存在")
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	now := model.Now()
	u := model.User{UID: fmt.Sprintf("u_%d", now), Username: username, Nickname: nickname,
		PasswordHash: string(h), Avatar: "#4f6ef7", DeptID: deptID, Phone: phone, Title: title,
		Status: 1, CreatedAt: now, UpdatedAt: now}
	if nickname == "" {
		u.Nickname = username
	}
	if err := a.S.DB.Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (a *AdminService) UpdateUser(id int64, updates map[string]interface{}) error {
	if pwd, ok := updates["password"]; ok {
		h, _ := bcrypt.GenerateFromPassword([]byte(fmt.Sprint(pwd)), bcrypt.DefaultCost)
		delete(updates, "password")
		updates["password_hash"] = string(h)
	}
	updates["updated_at"] = model.Now()
	return a.S.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (a *AdminService) DeleteUser(id int64) error {
	return a.S.DB.Model(&model.User{}).Where("id = ?", id).Update("status", 3).Error
}

func (a *AdminService) ResetPassword(id int64) (string, error) {
	pwd := "123456"
	h, _ := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	err := a.S.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"password_hash": string(h), "updated_at": model.Now(),
	}).Error
	return pwd, err
}

func (a *AdminService) Roles() []model.Role {
	var roles []model.Role
	a.S.DB.Find(&roles)
	return roles
}

func (a *AdminService) GrantRole(uid string, roleID int64) error {
	var cnt int64
	a.S.DB.Model(&model.UserRole{}).Where("uid = ? AND role_id = ?", uid, roleID).Count(&cnt)
	if cnt > 0 {
		return nil
	}
	return a.S.DB.Create(&model.UserRole{UID: uid, RoleID: roleID}).Error
}

func (a *AdminService) Permissions(uid string) []string {
	var roles []model.Role
	a.S.DB.Joins("JOIN sys_user_role ur ON ur.role_id = sys_role.id").Where("ur.uid = ?", uid).Find(&roles)
	set := map[string]bool{}
	for _, r := range roles {
		var perms []string
		_ = json.Unmarshal([]byte(r.Permissions), &perms)
		for _, p := range perms {
			set[p] = true
		}
	}
	out := []string{}
	for p := range set {
		out = append(out, p)
	}
	return out
}

func (a *AdminService) AuditList(page, size int) map[string]interface{} {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var total int64
	a.S.DB.Model(&model.AuditLog{}).Count(&total)
	var list []model.AuditLog
	a.S.DB.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list)
	return map[string]interface{}{"total": total, "list": list}
}

func (a *AdminService) Audit(uid, action, target, detail string) {
	a.S.DB.Create(&model.AuditLog{UID: uid, Action: action, Target: target, Detail: detail, CreatedAt: model.Now()})
}

func (a *AdminService) MessageAudit(page, size int) map[string]interface{} {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	total, _ := a.S.Msg.Count()
	list, err := a.S.Msg.Page((page-1)*size, size)
	if err != nil {
		list = []model.Message{}
	}
	return map[string]interface{}{"total": total, "list": list}
}

func (a *AdminService) Notices() []model.Notice {
	var list []model.Notice
	a.S.DB.Order("id desc").Find(&list)
	return list
}

func (a *AdminService) CreateNotice(title, content, author string) (*model.Notice, error) {
	if title == "" || content == "" {
		return nil, fmt.Errorf("标题和内容不能为空")
	}
	now := model.Now()
	n := model.Notice{Title: title, Content: content, Author: author, Status: 1, CreatedAt: now, UpdatedAt: now}
	if err := a.S.DB.Create(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (a *AdminService) UpdateNotice(id int64, updates map[string]interface{}) error {
	updates["updated_at"] = model.Now()
	return a.S.DB.Model(&model.Notice{}).Where("id = ?", id).Updates(updates).Error
}

func (a *AdminService) SensitiveWords() []string {
	val, err := a.S.RDB.SMembers(a.S.Ctx, "sensitive:words:v1").Result()
	if err != nil {
		return []string{}
	}
	return val
}

func (a *AdminService) AddSensitiveWord(word string) error {
	if word == "" {
		return gorm.ErrRecordNotFound
	}
	return a.S.RDB.SAdd(a.S.Ctx, "sensitive:words:v1", word).Err()
}
