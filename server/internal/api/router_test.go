package api

import (
	"testing"

	"nexim/server/internal/config"

	"github.com/gin-gonic/gin"
)

// TestEngineRoutes 捕获 gin 路由注册冲突（静态/通配路径同级等运行期 panic）
func TestEngineRoutes(t *testing.T) {
	c := &config.Config{}
	c.Server.Mode = gin.TestMode
	rt := &Router{C: c}
	if e := rt.Engine(); e == nil {
		t.Fatal("engine is nil")
	}
}
