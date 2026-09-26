//go:build linux || darwin || freebsd || dragonfly

package sysinfo

import "syscall"

// diskSpace 用 statfs 查询 dir 所在文件系统的容量。
//
// 只覆盖 statfs 字段命名的平台：openbsd 的 Statfs_t 用 F_bsize/F_blocks/F_bavail，
// netbsd 只有 statvfs，这两者不在这里处理（上层会显示占位符）。
func diskSpace(dir string) (total, free uint64, label string, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, 0, "", err
	}

	// 各平台的 Statfs_t 字段类型不完全一致（linux 上是 uint64、freebsd 上是 int64），
	// 统一转成 uint64 再计算
	bsize := uint64(st.Bsize)
	total = uint64(st.Blocks) * bsize
	// Bavail 是普通用户可用的块数，与 df 的 Available 列一致
	free = uint64(st.Bavail) * bsize

	return total, free, dir, nil
}
