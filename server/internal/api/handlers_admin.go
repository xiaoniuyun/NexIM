package api

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	"nexim/server/internal/model"
	"nexim/server/internal/pkg"
	"nexim/server/internal/service"

	"github.com/gin-gonic/gin"
)

// userQueryFrom 统一解析用户列表 / 导出的筛选条件
func userQueryFrom(c *gin.Context) service.UserQuery {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	deptID, _ := strconv.ParseInt(c.Query("deptId"), 10, 64)
	return service.UserQuery{
		Keyword: c.Query("kw"), Status: c.Query("status"), DeptID: deptID,
		IncludeSub: c.Query("includeSub") == "1", Page: page, Size: size,
	}
}

// GET /admin/users-export 导出筛选后的用户（CSV，带 BOM 便于 Excel 打开中文）
func (rt *Router) adminUsersExport(c *gin.Context) {
	q := userQueryFrom(c)
	q.CallerUID = uid(c)
	rows := rt.Admin.ExportUsers(q)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="nexim-users.csv"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"账号", "昵称", "部门", "岗位", "手机号", "角色", "状态", "最后登录"})
	for _, u := range rows {
		state := "正常"
		if u.Status != 1 {
			state = "禁用"
		}
		last := "—"
		if u.LastLoginAt > 0 {
			last = strconv.FormatInt(u.LastLoginAt, 10)
		}
		_ = w.Write([]string{u.Username, u.Nickname, u.DeptName, u.Title, u.Phone,
			strings.Join(u.RoleNames, " / "), state, last})
	}
	w.Flush()
	rt.Admin.Audit(uid(c), "user:export", "", strconv.Itoa(len(rows)))
}

// PUT /admin/users/:id/roles 全量替换用户角色
func (rt *Router) adminSetUserRoles(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var u model.User
	if rt.S.DB.Where("id = ?", id).First(&u).Error != nil {
		pkg.Fail(c, 404, 4000, "用户不存在")
		return
	}
	var req struct {
		RoleIDs []int64 `json:"roleIds"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	if rt.Admin.HasSuperRole(req.RoleIDs) && !rt.Admin.IsSuperAdmin(uid(c)) {
		pkg.Fail(c, 403, 4003, "只有超级管理员可以授予超级管理员角色")
		return
	}
	if err := rt.Admin.SetUserRoles(u.UID, req.RoleIDs); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "rbac:role:grant", u.UID, strings.Join(rt.Admin.EffectFor(u.UID).RoleNames, ","))
	pkg.OK(c, rt.Admin.EffectFor(u.UID))
}

// GET /admin/user-permissions/:id 预览指定用户的最终权限
func (rt *Router) adminUserPermissions(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var u model.User
	if rt.S.DB.Where("id = ?", id).First(&u).Error != nil {
		pkg.Fail(c, 404, 4000, "用户不存在")
		return
	}
	pkg.OK(c, rt.Admin.EffectFor(u.UID))
}

// GET /admin/permissions 权限目录与数据范围选项（角色配置页数据源）
func (rt *Router) adminPermissions(c *gin.Context) {
	pkg.OK(c, gin.H{
		"catalog": service.PermCatalog,
		"scopes":  service.DataScopes,
		"version": service.PermVersion(rt.S),
	})
}

// POST /admin/roles 新建角色
func (rt *Router) adminCreateRole(c *gin.Context) {
	var req struct {
		Code        string   `json:"code" binding:"required"`
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions"`
		DataScope   string   `json:"dataScope"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	r, err := rt.Admin.CreateRole(req.Code, req.Name, req.Permissions, req.DataScope)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "rbac:role:create", r.Code, r.Name)
	pkg.OK(c, r)
}

// PUT /admin/roles/:id 保存角色权限 / 名称 / 数据范围
func (rt *Router) adminUpdateRole(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req map[string]interface{}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	r, err := rt.Admin.UpdateRole(id, req)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "rbac:role:edit", r.Code, strconv.Itoa(r.Version))
	pkg.OK(c, r)
}

// DELETE /admin/roles/:id 删除自定义角色
func (rt *Router) adminDeleteRole(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := rt.Admin.DeleteRole(id); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "rbac:role:delete", strconv.FormatInt(id, 10), "")
	pkg.OK(c, nil)
}

// POST /admin/depts 新增部门
func (rt *Router) adminCreateDept(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		ParentID  int64  `json:"parentId"`
		Sort      int    `json:"sort"`
		LeaderUID string `json:"leaderUid"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	d, err := rt.Admin.CreateDept(req.Name, req.ParentID, req.Sort, req.LeaderUID)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "dept:create", strconv.FormatInt(d.ID, 10), d.Path)
	pkg.OK(c, d)
}

// PUT /admin/depts/:id 编辑 / 移动部门
func (rt *Router) adminUpdateDept(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req map[string]interface{}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	d, err := rt.Admin.UpdateDept(id, req)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "dept:edit", strconv.FormatInt(id, 10), d.Path)
	pkg.OK(c, d)
}

// DELETE /admin/depts/:id 删除部门
func (rt *Router) adminDeleteDept(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := rt.Admin.DeleteDept(id); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "dept:delete", strconv.FormatInt(id, 10), "")
	pkg.OK(c, nil)
}

// POST /admin/depts/:id/members 批量加入部门成员
func (rt *Router) adminDeptAddMembers(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		UIDs []string `json:"uids" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pkg.Fail(c, 400, 1001, "参数错误")
		return
	}
	n, err := rt.Admin.AddDeptMembers(id, req.UIDs)
	if err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "dept:member:add", strconv.FormatInt(id, 10), strconv.Itoa(n))
	pkg.OK(c, gin.H{"added": n})
}

// DELETE /admin/depts/:id/members/:uid 移出部门成员
func (rt *Router) adminDeptRemoveMember(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := rt.Admin.RemoveDeptMember(id, c.Param("uid")); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "dept:member:remove", strconv.FormatInt(id, 10), c.Param("uid"))
	pkg.OK(c, nil)
}

// GET /admin/roles/:id/members 角色成员列表
func (rt *Router) adminRoleMembers(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	pkg.OK(c, rt.Admin.RoleMembers(id))
}

// DELETE /admin/roles/:id/members/:uid 解绑角色成员
func (rt *Router) adminRoleRemoveMember(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := rt.Admin.RemoveRoleMember(id, c.Param("uid")); err != nil {
		pkg.Fail(c, 400, 4000, err.Error())
		return
	}
	rt.Admin.Audit(uid(c), "rbac:role:revoke", strconv.FormatInt(id, 10), c.Param("uid"))
	pkg.OK(c, nil)
}
