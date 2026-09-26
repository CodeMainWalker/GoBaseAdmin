//go:build !windows && !linux

package sysinfo

// cpuUsage 在当前平台没有实现，返回 false 让上层显示占位符。
func cpuUsage() (float64, bool) { return 0, false }
