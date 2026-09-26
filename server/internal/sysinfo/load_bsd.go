//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package sysinfo

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// loadavg 对应 <sys/resource.h> 中的 struct loadavg：
//
//	struct loadavg { fixpt_t ldavg[3]; long fscale; };
//
// fixpt_t 是 uint32，fscale 是 long（用 Go 的 int 对应，二者对齐规则一致），
// 负载 = ldavg[i] / fscale，因此可以直接把 sysctl 返回的原始字节按该结构体解读。
type loadavg struct {
	load  [3]uint32
	scale int
}

// loadAvg 通过 sysctl vm.loadavg 取 1 / 5 / 15 分钟平均负载。
//
// Go 标准库在 darwin / BSD 上既没有 getloadavg 也没有 loadavg 的封装，
// 这里按 gopsutil 等库的通行做法读取 sysctl，并对解析结果做合理性校验：
// 拿不到数据或数值明显异常时返回 ok=false，宁可显示占位符也不展示错值。
func loadAvg() (one, five, fifteen float64, ok bool) {
	raw, err := unix.Sysctl("vm.loadavg")
	if err != nil || len(raw) < int(unsafe.Sizeof(loadavg{})) {
		return 0, 0, 0, false
	}

	la := *(*loadavg)(unsafe.Pointer(unsafe.StringData(raw)))
	scale := float64(la.scale)
	if scale <= 0 {
		return 0, 0, 0, false
	}

	one = float64(la.load[0]) / scale
	five = float64(la.load[1]) / scale
	fifteen = float64(la.load[2]) / scale
	if !plausibleLoad(one) || !plausibleLoad(five) || !plausibleLoad(fifteen) {
		return 0, 0, 0, false
	}
	return one, five, fifteen, true
}

// plausibleLoad 判断负载值是否在合理范围内（用于识别解析错误）。
func plausibleLoad(v float64) bool {
	return v >= 0 && v <= 100000
}
