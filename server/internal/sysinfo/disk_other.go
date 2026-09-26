//go:build !windows && !linux && !darwin && !freebsd && !dragonfly

package sysinfo

import "errors"

// errUnsupported 表示当前平台没有实现该指标（服务监控页会显示占位符）。
var errUnsupported = errors.New("sysinfo: 当前平台未实现该指标的采集")

// diskSpace 在不支持的平台上直接报错，让上层显示占位符而不是 0。
func diskSpace(string) (total, free uint64, label string, err error) {
	return 0, 0, "", errUnsupported
}
