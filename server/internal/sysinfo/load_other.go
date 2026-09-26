//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package sysinfo

// loadAvg 在 Windows 等平台不可用：
// Windows 没有 load average 概念（等价指标是处理器队列长度，需要 PDH 计数器），
// 这里返回 ok=false，由前端提示「该平台不提供负载」，CPU 使用率仍然可用。
func loadAvg() (one, five, fifteen float64, ok bool) {
	return 0, 0, 0, false
}
