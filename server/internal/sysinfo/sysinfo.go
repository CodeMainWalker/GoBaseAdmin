// Package sysinfo 采集主机资源指标：磁盘容量、CPU 使用率、系统平均负载。
//
// 指标全部来自操作系统原生接口，按平台分文件实现：
//
//	磁盘：Windows 用 GetDiskFreeSpaceEx，Unix 用 statfs
//	CPU ：Windows 用 GetSystemTimes，Linux 读 /proc/stat，两次采样求差值
//	负载：Linux / macOS 用 getloadavg；Windows 没有 load average 概念，
//	     返回 Available=false，由调用方决定如何提示
//
// 约定：采集失败或平台不支持时返回 ok=false（负载为 Available=false），
// 调用方应显示占位符而不是把失败当成“0”。
package sysinfo

import (
	"math"
	"os"
)

// DiskUsage 磁盘容量统计，单位字节。
type DiskUsage struct {
	// Path 统计对象：Windows 为卷根目录（如 D:\），Unix 为被查询的目录
	Path  string
	Total uint64
	Used  uint64
	// Free 普通用户可用的空间（与 df 的 Available 列、资源管理器的“可用”一致）
	Free  uint64
	Usage float64 // 已用百分比，0-100，保留两位小数
}

// LoadAvg 系统平均负载（1 / 5 / 15 分钟）。
type LoadAvg struct {
	One     float64
	Five    float64
	Fifteen float64
	// Available 平台是否提供 load average（Windows 为 false）
	Available bool
}

// Disk 返回当前工作目录所在卷 / 文件系统的容量统计。
func Disk() (DiskUsage, bool) {
	dir, err := os.Getwd()
	if err != nil || dir == "" {
		dir = "."
	}

	total, free, label, err := diskSpace(dir)
	if err != nil || total == 0 {
		return DiskUsage{}, false
	}
	if free > total {
		free = total
	}

	used := total - free
	return DiskUsage{
		Path:  label,
		Total: total,
		Used:  used,
		Free:  free,
		Usage: percent(used, total),
	}, true
}

// Load 返回系统平均负载；不支持的平台返回 Available=false。
func Load() LoadAvg {
	one, five, fifteen, ok := loadAvg()
	if !ok {
		return LoadAvg{}
	}
	return LoadAvg{
		One:       round2(one),
		Five:      round2(five),
		Fifteen:   round2(fifteen),
		Available: true,
	}
}

// Cpu 返回自上一次采样以来的 CPU 使用率（百分比）。
//
// 首次调用没有历史采样，返回的是自开机以来的平均值；调用方可以按固定间隔轮询，
// 之后的每次调用都是该间隔内的真实使用率。
func Cpu() (float64, bool) {
	usage, ok := cpuUsage()
	if !ok {
		return 0, false
	}
	return round2(usage), true
}

// percent 计算百分比并保留两位小数，避免出现 NaN / Inf。
func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return round2(float64(used) / float64(total) * 100)
}

// round2 保留两位小数。
func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}
