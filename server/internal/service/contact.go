package service

import (
	"strconv"
	"strings"

	"nexim/server/internal/model"
	"nexim/server/internal/store"

	"gorm.io/gorm"
)

type ContactService struct{ S *store.Store }

func NewContact(s *store.Store) *ContactService { return &ContactService{S: s} }

type ContactVO struct {
	ID       int64  `json:"id"`
	UID      string `json:"uid"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Title    string `json:"title"`
	Avatar   string `json:"avatar"`
	Signature string `json:"signature"`
	Gender    int    `json:"gender"`
	DeptID    int64  `json:"deptId"`
	DeptName string `json:"deptName"`
	// OrgName 所在组织全路径（总部 / 研发中心）
	OrgName string `json:"orgName"`
	// External 企业外部用户（未归属任何部门），与企业内部用户单聊前需先添加好友
	External bool `json:"external"`
	Online   bool `json:"online"`
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

// IsExternalUser 外部联系人：注册页注册的账号（user_type=1），或历史未归属部门的账号
func IsExternalUser(u *model.User) bool { return u.UserType == 1 || u.DeptID == 0 }

// IsExternal 当前用户是否外部联系人（组织架构权限以此为准）
func (c *ContactService) IsExternal(uid string) bool {
	var u model.User
	if c.S.DB.Select("user_type, dept_id").Where("uid = ?", uid).First(&u).Error != nil {
		return false
	}
	return IsExternalUser(&u)
}

// Users 联系人搜索：外部联系人只能检索其他外部联系人，无法查询组织架构内部人员
func (c *ContactService) Users(keyword, requesterUID string) []ContactVO {
	var users []model.User
	q := c.S.DB.Where("status = 1")
	if requesterUID != "" && c.IsExternal(requesterUID) {
		q = q.Where("(user_type = 1 OR dept_id = 0)")
	}
	if keyword != "" {
		q = q.Where("(username LIKE ? OR nickname LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
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
	c.S.DB.Where("status = 1").Order("path, sort, id").Find(&ds)
	return ds
}

// DeptMembers 支持按物化路径包含所有下级部门成员（数据范围「本部门及下级」）
// 组织架构仅展示企业内部用户，未分配部门的外部用户不出现在通讯录中
func (c *ContactService) DeptMembers(deptID int64) []ContactVO {
	var users []model.User
	if deptID == 0 {
		c.S.DB.Where("status = 1 AND dept_id > 0 AND user_type = 0").Find(&users)
	} else {
		ids := []int64{deptID}
		var d model.Dept
		if c.S.DB.Where("id = ?", deptID).First(&d).Error == nil && d.Path != "" {
			var subs []model.Dept
			c.S.DB.Where("path LIKE ? AND id <> ?", d.Path+"%", deptID).Find(&subs)
			for _, s := range subs {
				ids = append(ids, s.ID)
			}
		}
		c.S.DB.Where("status = 1 AND dept_id IN ? AND user_type = 0", ids).Find(&users)
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

// ApplyVO 好友申请，附带申请人资料便于直接展示
type ApplyVO struct {
	ID        int64  `json:"id"`
	FromUID   string `json:"fromUid"`
	ToUID     string `json:"toUid"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"createdAt"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
}

func (c *ContactService) Applies(uid string) []ApplyVO {
	var list []model.FriendApply
	c.S.DB.Where("to_uid = ? AND status = 0", uid).Order("created_at desc").Find(&list)
	out := make([]ApplyVO, 0, len(list))
	for _, a := range list {
		vo := ApplyVO{ID: a.ID, FromUID: a.FromUID, ToUID: a.ToUID,
			Message: a.Message, Status: a.Status, CreatedAt: a.CreatedAt}
		if u := c.userByUID(a.FromUID); u != nil {
			vo.Username, vo.Nickname, vo.Avatar = u.Username, u.Nickname, u.Avatar
		}
		out = append(out, vo)
	}
	return out
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
	return ContactVO{ID: u.ID, UID: u.UID, Username: u.Username, Nickname: u.Nickname,
		Phone: u.Phone, Email: u.Email, Title: u.Title, Avatar: u.Avatar, Signature: u.Signature, Gender: u.Gender,
		DeptID: u.DeptID, DeptName: deptName, OrgName: c.OrgName(u.DeptID), External: IsExternalUser(u), Online: online}
}

// OrgName 组织全路径名称：按部门物化路径逐级取名称，如「总部 / 研发中心」
// 未归属部门（外部联系人）返回空串
func (c *ContactService) OrgName(deptID int64) string {
	if deptID <= 0 {
		return ""
	}
	var d model.Dept
	if c.S.DB.Where("id = ?", deptID).First(&d).Error != nil {
		return ""
	}
	if d.Path == "" {
		return d.Name
	}
	names := []string{}
	for _, seg := range strings.Split(d.Path, "/") {
		id, err := strconv.ParseInt(seg, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		var one model.Dept
		if c.S.DB.Select("name").Where("id = ?", id).First(&one).Error == nil && one.Name != "" {
			names = append(names, one.Name)
		}
	}
	if len(names) == 0 {
		return d.Name
	}
	return strings.Join(names, " / ")
}
