//go:build windows

package sysinfo

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// diskSpace 用 GetDiskFreeSpaceEx 查询 dir 所在卷的容量。
//
// 返回的 label 是卷根目录（如 D:\），total/free 单位为字节。
func diskSpace(dir string) (total, free uint64, label string, err error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}

	// VolumeName("D:\a\b") == "D:"；UNC 路径会得到 \\server\share
	volume := filepath.VolumeName(abs)
	if volume == "" {
		volume = abs
	}
	root := volume + `\`

	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0, "", err
	}

	var freeToCaller, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeToCaller, &totalBytes, &totalFree); err != nil {
		return 0, 0, "", err
	}

	// 用「调用者可用字节数」，与资源管理器显示的可用空间一致
	return totalBytes, freeToCaller, root, nil
}
