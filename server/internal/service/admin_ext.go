package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"nexim/server/internal/model"

	"gorm.io/gorm"
)

// UserQuery 用户列表查询条件（服务端筛选 + 分页，不再由前端截断）
type UserQuery struct {
	Keyword    string
	Status     string // "" 全部 | "1" 正常 | "0" 禁用
	DeptID     int64
	IncludeSub bool
	Page       int
	Size       int
}

// UserVO 用户列表项：附带部门名称与真实角色绑定
type UserVO struct {
	model.User
	DeptName  string   `json:"deptName"`
	RoleIDs   []int64  `json:"roleIds"`
	RoleNames []string `json:"roleNames"`
	RoleCodes []string `json:"roleCodes"`
}

// RoleVO 角色配置项：权限解析为数组，并带成员数
type RoleVO struct {
	ID          int64    `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	DataScope   string   `json:"dataScope"`
	Version     int      `json:"version"`
	MemberCount int64    `json:"memberCount"`
	BuiltIn     bool     `json:"builtIn"`
}

var builtInRoles = map[string]bool{
	"SUPER_ADMIN": true, "USER": true, "DEPT_ADMIN": true, "OPS_ADMIN": true, "AUDITOR": true,
}

func isBuiltIn(code string) bool { return builtInRoles[strings.ToUpper(code)] }

func normalizeScope(scope string) string {
	if scope == "" {
		return "self"
	}
	switch scope {
	case "all", "deptAndSub", "dept", "self", "custom":
		return scope
	// 兼容历史数字取值
	case "1":
		return "all"
	case "2":
		return "deptAndSub"
	case "3":
		return "dept"
	case "4":
		return "self"
	case "5":
		return "custom"
	default:
		return ""
	}
}

// validPerms 校验权限码在目录中登记（* 为超管通配）
func validPerms(perms []string) error {
	known := map[string]bool{"*": true}
	for _, p := range AllPermCodes() {
		known[p] = true
	}
	for _, p := range perms {
		if !known[p] {
			return fmt.Errorf("未知权限码 %s", p)
		}
	}
	return nil
}

// userRows 按条件查询用户（含部门名、角色绑定）
func (a *AdminService) userRows(q UserQuery) ([]UserVO, error) {
	db := a.S.DB.Model(&model.User{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("(username LIKE ? OR nickname LIKE ? OR phone LIKE ? OR title LIKE ?)", like, like, like, like)
	}
	switch q.Status {
	case "1":
		db = db.Where("status = 1")
	case "0":
		db = db.Where("status <> 1")
	}
	if q.DeptID > 0 {
		ids := []int64{q.DeptID}
		if q.IncludeSub {
			var parent model.Dept
			if a.S.DB.Where("id = ?", q.DeptID).First(&parent).Error == nil && parent.Path != "" {
				var children []model.Dept
				a.S.DB.Where("path LIKE ? AND id <> ?", parent.Path+"%", parent.ID).Find(&children)
				for _, c := range children {
					ids = append(ids, c.ID)
				}
			}
		}
		db = db.Where("dept_id IN ?", ids)
	}
	var users []model.User
	if err := db.Order("id").Find(&users).Error; err != nil {
		return nil, err
	}

	// 一次性取出部门与角色，避免 N+1 查询
	deptNames := map[int64]string{}
	var depts []model.Dept
	a.S.DB.Find(&depts)
	for _, d := range depts {
		deptNames[d.ID] = d.Name
	}
	type roleRow struct {
		UID  string `gorm:"column:uid"`
		ID   int64  `gorm:"column:role_id"`
		Code string `gorm:"column:code"`
		Name string `gorm:"column:name"`
	}
	rows := []roleRow{}
	if len(users) > 0 {
		uids := make([]string, 0, len(users))
		for _, u := range users {
			uids = append(uids, u.UID)
		}
		a.S.DB.Raw(`SELECT ur.uid, r.id AS role_id, r.code, r.name
			FROM sys_user_role ur JOIN sys_role r ON r.id = ur.role_id
			WHERE ur.uid IN ? ORDER BY r.id`, uids).Scan(&rows)
	}
	rolesByUID := map[string][]roleRow{}
	for _, r := range rows {
		rolesByUID[r.UID] = append(rolesByUID[r.UID], r)
	}
	out := make([]UserVO, 0, len(users))
	for _, u := range users {
		vo := UserVO{User: u, DeptName: deptNames[u.DeptID],
			RoleIDs: []int64{}, RoleNames: []string{}, RoleCodes: []string{}}
		for _, r := range rolesByUID[u.UID] {
			vo.RoleIDs = append(vo.RoleIDs, r.ID)
			vo.RoleNames = append(vo.RoleNames, r.Name)
			vo.RoleCodes = append(vo.RoleCodes, r.Code)
		}
		out = append(out, vo)
	}
	return out, nil
}

// UserListEx 服务端筛选 / 分页用户列表
func (a *AdminService) UserListEx(q UserQuery) map[string]interface{} {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 || q.Size > 1000 {
		q.Size = 20
	}
	all, err := a.userRows(q)
	if err != nil {
		return map[string]interface{}{"total": 0, "list": []UserVO{}, "page": q.Page, "size": q.Size}
	}
	total := len(all)
	start := (q.Page - 1) * q.Size
	if start > total {
		start = total
	}
	end := start + q.Size
	if end > total {
		end = total
	}
	return map[string]interface{}{"total": total, "list": all[start:end], "page": q.Page, "size": q.Size}
}

// ExportUsers 导出筛选结果（不受分页限制）
func (a *AdminService) ExportUsers(q UserQuery) []UserVO {
	rows, err := a.userRows(q)
	if err != nil {
		return []UserVO{}
	}
	return rows
}

// UpdateUserSafe 白名单更新用户信息，禁止直接改写 UID / 密码散列等敏感列
func (a *AdminService) UpdateUserSafe(id int64, updates map[string]interface{}) error {
	allowed := map[string]string{
		"nickname": "nickname", "username": "username", "phone": "phone", "title": "title",
		"dept_id": "dept_id", "deptId": "dept_id", "status": "status",
	}
	clean := map[string]interface{}{}
	for k, v := range updates {
		if col, ok := allowed[k]; ok {
			clean[col] = v
		}
	}
	if len(clean) == 0 {
		return fmt.Errorf("没有可更新的字段")
	}
	if v, ok := clean["status"]; ok {
		s, _ := strconv.Atoi(fmt.Sprint(v))
		if s != 1 && s != 3 {
			return fmt.Errorf("状态值非法")
		}
		clean["status"] = s
	}
	if v, ok := clean["dept_id"]; ok {
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		clean["dept_id"] = n
	}
	clean["updated_at"] = model.Now()
	return a.S.DB.Model(&model.User{}).Where("id = ?", id).Updates(clean).Error
}

// SetUserStatus 启用 / 禁用账号（禁用由鉴权中间件即时拦截）
func (a *AdminService) SetUserStatus(id int64, status int) error {
	if status != 1 && status != 3 {
		return fmt.Errorf("状态值非法")
	}
	return a.S.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": status, "updated_at": model.Now(),
	}).Error
}

// SetUserRoles 全量替换用户角色绑定（支持传空数组解绑）
func (a *AdminService) SetUserRoles(uid string, roleIDs []int64) error {
	if uid == "" {
		return fmt.Errorf("用户不存在")
	}
	uniq := map[int64]bool{}
	ids := []int64{}
	for _, id := range roleIDs {
		if id > 0 && !uniq[id] {
			uniq[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		var cnt int64
		a.S.DB.Model(&model.Role{}).Where("id IN ?", ids).Count(&cnt)
		if int(cnt) != len(ids) {
			return fmt.Errorf("角色不存在")
		}
	}
	tx := a.S.DB.Begin()
	if err := tx.Where("uid = ?", uid).Delete(&model.UserRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, id := range ids {
		if err := tx.Create(&model.UserRole{UID: uid, RoleID: id}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	BumpPermVersion(a.S)
	return nil
}

// GrantRole 追加单个角色（兼容旧接口），并即时失效权限缓存
func (a *AdminService) GrantRoleEx(uid string, roleID int64) error {
	var cnt int64
	a.S.DB.Model(&model.Role{}).Where("id = ?", roleID).Count(&cnt)
	if cnt == 0 {
		return fmt.Errorf("角色不存在")
	}
	a.S.DB.Where("uid = ? AND role_id = ?", uid, roleID).FirstOrCreate(&model.UserRole{UID: uid, RoleID: roleID})
	BumpPermVersion(a.S)
	return nil
}

// RoleList 角色列表（权限数组、成员数、内置标记、版本）
func (a *AdminService) RoleList() []RoleVO {
	var roles []model.Role
	a.S.DB.Order("id").Find(&roles)
	type cntRow struct {
		RoleID int64 `gorm:"column:role_id"`
		C      int64 `gorm:"column:c"`
	}
	var counts []cntRow
	a.S.DB.Raw("SELECT role_id, COUNT(*) AS c FROM sys_user_role GROUP BY role_id").Scan(&counts)
	cm := map[int64]int64{}
	for _, c := range counts {
		cm[c.RoleID] = c.C
	}
	out := make([]RoleVO, 0, len(roles))
	for _, r := range roles {
		perms := []string{}
		_ = json.Unmarshal([]byte(r.Permissions), &perms)
		if perms == nil {
			perms = []string{}
		}
		out = append(out, RoleVO{
			ID: r.ID, Code: r.Code, Name: r.Name, Permissions: perms,
			DataScope: normalizeScope(r.DataScope), Version: r.Version,
			MemberCount: cm[r.ID], BuiltIn: isBuiltIn(r.Code),
		})
	}
	return out
}

// CreateRole 新建角色
func (a *AdminService) CreateRole(code, name string, perms []string, scope string) (*RoleVO, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return nil, fmt.Errorf("角色编码和名称不能为空")
	}
	if err := validPerms(perms); err != nil {
		return nil, err
	}
	scope = normalizeScope(scope)
	if scope == "" {
		return nil, fmt.Errorf("数据范围非法")
	}
	var cnt int64
	a.S.DB.Model(&model.Role{}).Where("code = ?", code).Count(&cnt)
	if cnt > 0 {
		return nil, fmt.Errorf("角色编码已存在")
	}
	b, _ := json.Marshal(perms)
	r := model.Role{Code: code, Name: name, Permissions: string(b), DataScope: scope, Version: 1}
	if err := a.S.DB.Create(&r).Error; err != nil {
		return nil, err
	}
	BumpPermVersion(a.S)
	return &RoleVO{ID: r.ID, Code: r.Code, Name: r.Name, Permissions: perms,
		DataScope: scope, Version: r.Version, BuiltIn: false}, nil
}

// UpdateRole 更新角色名称 / 权限 / 数据范围，并自增权限版本
func (a *AdminService) UpdateRole(id int64, updates map[string]interface{}) (*RoleVO, error) {
	var r model.Role
	if err := a.S.DB.Where("id = ?", id).First(&r).Error; err != nil {
		return nil, fmt.Errorf("角色不存在")
	}
	if r.Code == "SUPER_ADMIN" {
		return nil, fmt.Errorf("超级管理员为内置角色，不可修改")
	}
	clean := map[string]interface{}{}
	if v, ok := updates["name"]; ok {
		name := strings.TrimSpace(fmt.Sprint(v))
		if name == "" {
			return nil, fmt.Errorf("角色名称不能为空")
		}
		clean["name"] = name
	}
	if v, ok := updates["permissions"]; ok {
		perms := toStringSlice(v)
		if err := validPerms(perms); err != nil {
			return nil, err
		}
		b, _ := json.Marshal(perms)
		clean["permissions"] = string(b)
	}
	if v, ok := updates["dataScope"]; ok {
		scope := normalizeScope(fmt.Sprint(v))
		if scope == "" {
			return nil, fmt.Errorf("数据范围非法")
		}
		clean["data_scope"] = scope
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("没有可更新的字段")
	}
	clean["version"] = gorm.Expr("version + 1")
	if err := a.S.DB.Model(&r).Updates(clean).Error; err != nil {
		return nil, err
	}
	BumpPermVersion(a.S)
	a.S.DB.Where("id = ?", id).First(&r)
	for _, vo := range a.RoleList() {
		if vo.ID == id {
			return &vo, nil
		}
	}
	return nil, fmt.Errorf("角色不存在")
}

// DeleteRole 删除角色（内置角色、已绑定角色需先解绑）
func (a *AdminService) DeleteRole(id int64) error {
	var r model.Role
	if err := a.S.DB.Where("id = ?", id).First(&r).Error; err != nil {
		return fmt.Errorf("角色不存在")
	}
	if isBuiltIn(r.Code) {
		return fmt.Errorf("内置角色不可删除")
	}
	var cnt int64
	a.S.DB.Model(&model.UserRole{}).Where("role_id = ?", id).Count(&cnt)
	if cnt > 0 {
		return fmt.Errorf("该角色仍有 %d 名成员，请先解绑", cnt)
	}
	tx := a.S.DB.Begin()
	tx.Where("role_id = ?", id).Delete(&model.UserRole{})
	if err := tx.Delete(&model.Role{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	BumpPermVersion(a.S)
	return nil
}

// EffectFor 预览指定用户的最终权限（多角色并集）
func (a *AdminService) EffectFor(uid string) Effect {
	return ComputeEffect(a.S, uid)
}

func toStringSlice(v interface{}) []string {
	if arr, ok := v.([]string); ok {
		sort.Strings(arr)
		return arr
	}
	if arr, ok := v.([]interface{}); ok {
		out := []string{}
		for _, x := range arr {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	b, _ := json.Marshal(v)
	out := []string{}
	_ = json.Unmarshal(b, &out)
	return out
}
