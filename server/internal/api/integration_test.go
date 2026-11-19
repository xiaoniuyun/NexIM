package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"nexim/server/internal/config"
	"nexim/server/internal/service"
	"nexim/server/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// TestAdminIntegration 用户管理 / 角色权限 / 组织架构端到端集成测试。
//
// 默认连接本机 docker compose 的 MySQL 与 Redis，在独立库 nexim_itest 上跑完整迁移 + 种子，
// 通过 httptest 驱动真实 Router（含服务端 RBAC 中间件），结束后删除测试库。
// 中间件不可用时自动 Skip，因此 go test ./... 在纯代码环境也不会失败。
//
// 可覆盖连接参数：
//   NEXIM_TEST_MYSQL_DSN  服务器级 DSN，需以 "/" 结尾，例 root:pwd@tcp(127.0.0.1:3306)/
//   NEXIM_TEST_REDIS_ADDR 例 127.0.0.1:6379
func TestAdminIntegration(t *testing.T) {
	mysqlBase := envOr("NEXIM_TEST_MYSQL_DSN", "root:nexim123@tcp(127.0.0.1:3306)/")
	redisAddr := envOr("NEXIM_TEST_REDIS_ADDR", "127.0.0.1:6379")
	const dbName = "nexim_itest"
	testDSN := mysqlBase + dbName + "?charset=utf8mb4&parseTime=True&loc=Local"

	adminDB, err := sql.Open("mysql", mysqlBase)
	if err != nil {
		t.Skipf("MySQL 不可用，跳过集成测试: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	pingErr := adminDB.PingContext(ctx)
	cancel()
	if pingErr != nil {
		adminDB.Close()
		t.Skipf("MySQL 不可用，跳过集成测试: %v", pingErr)
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr, DialTimeout: 500 * time.Millisecond, MaxRetries: -1})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		adminDB.Close()
		t.Skipf("Redis 不可用，跳过集成测试: %v", err)
	}
	rdb.Close()

	if _, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)); err != nil {
		t.Fatalf("清理旧测试库失败: %v", err)
	}
	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE `%s` DEFAULT CHARACTER SET utf8mb4", dbName)); err != nil {
		t.Fatalf("创建测试库失败: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
		_ = adminDB.Close()
	})

	cfg := &config.Config{}
	cfg.Deployment.Profile = "lite"
	cfg.Server.Mode = gin.TestMode
	cfg.DB.DSN = testDSN
	cfg.DB.MessageDriver = "mysql"
	cfg.DB.MaxOpenConns = 5
	cfg.DB.MaxIdleConns = 2
	cfg.Redis.Addr = redisAddr
	cfg.Redis.DB = 15 // 与开发库隔离
	cfg.Auth.JWTSecret = "nexim-integration-test-secret"
	cfg.Auth.AccessTokenTTL = 10 * time.Minute
	cfg.Auth.RefreshTokenTTL = 24 * time.Hour
	cfg.Storage.Driver = "localFS"
	cfg.Storage.LocalDir = t.TempDir()

	st, err := store.New(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("初始化 store: %v", err)
	}
	t.Cleanup(func() { _ = st.RDB.FlushDB(context.Background()).Err(); _ = st.RDB.Close() })

	rt := &Router{
		S: st, C: cfg,
		Auth:    service.NewAuth(st, cfg),
		Contact: service.NewContact(st),
		Admin:   service.NewAdmin(st),
	}
	srv := httptest.NewServer(rt.Engine())
	defer srv.Close()

	call := func(method, path, token string, body interface{}) (int, apiResult, []byte) {
		t.Helper()
		var reader io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reader = bytes.NewReader(b)
		}
		req, err := http.NewRequest(method, srv.URL+path, reader)
		if err != nil {
			t.Fatalf("构造请求: %v", err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		var out apiResult
		_ = json.Unmarshal(raw, &out)
		return res.StatusCode, out, raw
	}
	ok := func(status int, r apiResult, want string) {
		t.Helper()
		if status != 200 || r.Code != 0 {
			t.Fatalf("%s: status=%d code=%d msg=%s", want, status, r.Code, r.Message)
		}
	}

	// 1. 登录：角色来自 RBAC 表，超管拿到通配权限
	status, r, _ := call("POST", "/api/v1/auth/login", "", map[string]string{
		"account": "linxiao", "password": "123456", "deviceId": "itest", "platform": "Web",
	})
	ok(status, r, "超管登录")
	var login struct {
		AccessToken string   `json:"accessToken"`
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
	}
	decode(t, r.Data, &login)
	adminToken := login.AccessToken
	if adminToken == "" || !contains(login.Roles, "admin") || !contains(login.Permissions, "*") {
		t.Fatalf("登录返回的 RBAC 数据异常: roles=%v perms=%v", login.Roles, login.Permissions)
	}
	t.Log("登录 / 角色下发：PASS")

	// 2. 权限目录 / 权限预览
	status, r, _ = call("GET", "/api/v1/auth/permissions", adminToken, nil)
	ok(status, r, "权限下发")
	var perm struct {
		DataScope string `json:"dataScope"`
		Version   string `json:"version"`
	}
	decode(t, r.Data, &perm)
	if perm.DataScope != "all" || perm.Version == "" {
		t.Fatalf("权限下发异常: %+v", perm)
	}
	status, r, _ = call("GET", "/api/v1/admin/permissions", adminToken, nil)
	ok(status, r, "权限目录")
	var catalog struct {
		Catalog []struct {
			Title string `json:"title"`
			Nodes []struct {
				Code string `json:"code"`
			} `json:"nodes"`
		} `json:"catalog"`
		Scopes []struct {
			Value string `json:"value"`
		} `json:"scopes"`
	}
	decode(t, r.Data, &catalog)
	if len(catalog.Catalog) < 4 || len(catalog.Scopes) != 5 {
		t.Fatalf("权限目录不完整: groups=%d scopes=%d", len(catalog.Catalog), len(catalog.Scopes))
	}
	t.Log("权限目录与数据范围：PASS")

	// 3. 用户筛选 / 分页 / 真实角色与部门
	status, r, _ = call("GET", "/api/v1/admin/users?kw=linxiao&status=1&page=1&size=8", adminToken, nil)
	ok(status, r, "用户列表")
	var users struct {
		Total int64 `json:"total"`
		List  []struct {
			ID        int64    `json:"id"`
			Username  string   `json:"username"`
			DeptName  string   `json:"deptName"`
			RoleNames []string `json:"roleNames"`
		} `json:"list"`
	}
	decode(t, r.Data, &users)
	if users.Total != 1 || len(users.List) != 1 || users.List[0].DeptName == "" || len(users.List[0].RoleNames) == 0 {
		t.Fatalf("用户筛选结果异常: %+v", users)
	}
	status, r, _ = call("GET", "/api/v1/admin/roles", adminToken, nil)
	ok(status, r, "角色列表")
	var roleList []struct {
		ID      int64  `json:"id"`
		Code    string `json:"code"`
		BuiltIn bool   `json:"builtIn"`
	}
	decode(t, r.Data, &roleList)
	if len(roleList) < 5 {
		t.Fatalf("内置角色数量不足: %d", len(roleList))
	}
	t.Log("用户筛选 / 分页 / 角色部门关联：PASS")

	// 4. 角色 CRUD + 权限版本递增 + 数据范围
	status, r, _ = call("POST", "/api/v1/admin/roles", adminToken, map[string]interface{}{
		"code": "IT_TEST_ROLE", "name": "集成测试角色", "permissions": []string{"user:list"}, "dataScope": "dept",
	})
	ok(status, r, "新建角色")
	var createdRole struct {
		ID int64 `json:"id"`
	}
	decode(t, r.Data, &createdRole)
	roleID := createdRole.ID
	status, r, _ = call("PUT", fmt.Sprintf("/api/v1/admin/roles/%d", roleID), adminToken, map[string]interface{}{
		"permissions": []string{"user:list", "dept:list", "user:export"}, "dataScope": "deptAndSub",
	})
	ok(status, r, "保存角色权限")
	var savedRole struct {
		Version     int      `json:"version"`
		DataScope   string   `json:"dataScope"`
		Permissions []string `json:"permissions"`
	}
	decode(t, r.Data, &savedRole)
	if savedRole.Version != 2 || savedRole.DataScope != "deptAndSub" || len(savedRole.Permissions) != 3 {
		t.Fatalf("角色权限保存异常: %+v", savedRole)
	}
	t.Log("角色 CRUD / 权限版本：PASS")

	// 5. 部门 CRUD / 层级调整 / 物化路径重算
	status, r, _ = call("GET", "/api/v1/depts/tree", adminToken, nil)
	ok(status, r, "组织树")
	var tree []struct {
		ID       int64  `json:"id"`
		Path     string `json:"path"`
		Children []struct {
			ID   int64  `json:"id"`
			Path string `json:"path"`
		} `json:"children"`
	}
	decode(t, r.Data, &tree)
	if len(tree) != 1 || len(tree[0].Children) != 3 {
		t.Fatalf("初始组织树异常: roots=%d children=%d", len(tree), len(tree[0].Children))
	}
	rootID := tree[0].ID
	status, r, _ = call("POST", "/api/v1/admin/depts", adminToken, map[string]interface{}{
		"name": "集成测试部门", "parentId": rootID, "sort": 9,
	})
	ok(status, r, "新增部门")
	var createdDept struct {
		ID   int64  `json:"id"`
		Path string `json:"path"`
	}
	decode(t, r.Data, &createdDept)
	deptID := createdDept.ID
	if !strings.HasPrefix(createdDept.Path, tree[0].Path) {
		t.Fatalf("子部门路径未按父路径生成: %s", createdDept.Path)
	}
	status, r, _ = call("PUT", fmt.Sprintf("/api/v1/admin/depts/%d", deptID), adminToken, map[string]interface{}{
		"name": "集成测试部门2", "parentId": 0, "sort": 8,
	})
	ok(status, r, "移动部门")
	var movedDept struct {
		Path string `json:"path"`
	}
	decode(t, r.Data, &movedDept)
	if movedDept.Path != fmt.Sprintf("/%d/", deptID) {
		t.Fatalf("移动后路径未重算: %s", movedDept.Path)
	}
	status, r, _ = call("DELETE", fmt.Sprintf("/api/v1/admin/depts/%d", deptID), adminToken, nil)
	ok(status, r, "删除部门")
	t.Log("组织树 CRUD / 物化路径：PASS")

	// 6. 用户角色绑定 → 权限即时生效 → 解绑后缓存失效
	status, r, _ = call("GET", "/api/v1/admin/users?kw=zhangwei", adminToken, nil)
	ok(status, r, "查找普通用户")
	decode(t, r.Data, &users)
	if len(users.List) != 1 {
		t.Fatalf("未找到张伟: %+v", users)
	}
	normalID := users.List[0].ID
	status, r, _ = call("PUT", fmt.Sprintf("/api/v1/admin/users/%d/roles", normalID), adminToken,
		map[string]interface{}{"roleIds": []int64{roleID}})
	ok(status, r, "绑定角色")
	status, r, _ = call("POST", "/api/v1/auth/login", "", map[string]string{
		"account": "zhangwei", "password": "123456", "deviceId": "itest-normal", "platform": "Web",
	})
	ok(status, r, "普通用户登录")
	decode(t, r.Data, &login)
	normalToken := login.AccessToken
	status, _, _ = call("GET", "/api/v1/admin/users", normalToken, nil)
	if status != 200 {
		t.Fatalf("已授权接口应放行，实际 status=%d", status)
	}
	status, _, _ = call("GET", "/api/v1/admin/roles", normalToken, nil)
	if status != 403 {
		t.Fatalf("未授权接口应 403，实际 status=%d", status)
	}
	status, r, _ = call("PUT", fmt.Sprintf("/api/v1/admin/users/%d/roles", normalID), adminToken,
		map[string]interface{}{"roleIds": []int64{}})
	ok(status, r, "解绑角色")
	status, _, _ = call("GET", "/api/v1/admin/users", normalToken, nil)
	if status != 403 {
		t.Fatalf("解绑后权限缓存应立即失效，期望 403，实际 status=%d", status)
	}
	t.Log("服务端 RBAC 校验 / 缓存即时失效：PASS")

	// 7. 禁用即时拦截 + 启用恢复
	status, r, _ = call("DELETE", fmt.Sprintf("/api/v1/admin/users/%d", normalID), adminToken, nil)
	ok(status, r, "禁用账号")
	status, _, _ = call("GET", "/api/v1/admin/users", normalToken, nil)
	if status != 401 {
		t.Fatalf("禁用后应 401，实际 status=%d", status)
	}
	status, r, _ = call("PUT", fmt.Sprintf("/api/v1/admin/users/%d", normalID), adminToken,
		map[string]interface{}{"status": 1})
	ok(status, r, "启用账号")
	status, r, _ = call("POST", "/api/v1/auth/login", "", map[string]string{
		"account": "zhangwei", "password": "123456", "deviceId": "itest-normal2", "platform": "Web",
	})
	ok(status, r, "重新登录")
	t.Log("账号启停即时生效：PASS")

	// 8. CSV 导出（带 BOM 与中文表头）
	status, _, raw := call("GET", "/api/v1/admin/users-export?status=1", adminToken, nil)
	if status != 200 || !bytes.HasPrefix(raw, []byte("\xEF\xBB\xBF")) || !strings.Contains(string(raw), "账号") {
		t.Fatalf("CSV 导出异常: status=%d bytes=%d", status, len(raw))
	}
	t.Log("CSV 导出：PASS")

	// 9. 收尾：删除测试角色，确认组织树恢复
	status, r, _ = call("DELETE", fmt.Sprintf("/api/v1/admin/roles/%d", roleID), adminToken, nil)
	ok(status, r, "删除角色")
	status, r, _ = call("GET", "/api/v1/depts/tree", adminToken, nil)
	ok(status, r, "复查组织树")
	decode(t, r.Data, &tree)
	if len(tree) != 1 {
		t.Fatalf("测试部门未清理干净: roots=%d", len(tree))
	}
}

type apiResult struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decode(t *testing.T, raw json.RawMessage, out interface{}) {
	t.Helper()
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("解析响应失败: %v raw=%s", err, string(raw))
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
