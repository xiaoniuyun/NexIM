package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"nexim/server/internal/model"
	"nexim/server/internal/store"
)

// PermNode 权限点：code 同时用于服务端中间件校验与前端按钮显隐
type PermNode struct {
	Label string `json:"label"`
	Code  string `json:"code"`
	Hint  string `json:"hint"`
}

// PermGroup 权限分组（角色配置页按分组渲染）
type PermGroup struct {
	Title string     `json:"title"`
	Hint  string     `json:"hint"`
	Nodes []PermNode `json:"nodes"`
}

// PermCatalog 全站权限目录：新增受保护接口时应在此登记
var PermCatalog = []PermGroup{
	{Title: "菜单与目录", Hint: "控制动态路由与侧边栏可见性", Nodes: []PermNode{
		{"工作台", "dashboard:view", "菜单"}, {"用户管理", "menu:users", "菜单"},
		{"组织架构", "menu:depts", "菜单"}, {"角色权限", "menu:roles", "菜单"},
		{"消息审计", "menu:audit", "菜单"}, {"系统设置", "menu:settings", "菜单"},
		{"群组管理", "menu:groups", "菜单"}, {"操作日志", "menu:logs", "菜单"},
		{"公告管理", "menu:notices", "菜单"}, {"内容安全", "menu:sensitive", "菜单"},
		{"音视频", "menu:calls", "菜单"},
	}},
	{Title: "用户与组织", Hint: "用户增删改查、启停与部门维护", Nodes: []PermNode{
		{"用户列表", "user:list", "GET /admin/users"}, {"新增用户", "user:create", "POST /admin/users"},
		{"编辑用户", "user:edit", "PUT /admin/users/{id}"}, {"启停账号", "user:disable", "DELETE /admin/users/{id}"},
		{"重置密码", "user:reset-pwd", "POST /admin/users/{id}/reset-password"},
		{"导出用户", "user:export", "GET /admin/users/export"},
		{"部门列表", "dept:list", "GET /depts"}, {"新增部门", "dept:create", "POST /admin/depts"},
		{"编辑部门", "dept:edit", "PUT /admin/depts/{id}"}, {"删除部门", "dept:delete", "DELETE /admin/depts/{id}"},
	}},
	{Title: "角色与授权", Hint: "角色 CRUD、权限配置与用户绑定", Nodes: []PermNode{
		{"角色列表", "rbac:role:list", "GET /admin/roles"}, {"新建角色", "rbac:role:create", "POST /admin/roles"},
		{"编辑角色", "rbac:role:edit", "PUT /admin/roles/{id}"}, {"删除角色", "rbac:role:delete", "DELETE /admin/roles/{id}"},
		{"授权变更", "rbac:role:grant", "POST /admin/roles/grant"},
		{"权限预览", "rbac:perm:preview", "GET /admin/users/{id}/permissions"},
	}},
	{Title: "内容与审计", Hint: "审计、日志、公告与敏感词", Nodes: []PermNode{
		{"消息审计", "audit:message:list", "GET /admin/audit/messages"}, {"导出审计", "audit:message:export", "按钮"},
		{"操作日志", "log:list", "GET /admin/audit"}, {"导出日志", "log:export", "按钮"},
		{"敏感词配置", "sensitive:config", "POST /admin/sensitive-words"},
		{"公告发布", "notice:publish", "POST /admin/notices"},
		{"通话记录", "rtc:record:list", "GET /admin/calls"}, {"质量监控", "rtc:quality:view", "按钮"},
		{"处理举报", "report:handle", "按钮"},
	}},
}

// DataScopes 数据范围取值（多角色取最宽）
var DataScopes = []map[string]string{
	{"value": "all", "label": "全部数据"},
	{"value": "deptAndSub", "label": "本部门及下级"},
	{"value": "dept", "label": "本部门"},
	{"value": "self", "label": "仅本人"},
	{"value": "custom", "label": "自定义部门"},
}

var scopeRank = map[string]int{"self": 1, "custom": 2, "dept": 3, "deptAndSub": 4, "all": 5}

// Effect 用户最终权限（多角色并集 + 最宽数据范围）
type Effect struct {
	Permissions []string `json:"permissions"`
	RoleCodes   []string `json:"roleCodes"`
	RoleNames   []string `json:"roleNames"`
	DataScope   string   `json:"dataScope"`
	Version     string   `json:"version"`
}

// AllPermCodes 目录内全部权限码
func AllPermCodes() []string {
	out := []string{}
	for _, g := range PermCatalog {
		for _, n := range g.Nodes {
			out = append(out, n.Code)
		}
	}
	return out
}

// PermVersion 返回当前权限版本号；角色/绑定变更后自增即可让所有缓存立即失效
func PermVersion(s *store.Store) string {
	v, err := s.RDB.Get(s.Ctx, "rbac:ver").Result()
	if err != nil || v == "" {
		return "1"
	}
	return v
}

// BumpPermVersion 权限变更后调用，在线用户下一次请求即取到新权限
func BumpPermVersion(s *store.Store) {
	_ = s.RDB.Incr(s.Ctx, "rbac:ver").Err()
}

// ComputeEffect 计算用户最终权限；结果按权限版本号缓存在 Redis 中
func ComputeEffect(s *store.Store, uid string) Effect {
	ver := PermVersion(s)
	key := fmt.Sprintf("rbac:perm:%s:%s", ver, uid)
	if cached, err := s.RDB.Get(s.Ctx, key).Result(); err == nil {
		var e Effect
		if json.Unmarshal([]byte(cached), &e) == nil {
			return e
		}
	}
	var roles []model.Role
	s.DB.Joins("JOIN sys_user_role ur ON ur.role_id = sys_role.id").
		Where("ur.uid = ?", uid).Order("sys_role.id").Find(&roles)

	set := map[string]bool{}
	scope := "self"
	wildcard := false
	e := Effect{Permissions: []string{}, RoleCodes: []string{}, RoleNames: []string{}, DataScope: scope, Version: ver}
	for _, r := range roles {
		e.RoleCodes = append(e.RoleCodes, r.Code)
		e.RoleNames = append(e.RoleNames, r.Name)
		if scopeRank[r.DataScope] > scopeRank[scope] {
			scope = r.DataScope
		}
		var perms []string
		if json.Unmarshal([]byte(r.Permissions), &perms) != nil {
			continue
		}
		for _, p := range perms {
			if p == "*" {
				wildcard = true
				continue
			}
			set[p] = true
		}
	}
	if wildcard {
		set["*"] = true
		for _, p := range AllPermCodes() {
			set[p] = true
		}
	}
	for p := range set {
		e.Permissions = append(e.Permissions, p)
	}
	sort.Strings(e.Permissions)
	e.DataScope = scope
	if data, err := json.Marshal(e); err == nil {
		_ = s.RDB.Set(s.Ctx, key, string(data), 5*time.Minute).Err()
	}
	return e
}

// HasPerm 判断用户是否拥有权限（* 通配符放行）
func HasPerm(s *store.Store, uid, code string) bool {
	if code == "" {
		return true
	}
	for _, p := range ComputeEffect(s, uid).Permissions {
		if p == "*" || strings.EqualFold(p, code) {
			return true
		}
	}
	return false
}

// MenusOf 根据菜单权限码生成可访问菜单
func MenusOf(e Effect) []map[string]string {
	have := map[string]bool{}
	for _, p := range e.Permissions {
		have[p] = true
	}
	menus := []map[string]string{}
	if have["*"] || have["dashboard:view"] {
		menus = append(menus, map[string]string{"name": "工作台", "path": "/dashboard"})
	}
	menuDef := [][2]string{
		{"menu:users", "/users"}, {"menu:depts", "/depts"}, {"menu:roles", "/roles"},
		{"menu:audit", "/audit"}, {"menu:settings", "/settings"}, {"menu:groups", "/groups"},
		{"menu:logs", "/logs"}, {"menu:notices", "/notices"}, {"menu:sensitive", "/sensitive"},
		{"menu:calls", "/calls"},
	}
	for _, m := range menuDef {
		if have["*"] || have[m[0]] {
			menus = append(menus, map[string]string{"name": m[0], "path": m[1]})
		}
	}
	return menus
}
