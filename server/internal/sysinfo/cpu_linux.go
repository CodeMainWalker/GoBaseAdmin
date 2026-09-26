//go:build linux

package sysinfo

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"sync"
)

// cpuSample 一次 /proc/stat 采样：idle 为空闲累计值，total 为全部累计值（单位 jiffies）。
type cpuSample struct {
	idle  uint64
	total uint64
}

var (
	cpuMu   sync.Mutex
	cpuLast *cpuSample
)

// cpuUsage 读取 /proc/stat 并与上一次采样求差值，得到真实的 CPU 使用率。
//
// 首次调用没有历史样本，返回自开机以来的平均值（与 uptime 展示的口径一致）。
func cpuUsage() (float64, bool) {
	cur, ok := readProcStat()
	if !ok {
		return 0, false
	}

	cpuMu.Lock()
	prev := cpuLast
	cpuLast = &cur
	cpuMu.Unlock()

	if prev == nil || cur.total <= prev.total {
		return sampleUsage(cur, cpuSample{}), true
	}
	return sampleUsage(cur, *prev), true
}

// sampleUsage 用两次采样的差值算使用率；prev 为零值时退化为「自开机以来」。
func sampleUsage(cur, prev cpuSample) float64 {
	total := cur.total - prev.total
	idle := cur.idle - prev.idle
	if total == 0 {
		return 0
	}
	busy := total - idle
	if busy > total {
		busy = total
	}
	return float64(busy) / float64(total) * 100
}

// readProcStat 解析 /proc/stat 的第一行（cpu 汇总行）。
//
// 字段顺序：user nice system idle iowait irq softirq steal guest guest_nice
// 其中 idle 计入空闲的是 idle + iowait；guest 已包含在 user/nice 中，不重复累加。
func readProcStat() (cpuSample, bool) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuSample{}, false
	}

	line := b
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		line = b[:i]
	}
	fields := strings.Fields(string(line))
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSample{}, false
	}

	var s cpuSample
	for i, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuSample{}, false
		}
		s.total += v
		switch i {
		case 3, 4: // idle, iowait
			s.idle += v
		}
	}
	return s, s.total > 0
}
