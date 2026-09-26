//go:build windows

package sysinfo

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetSystemTimes 在 x/sys/windows 里没有封装，这里按需从 kernel32.dll 取过程地址。
var (
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes = kernel32.NewProc("GetSystemTimes")
)

// getSystemTimes 读取系统累计的 idle / kernel / user 时间（单位 100 纳秒）。
//
// 注意 Windows 的 kernel 时间**包含** idle 时间。
func getSystemTimes(idle, kernel, user *windows.Filetime) error {
	r, _, err := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(idle)),
		uintptr(unsafe.Pointer(kernel)),
		uintptr(unsafe.Pointer(user)),
	)
	if r == 0 {
		return err
	}
	return nil
}

// cpuSample 一次 GetSystemTimes 采样（单位 100 纳秒）。
//
// 注意 Windows 的 kernel 时间**包含** idle 时间，所以：
//
//	total = kernel + user
//	busy  = (kernel - idle) + user
type cpuSample struct {
	idle   uint64
	kernel uint64
	user   uint64
}

var (
	cpuMu   sync.Mutex
	cpuLast *cpuSample
)

// cpuUsage 用 GetSystemTimes 的两次采样差值计算真实的 CPU 使用率。
//
// 首次调用没有历史样本，返回自开机以来的平均值。
func cpuUsage() (float64, bool) {
	var idle, kernel, user windows.Filetime
	if err := getSystemTimes(&idle, &kernel, &user); err != nil {
		return 0, false
	}

	cur := cpuSample{
		idle:   filetimeToUint64(idle),
		kernel: filetimeToUint64(kernel),
		user:   filetimeToUint64(user),
	}

	cpuMu.Lock()
	prev := cpuLast
	cpuLast = &cur
	cpuMu.Unlock()

	var d cpuSample
	if prev != nil && cur.total() > prev.total() {
		d = cpuSample{
			idle:   cur.idle - prev.idle,
			kernel: cur.kernel - prev.kernel,
			user:   cur.user - prev.user,
		}
	} else {
		// 没有历史样本：用自开机以来的累计值
		d = cur
	}

	total := d.total()
	if total == 0 {
		return 0, false
	}
	busy := d.busy()
	if busy > total {
		busy = total
	}
	return float64(busy) / float64(total) * 100, true
}

// total 返回区间内的总 CPU 时间。
func (s cpuSample) total() uint64 { return s.kernel + s.user }

// busy 返回区间内的非空闲时间（kernel 含 idle，需要减掉）。
func (s cpuSample) busy() uint64 {
	idle := s.idle
	if idle > s.kernel {
		idle = s.kernel
	}
	return (s.kernel - idle) + s.user
}

// filetimeToUint64 把 FILETIME 的高低 32 位拼成 64 位计数。
func filetimeToUint64(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}
