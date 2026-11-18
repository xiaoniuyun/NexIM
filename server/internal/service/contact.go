package service

import (
	"nexim/server/internal/model"
	"nexim/server/internal/store"

	"gorm.io/gorm"
)

type ContactService struct{ S *store.Store }

func NewContact(s *store.Store) *ContactService { return &ContactService{S: s} }

type ContactVO struct {
	UID      string `json:"uid"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	DeptID   int64  `json:"deptId"`
	DeptName string `json:"deptName"`
	Online   bool   `json:"online"`
}

func (c *ContactService) Friends(uid string) []ContactVO {
	var friends []model.Friend
	c.S.DB.Where("uid = ? AND status = 1", uid).Find(&friends)
	out := []ContactVO{}
	for _, f := range friends {
		if u := c.userByUID(f.FriendUID); u != nil {
			out = append(out, *u)
		}
	}
	return out
}

func (c *ContactService) Users(keyword string) []ContactVO {
	var users []model.User
	q := c.S.DB.Where("status = 1")
	if keyword != "" {
		q = q.Where("username LIKE ? OR nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	q.Limit(50).Find(&users)
	out := []ContactVO{}
	for i := range users {
		out = append(out, c.toVO(&users[i]))
	}
	return out
}

func (c *ContactService) Depts() []model.Dept {
	var ds []model.Dept
	c.S.DB.Order("path, id").Find(&ds)
	return ds
}

func (c *ContactService) DeptMembers(deptID int64) []ContactVO {
	var users []model.User
	if deptID == 0 {
		c.S.DB.Where("status = 1").Find(&users)
	} else {
		c.S.DB.Where("status = 1 AND dept_id = ?", deptID).Find(&users)
	}
	out := []ContactVO{}
	for i := range users {
		out = append(out, c.toVO(&users[i]))
	}
	return out
}

func (c *ContactService) ApplyFriend(fromUID, toUID, message string) error {
	var cnt int64
	c.S.DB.Where("(uid = ? AND friend_uid = ?) OR (uid = ? AND friend_uid = ?)", fromUID, toUID, toUID, fromUID).Count(&cnt)
	if cnt > 0 {
		return gorm.ErrDuplicatedKey
	}
	return c.S.DB.Create(&model.FriendApply{FromUID: fromUID, ToUID: toUID, Message: message, Status: 0, CreatedAt: model.Now()}).Error
}

func (c *ContactService) Applies(uid string) []model.FriendApply {
	var list []model.FriendApply
	c.S.DB.Where("to_uid = ? AND status = 0", uid).Order("created_at desc").Find(&list)
	return list
}

func (c *ContactService) HandleApply(uid string, applyID int64, accept bool) error {
	var a model.FriendApply
	if err := c.S.DB.Where("id = ? AND to_uid = ?", applyID, uid).First(&a).Error; err != nil {
		return err
	}
	status := 2
	if accept {
		status = 1
		now := model.Now()
		c.S.DB.Create(&model.Friend{UID: uid, FriendUID: a.FromUID, Status: 1, CreatedAt: now})
		c.S.DB.Create(&model.Friend{UID: a.FromUID, FriendUID: uid, Status: 1, CreatedAt: now})
	}
	return c.S.DB.Model(&a).Update("status", status).Error
}

func (c *ContactService) userByUID(uid string) *ContactVO {
	var u model.User
	if c.S.DB.Where("uid = ?", uid).First(&u).Error != nil {
		return nil
	}
	vo := c.toVO(&u)
	return &vo
}

func (c *ContactService) toVO(u *model.User) ContactVO {
	var d model.Dept
	deptName := ""
	if c.S.DB.Where("id = ?", u.DeptID).First(&d).Error == nil {
		deptName = d.Name
	}
	online := false
	if _, e := c.S.RDB.Get(c.S.Ctx, "online:"+u.UID).Result(); e == nil {
		online = true
	}
	return ContactVO{UID: u.UID, Username: u.Username, Nickname: u.Nickname, Avatar: u.Avatar, DeptID: u.DeptID, DeptName: deptName, Online: online}
}
