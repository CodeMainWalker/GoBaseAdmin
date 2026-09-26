package handler

import (
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/cache"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/response"
)

// ServerHandler 服务监控与缓存管理
type ServerHandler struct{}

func NewServerHandler() *ServerHandler { return &ServerHandler{} }

// Monitor GET /core/server/monitor —— 服务器状态
func (h *ServerHandler) Monitor(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 缓存键数量（字段名沿用上游契约，便于前端直接复用）
	var cacheKeys int64
	cacheKeys = int64(len(cacheSnapshot()))

	response.Success(c, gin.H{
		"cpu": gin.H{
			"cores":      runtime.NumCPU(),
			"used":       "-",
			"idle":       "-",
			"usage":      0,
			"go_version": runtime.Version(),
		},
		"memory": gin.H{
			"total": m.Sys,
			"used":  m.Alloc,
			"free":  m.Sys - m.Alloc,
			"usage": pct(m.Alloc, m.Sys),
			// 键名沿用上游契约，值实为 Go 运行时版本
			"php_version": runtime.Version(),
		},
		"disk": gin.H{
			"total": 0,
			"used":  0,
			"free":  0,
			"usage": 0,
			"path":  ".",
		},
		"load": gin.H{
			"one":     loadAvg(),
			"five":    loadAvg(),
			"fifteen": loadAvg(),
		},
		"server": gin.H{
			"hostname":   hostname(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"go":         runtime.Version(),
			"uptime":     uptimeSeconds(),
			"start_time": startTime.Format("2006-01-02 15:04:05"),
			"now_time":   time.Now().Format("2006-01-02 15:04:05"),
		},
		"cache_keys": cacheKeys,
		"goroutines": runtime.NumGoroutine(),
	})
}

// Cache GET /core/server/cache —— 缓存信息
func (h *ServerHandler) Cache(c *gin.Context) {
	snap := cacheSnapshot()
	response.Success(c, gin.H{
		"mode":  "memory",
		"count": len(snap),
		"keys":  snap,
		"size":  0,
	})
}

// Clear POST /core/server/clear —— 清理全部缓存
func (h *ServerHandler) Clear(c *gin.Context) {
	cache.C.Clear()
	response.Success(c, "清理成功")
}

var startTime = time.Now()

func cacheSnapshot() []string {
	return cache.C.Keys()
}

func pct(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

func loadAvg() float64 { return 0 }

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func uptimeSeconds() int64 {
	return int64(time.Since(startTime).Seconds())
}
