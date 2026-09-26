package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// BusinessError 业务异常（由 Abort/AbortCode 抛出，Recovery 统一渲染）。
// 其响应**只含** code/message 两个字段（无 type、无调试信息）。
type BusinessError struct {
	Code int
	Msg  string
}

func (e *BusinessError) Error() string { return e.Msg }

// NewBizError 构造业务异常，code 默认 400
func NewBizError(msg string) *BusinessError {
	return &BusinessError{Code: 400, Msg: msg}
}

// NewBizErrorCode 指定业务码
func NewBizErrorCode(code int, msg string) *BusinessError {
	return &BusinessError{Code: code, Msg: msg}
}

// Recovery 全局异常兜底。
//
// 与上游 SaiAdmin 6.x 异常渲染规则对齐的关键点：
//   - HTTP 状态码**恒为 200**，业务状态在 body.code
//   - 业务异常（BusinessError）走 Message 直出、无 type 字段
//   - 其它异常带 type:"failed"；当 code 恰为 500 时，message 被替换为
//     字面量 'Server internal error'（与上游 SaiAdmin 6.x 行为一致）
func Recovery(debugMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 业务异常：直接输出，无 type
				if be, ok := r.(*BusinessError); ok {
					code := be.Code
					if code == 0 {
						code = 500
					}
					c.AbortWithStatusJSON(http.StatusOK, gin.H{
						"code":    code,
						"message": be.Msg,
					})
					return
				}

				// 其它异常：携带 type:"failed"
				body := gin.H{
					"code":    500,
					"message": "Server internal error",
					"type":    "failed",
				}
				if debugMode {
					body["request_url"] = c.Request.Method + " " + c.Request.URL.RequestURI()
					body["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
					body["client_ip"] = c.ClientIP()
					body["exception_handle"] = fmt.Sprintf("%T", r)
					body["exception_info"] = gin.H{
						"message": fmt.Sprint(r),
						"trace":   splitLines(string(debug.Stack())),
					}
				}
				c.AbortWithStatusJSON(http.StatusOK, body)
			}
		}()
		c.Next()
	}
}

func splitLines(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// Abort 在 handler 中抛错，交由 Recovery 统一处理
func Abort(c *gin.Context, msg string) {
	panic(NewBizError(msg))
}

// AbortCode 指定业务码抛错
func AbortCode(c *gin.Context, code int, msg string) {
	panic(NewBizErrorCode(code, msg))
}
