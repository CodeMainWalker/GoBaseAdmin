// Package handler 实现各控制器，JSON 结构与业务规则与上游 SaiAdmin 6.x 保持一致。
package handler

import (
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/auth"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/cache"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/captcha"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/middleware"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/model"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// LoginHandler 登录相关控制器
type LoginHandler struct{}

func NewLoginHandler() *LoginHandler { return &LoginHandler{} }

// Captcha GET /core/captcha
// 返回 {result:1, uuid, image}，image 为 data:image/png;base64,...
func (h *LoginHandler) Captcha(c *gin.Context) {
	uuid, code, png := captcha.Generate()
	cache.C.Set(uuid, code, captcha.Expire)

	response.Success(c, gin.H{
		"result": 1,
		"uuid":   uuid,
		"image":  "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	})
}

// checkCaptcha 校验并**立即删除**，失败即作废（与上游 SaiAdmin 6.x 行为一致）
func checkCaptcha(uuid, code string) bool {
	v := cache.C.Get(uuid)
	cache.C.Delete(uuid)
	stored, ok := v.(string)
	if !ok {
		return false
	}
	return strings.ToLower(code) == stored
}

// DebugCaptcha GET /__debug/captcha —— 仅调试模式注册，返回新验证码的明文，
// 便于本地端到端联调（生产构建不会注册该路由）。
func (h *LoginHandler) DebugCaptcha(c *gin.Context) {
	uuid, code, _ := captcha.Generate()
	cache.C.Set(uuid, code, captcha.Expire)
	c.JSON(200, gin.H{"uuid": uuid, "code": code})
}

// Login POST /core/login
func (h *LoginHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Type     string `json:"type"`
		Code     string `json:"code"`
		UUID     string `json:"uuid"`
	}
	_ = c.ShouldBindJSON(&req)
	// 兼容表单提交
	if req.Username == "" {
		req.Username = c.PostForm("username")
		req.Password = c.PostForm("password")
		req.Code = c.PostForm("code")
		req.UUID = c.PostForm("uuid")
		req.Type = c.PostForm("type")
	}
	if req.Type == "" {
		req.Type = "pc"
	}

	if !checkCaptcha(req.UUID, req.Code) {
		response.Fail(c, "验证码错误")
		return
	}

	var user model.User
	err := store.DB.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		emitLoginLog(req.Username, 0, "账号或密码错误，请重新输入!", c.ClientIP())
		response.Fail(c, "账号或密码错误，请重新输入!")
		return
	}

	status := 1
	message := "登录成功"
	if user.Status == 2 {
		status = 0
		message = "您已被禁止登录!"
	}
	if !auth.CheckPassword(user.Password, req.Password) {
		status = 0
		message = "账号或密码错误，请重新输入!"
	}
	if status == 0 {
		emitLoginLog(req.Username, 0, message, c.ClientIP())
		response.Fail(c, message)
		return
	}

	// 更新最后登录信息
	now := model.Now()
	store.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"login_time": now,
		"login_ip":   c.ClientIP(),
	})

	tokens, err := auth.Generate(int(user.ID), user.Username, req.Type)
	if err != nil {
		middleware.Abort(c, "登录失败")
		return
	}
	emitLoginLog(req.Username, 1, message, c.ClientIP())
	response.Success(c, tokens)
}

// emitLoginLog 写登录日志（沿用上游 SaiAdmin 6.x 的 user.login 事件语义）
func emitLoginLog(username string, status int, message, ip string) {
	ua := ""
	osName := parseOS(ua)
	browser := parseBrowser(ua)
	now := model.Now()
	log := model.LoginLog{
		Username:   username,
		IP:         ip,
		IPLocation: "内网IP",
		OS:         osName,
		Browser:    browser,
		Status:     status,
		Message:    message,
		LoginTime:  &now,
	}
	_ = store.DB.Create(&log).Error
}

func parseOS(ua string) string {
	switch {
	case strings.Contains(ua, "Windows"):
		return "Win"
	case strings.Contains(ua, "Mac"):
		return "Mac"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	}
	return "Other"
}

func parseBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "MSIE"):
		return "MSIE"
	case strings.Contains(ua, "Firefox"):
		return "Firefox"
	case strings.Contains(ua, "Chrome"):
		return "Chrome"
	case strings.Contains(ua, "Safari"):
		return "Safari"
	case strings.Contains(ua, "Opera"):
		return "Opera"
	}
	return "Other"
}
