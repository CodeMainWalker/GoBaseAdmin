package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/install"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// InstallGuard 未安装时统一返回 JSON 业务失败。
//
// 这是把「数据库未就绪」这件事收口到一处的关键：
// store.DB 在未安装时为 nil，若请求漏进来会直接空指针 panic，
// 所以本中间件必须注册为**全局**中间件，且早于 CheckLogin 等依赖 DB 的中间件。
//
// 网页安装向导下线后不再放行任何白名单路径，也不再做 HTML 跳转，
// 只给出一句能照做的命令行提示。
func InstallGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if store.Ready() || install.Installed() {
			c.Next()
			return
		}
		response.AbortFail(c, "系统尚未安装，请先执行 gobaseadmin install 完成安装，完成后重启服务")
	}
}
