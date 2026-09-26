package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 统一响应信封。
// 关键契约：所有响应（含错误、未授权、异常）HTTP 状态码**恒为 200**（沿用上游
// SaiAdmin 6.x 的做法），业务状态放在 JSON 的 code 字段里；前端 axios 拦截器
// 只判断 body.code === 200。
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应：{code:200,message:'success',data:...}
//
// 与上游 SaiAdmin 6.x 的成功响应一致。注意 data 为字符串时，
// 会把该字符串**同时**作为 message 与 data 返回。
func Success(c *gin.Context, data interface{}) {
	msg := "success"
	if s, ok := data.(string); ok {
		msg = s
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": msg,
		"data":    data,
	})
}

// SuccessMsg 成功响应并指定 message
func SuccessMsg(c *gin.Context, data interface{}, msg string) {
	c.JSON(http.StatusOK, Body{Code: 200, Message: msg, Data: data})
}

// Fail 失败响应：{code:400,message:...}，**不含 data 字段**
func Fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 400, "message": msg})
}

// AbortFail 失败响应并终止后续 handler
func AbortFail(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 400, "message": msg})
}

// AbortCode 指定业务码的失败响应并终止（用于 code:401 未登录）
func AbortCode(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": code, "message": msg})
}
