package router

import (
	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/handler"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/middleware"
)

// Perm 描述一个路由的权限元信息（相当于上游 SaiAdmin 6.x 在控制器上标注的权限注解）。
// title 用于操作日志的 service_name；slug 为空表示仅需登录、不校验权限。
type Perm struct {
	Title string
	Slug  string
}

// permMiddleware 在**路由级**完成两件事：
//  1. 写入权限标题/标识（供操作日志与后续判断使用）
//  2. 就地执行授权判断
//
// 注意：授权必须在这里做，不能放在 group 级的 CheckAuth 里。
// Gin 的 group 中间件先于路由级中间件执行，若在 group 级读取 slug，
// 此时 permMiddleware 尚未运行，slug 恒为空 => 权限校验会被整体跳过。
func permMiddleware(p Perm) gin.HandlerFunc {
	return func(c *gin.Context) {
		middleware.SetPermMeta(c, p.Title, p.Slug)
		if !middleware.Authorize(c, p.Slug) {
			return
		}
		c.Next()
	}
}

// Setup 注册全部路由
func Setup(debug bool) *gin.Engine {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	// InstallGuard 必须最早注册：未安装时 store.DB 为 nil，
	// 任何依赖 DB 的中间件/处理器都会被它拦在外面，避免空指针 panic。
	r.Use(gin.Logger(), middleware.CORS(), middleware.Recovery(debug), middleware.InstallGuard())

	login := handler.NewLoginHandler()
	sys := handler.NewSystemHandler()

	// ---- 免登录接口 ----
	// 网页安装向导已下线，首次安装改用命令行：gobaseadmin install
	core := r.Group("/core")
	{
		core.GET("/captcha", login.Captcha)
		core.POST("/login", login.Login)
	}

	// ---- 仅调试模式：暴露验证码明文，便于本地端到端联调 ----
	if debug {
		r.GET("/__debug/captcha", login.DebugCaptcha)
	}

	// ---- 需登录接口 ----
	// 这里只做登录校验与操作日志；授权统一由各路由的 permMiddleware 负责。
	api := r.Group("/core")
	api.Use(middleware.CheckLogin(), middleware.SystemLog())
	{
		// 系统通用
		api.GET("/system/user", sys.UserInfo)
		api.GET("/system/dictAll", sys.DictAll)
		api.GET("/system/menu", sys.Menu)
		api.GET("/system/statistics", permMiddleware(Perm{"工作台数据统计", "core:console:list"}), sys.Statistics)
		api.GET("/system/loginChart", permMiddleware(Perm{"工作台数据统计", "core:console:list"}), sys.LoginChart)
		api.GET("/system/loginBarChart", permMiddleware(Perm{"工作台数据统计", "core:console:list"}), sys.LoginBarChart)
		api.GET("/system/clearAllCache", sys.ClearAllCache)
		api.GET("/system/getUserList", permMiddleware(Perm{"用户列表读取", "core:system:user"}), sys.GetUserList)
		api.GET("/system/getLoginLogList", sys.GetLoginLogList)
		api.GET("/system/getOperationLogList", sys.GetOperationLogList)
	}

	RegisterSystemRoutes(api)
	RegisterSafeguardRoutes(api)
	RegisterUploadRoutes(api)

	return r
}
