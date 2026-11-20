package service

import (
	"fmt"
	"strconv"
	"strings"

	"nexim/server/internal/model"
)

// DeptNode 组织树节点（含成员数与下级）
type DeptNode struct {
	model.Dept
	MemberCount int64      `json:"memberCount"`
	LeaderName  string     `json:"leaderName"`
	Children    []DeptNode `json:"children"`
}

// DeptTree 构建真实嵌套组织树，并统计各部门直属成员数
func (a *AdminService) DeptTree() []DeptNode {
	var depts []model.Dept
	a.S.DB.Order("path, sort, id").Find(&depts)
	type cntRow struct {
		DeptID int64 `gorm:"column:dept_id"`
		C      int64 `gorm:"column:c"`
	}
	var rows []cntRow
	a.S.DB.Raw("SELECT dept_id, COUNT(*) AS c FROM sys_user WHERE dept_id > 0 GROUP BY dept_id").Scan(&rows)
	counts := map[int64]int64{}
	for _, r := range rows {
		counts[r.DeptID] = r.C
	}
	names := map[string]string{}
	var users []model.User
	a.S.DB.Select("uid, nickname").Find(&users)
	for _, u := range users {
		names[u.UID] = u.Nickname
	}

	nodes := map[int64]*DeptNode{}
	order := make([]int64, 0, len(depts))
	for _, d := range depts {
		n := DeptNode{Dept: d, MemberCount: counts[d.ID], LeaderName: names[d.LeaderUID], Children: []DeptNode{}}
		nodes[d.ID] = &n
		order = append(order, d.ID)
	}
	roots := []DeptNode{}
	attached := map[int64]bool{}
	for _, id := range order {
		n := nodes[id]
		if p, ok := nodes[n.ParentID]; ok && n.ParentID != 0 {
			p.Children = append(p.Children, *n)
			attached[id] = true
		}
	}
	// 子节点在挂载后才可能被追加，需按父节点重新汇总
	for _, id := range order {
		n := nodes[id]
		if attached[id] {
			continue
		}
		roots = append(roots, buildNode(n, nodes))
	}
	return roots
}

func buildNode(n *DeptNode, nodes map[int64]*DeptNode) DeptNode {
	out := *n
	out.Children = []DeptNode{}
	for _, child := range n.Children {
		if full, ok := nodes[child.ID]; ok {
			out.Children = append(out.Children, buildNode(full, nodes))
		}
	}
	return out
}

// CreateDept 新增部门并生成物化路径（/父ID/本ID/）
func (a *AdminService) CreateDept(name string, parentID int64, sortNo int, leaderUID string) (*model.Dept, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("部门名称不能为空")
	}
	var cnt int64
	a.S.DB.Model(&model.Dept{}).Where("name = ? AND parent_id = ?", name, parentID).Count(&cnt)
	if cnt > 0 {
		return nil, fmt.Errorf("同级下已存在同名部门")
	}
	parentPath := "/"
	if parentID > 0 {
		var p model.Dept
		if a.S.DB.Where("id = ?", parentID).First(&p).Error != nil {
			return nil, fmt.Errorf("上级部门不存在")
		}
		parentPath = p.Path
	}
	d := model.Dept{ParentID: parentID, Name: name, Sort: sortNo, LeaderUID: leaderUID, Status: 1}
	if err := a.S.DB.Create(&d).Error; err != nil {
		return nil, err
	}
	d.Path = parentPath + strconv.FormatInt(d.ID, 10) + "/"
	if err := a.S.DB.Model(&d).Update("path", d.Path).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDept 编辑部门；变更上级时重算自身及所有子孙的物化路径
func (a *AdminService) UpdateDept(id int64, updates map[string]interface{}) (*model.Dept, error) {
	var d model.Dept
	if err := a.S.DB.Where("id = ?", id).First(&d).Error; err != nil {
		return nil, fmt.Errorf("部门不存在")
	}
	clean := map[string]interface{}{}
	if v, ok := updates["name"]; ok {
		name := strings.TrimSpace(fmt.Sprint(v))
		if name == "" {
			return nil, fmt.Errorf("部门名称不能为空")
		}
		clean["name"] = name
	}
	if v, ok := updates["sort"]; ok {
		n, _ := strconv.Atoi(fmt.Sprint(v))
		clean["sort"] = n
	}
	if v, ok := updates["leaderUid"]; ok {
		clean["leader_uid"] = fmt.Sprint(v)
	}
	if v, ok := updates["status"]; ok {
		n, _ := strconv.Atoi(fmt.Sprint(v))
		if n != 1 && n != 2 {
			return nil, fmt.Errorf("状态值非法")
		}
		clean["status"] = n
	}
	newParent := d.ParentID
	if v, ok := updates["parentId"]; ok {
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		if n == id {
			return nil, fmt.Errorf("上级部门不能是自身")
		}
		newParent = n
	}
	if newParent != d.ParentID {
		parentPath := "/"
		if newParent > 0 {
			var p model.Dept
			if a.S.DB.Where("id = ?", newParent).First(&p).Error != nil {
				return nil, fmt.Errorf("上级部门不存在")
			}
			if strings.HasPrefix(p.Path, d.Path) {
				return nil, fmt.Errorf("不能将部门移动到自己的下级")
			}
			parentPath = p.Path
		}
		newPath := parentPath + strconv.FormatInt(id, 10) + "/"
		clean["parent_id"] = newParent
		clean["path"] = newPath
		// 子孙路径前缀替换，保证「本部门及下级」查询仍然正确
		var subs []model.Dept
		a.S.DB.Where("path LIKE ? AND id <> ?", d.Path+"%", id).Find(&subs)
		tx := a.S.DB.Begin()
		for i := range subs {
			sub := parentPath + strconv.FormatInt(id, 10) + "/" + strings.TrimPrefix(subs[i].Path, d.Path)
			if err := tx.Model(&model.Dept{}).Where("id = ?", subs[i].ID).Update("path", sub).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
	}
	if len(clean) == 0 {
		return &d, nil
	}
	if err := a.S.DB.Model(&d).Updates(clean).Error; err != nil {
		return nil, err
	}
	a.S.DB.Where("id = ?", id).First(&d)
	return &d, nil
}

// DeleteDept 删除部门：存在子部门或成员时拒绝，避免产生孤儿数据
func (a *AdminService) DeleteDept(id int64) error {
	var cnt int64
	a.S.DB.Model(&model.Dept{}).Where("parent_id = ?", id).Count(&cnt)
	if cnt > 0 {
		return fmt.Errorf("请先删除或移走 %d 个子部门", cnt)
	}
	a.S.DB.Model(&model.User{}).Where("dept_id = ?", id).Count(&cnt)
	if cnt > 0 {
		return fmt.Errorf("该部门下还有 %d 名成员，请先调整成员部门", cnt)
	}
	return a.S.DB.Delete(&model.Dept{}, id).Error
}

// AddDeptMembers 批量把用户加入部门
func (a *AdminService) AddDeptMembers(deptID int64, uids []string) (int, error) {
	var d model.Dept
	if a.S.DB.Where("id = ?", deptID).First(&d).Error != nil {
		return 0, fmt.Errorf("部门不存在")
	}
	clean := []string{}
	seen := map[string]bool{}
	for _, u := range uids {
		u = strings.TrimSpace(u)
		if u != "" && !seen[u] {
			seen[u] = true
			clean = append(clean, u)
		}
	}
	if len(clean) == 0 {
		return 0, fmt.Errorf("请选择要加入的成员")
	}
	res := a.S.DB.Model(&model.User{}).Where("uid IN ?", clean).Updates(map[string]interface{}{
		"dept_id": deptID, "updated_at": model.Now(),
	})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// RemoveDeptMember 将成员移出部门（置为未分配）
func (a *AdminService) RemoveDeptMember(deptID int64, uid string) error {
	res := a.S.DB.Model(&model.User{}).Where("uid = ? AND dept_id = ?", uid, deptID).Updates(map[string]interface{}{
		"dept_id": 0, "updated_at": model.Now(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("该用户不在本部门")
	}
	return nil
}

// DeptMembersFor 部门成员（含下级）；仅对具备管理端部门权限的账号按数据范围过滤，
// 普通用户查看企业通讯录时应看到完整组织架构
func (a *AdminService) DeptMembersFor(deptID int64, callerUID string) []ContactVO {
	list := NewContact(a.S).DeptMembers(deptID)
	if !HasPerm(a.S, callerUID, "dept:list") {
		return list
	}
	f := a.ScopeFor(callerUID)
	if f.Unlimited {
		return list
	}
	out := []ContactVO{}
	for _, m := range list {
		if f.SelfOnly {
			if m.UID == f.SelfUID {
				out = append(out, m)
			}
			continue
		}
		for _, id := range f.DeptIDs {
			if id == m.DeptID {
				out = append(out, m)
				break
			}
		}
	}
	return out
}
