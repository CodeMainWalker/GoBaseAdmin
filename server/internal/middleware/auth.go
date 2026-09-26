// Package middleware 实现鉴权 → 授权 → 操作日志的中间件链，
// 顺序与行为均与上游 SaiAdmin 6.x 一致（CheckLogin → CheckAuth → SystemLog）。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/auth"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/svc"
)

// 上下文键
const (
	CtxAdminInfo = "adminInfo" // *svc.UserInfo
	CtxAdminID   = "adminID"   // int
	CtxAdminName = "adminName" // string
	CtxSlug      = "permSlug"  // string  当前路由要求的权限标识
	CtxPermTitle = "permTitle" // string  用于操作日志的 service_name
)

// noNeedLogin 无需登录的路径白名单（与上游 SaiAdmin 6.x 声明的免登录接口一致）
var noNeedLogin = map[string]bool{
	"/core/captcha": true,
	"/core/login":   true,
}

// CheckLogin 登录校验中间件。
// 所有令牌错误统一输出 code:401 + 固定文案，与上游 SaiAdmin 6.x 完全一致，
// 以便前端 axios 拦截器据此触发重新登录。
func CheckLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if noNeedLogin[path] {
			c.Next()
			return
		}

		ext, err := auth.ExtractToken(c.GetHeader("Authorization"))
		if err != nil {
			response.AbortCode(c, 401, "您的登录凭证错误或者已过期，请重新登录")
			return
		}
		claims, err := auth.Parse(ext)
		if err != nil {
			response.AbortCode(c, 401, "您的登录凭证错误或者已过期，请重新登录")
			return
		}
		// plat 必须为 saiadmin
		if claims.Plat != "saiadmin" {
			response.AbortFail(c, "登录凭证校验失败")
			return
		}

		info := svc.GetUserInfo(claims.ID)
		if info == nil {
			response.AbortCode(c, 401, "您的登录凭证错误或者已过期，请重新登录")
			return
		}

		// 供 BaseModel 钩子填充 created_by / updated_by
		uid := claims.ID
		model.SetCurrentUser(&uid)

		c.Set(CtxAdminID, claims.ID)
		c.Set(CtxAdminName, claims.Username)
		c.Set(CtxAdminInfo, info)
		c.Next()
	}
}

// Authorize 就地执行授权判断，返回 false 表示已写出拒绝响应、调用方应直接返回。
//
// 之所以不在 group 级中间件里做，是因为 Gin 的 group 中间件先于路由级中间件执行，
// 那时 permMiddleware 还没写入 slug，会导致校验被整体跳过。
func Authorize(c *gin.Context, slug string) bool {
	if slug == "" {
		// 未声明权限标识的接口仅需登录（与上游 SaiAdmin 6.x 行为一致）
		return true
	}
	v, ok := c.Get(CtxAdminID)
	if !ok {
		response.AbortCode(c, 401, "您的登录凭证错误或者已过期，请重新登录")
		return false
	}
	uid, _ := v.(int)
	// 系统默认超级管理员，无需权限验证
	if uid == 1 {
		return true
	}
	if !contains(svc.GetUserAuth(uid), slug) {
		response.AbortFail(c, "权限不足，无法访问或操作")
		return false
	}
	return true
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// CORS 跨域中间件（行为与上游 SaiAdmin 6.x 的 CrossDomain 一致）
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		reqHeaders := c.GetHeader("Access-Control-Request-Headers")
		if reqHeaders == "" {
			reqHeaders = "Authorization,Content-Type,X-Requested-With"
		}
		c.Header("Access-Control-Allow-Headers", reqHeaders)
		c.Header("Access-Control-Expose-Headers", "Content-Disposition")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// GetAdminInfo 便捷读取当前用户上下文
func GetAdminInfo(c *gin.Context) *svc.UserInfo {
	if v, ok := c.Get(CtxAdminInfo); ok {
		if info, ok := v.(*svc.UserInfo); ok {
			return info
		}
	}
	return nil
}

// GetAdminID 便捷读取当前用户 ID
func GetAdminID(c *gin.Context) int {
	if v, ok := c.Get(CtxAdminID); ok {
		return v.(int)
	}
	return 0
}

// GetAdminName 便捷读取当前用户名
func GetAdminName(c *gin.Context) string {
	return c.GetString(CtxAdminName)
}

// IsNoNeedLogin 判断路径是否免登录
func IsNoNeedLogin(path string) bool {
	return noNeedLogin[path]
}

// stripQuery 去掉 uri 的查询串
func stripQuery(uri string) string {
	if i := strings.Index(uri, "?"); i >= 0 {
		return uri[:i]
	}
	return uri
}
