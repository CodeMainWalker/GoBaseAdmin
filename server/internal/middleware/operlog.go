package middleware

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// 请求数据中需要脱敏的字段（名单与上游 SaiAdmin 6.x 一致）
var sensitiveKeys = map[string]bool{
	"password":        true,
	"oldPassword":     true,
	"newPassword":     true,
	"confirmPassword": true,
}

// SystemLog 操作日志中间件（与上游 SaiAdmin 6.x 的 SystemLog + user.operateLog 事件行为一致）。
//
// 关键行为：
//   - **GET 请求不记录**（与上游 SaiAdmin 6.x 完全一致）
//   - 免登录接口不记录
//   - service_name 取路由上注册的权限标题，缺省为 '未命名业务'
//   - request_data 对密码字段脱敏为 ******
//   - 日志写入失败不得影响正常响应
func SystemLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Request.Method == "GET" {
			return
		}
		if IsNoNeedLogin(c.Request.URL.Path) {
			return
		}
		// 未通过鉴权的请求不记录（此时无用户上下文）
		info := GetAdminInfo(c)
		if info == nil {
			return
		}

		serviceName := c.GetString(CtxPermTitle)
		if serviceName == "" {
			serviceName = "未命名业务"
		}
		// 列宽 varchar(30)
		if len([]rune(serviceName)) > 30 {
			serviceName = string([]rune(serviceName)[:30])
		}

		ip := c.ClientIP()
		log := model.OperLog{
			Username:    info.Username,
			App:         "saiadmin",
			Method:      c.Request.Method,
			Router:      truncate(c.Request.URL.RequestURI(), 500),
			ServiceName: serviceName,
			IP:          ip,
			IPLocation:  ipLocation(ip),
			RequestData: requestData(c),
		}
		// 显式指定列，避免把零值标签写进日志表
		_ = store.DB.Create(&log).Error
	}
}

// requestData 收集请求参数并脱敏
func requestData(c *gin.Context) string {
	params := map[string]interface{}{}

	if c.Request.Method == "GET" {
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else {
		ct := c.GetHeader("Content-Type")
		if strings.Contains(ct, "application/json") {
			_ = c.ShouldBindBodyWithJSON(&params)
		} else {
			_ = c.Request.ParseMultipartForm(32 << 20)
			for k, v := range c.Request.Form {
				if len(v) > 0 {
					params[k] = v[0]
				}
			}
		}
	}

	for k := range params {
		if sensitiveKeys[k] {
			params[k] = "******"
		}
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ipLocation 返回 IP 归属地。
// 上游 SaiAdmin 6.x 使用 ip2region 离线库；Go 版不引入该数据文件，
// 仅对内外网做区分，其余返回 '未知'，保证字段语义与长度约束一致。
func ipLocation(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "未知"
	}
	if parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() {
		return "内网IP"
	}
	return "未知"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// SetPermMeta 由路由注册处调用，写入权限标题与标识
func SetPermMeta(c *gin.Context, title, slug string) {
	c.Set(CtxPermTitle, title)
	c.Set(CtxSlug, slug)
}
