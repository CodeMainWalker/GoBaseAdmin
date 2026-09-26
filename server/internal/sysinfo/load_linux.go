//go:build linux

package sysinfo

import (
	"os"
	"strconv"
	"strings"
)

// loadAvg 读取 /proc/loadavg 的前三个字段（1 / 5 / 15 分钟平均负载）。
//
// Linux 上 Go 标准库没有提供 getloadavg 封装（内核也没有对应系统调用，
// glibc 同样是读这个文件），所以这里直接解析 /proc/loadavg。
func loadAvg() (one, five, fifteen float64, ok bool) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, false
	}

	fields := strings.Fields(string(b))
	if len(fields) < 3 {
		return 0, 0, 0, false
	}

	values := make([]float64, 3)
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return 0, 0, 0, false
		}
		values[i] = v
	}
	return values[0], values[1], values[2], true
}
