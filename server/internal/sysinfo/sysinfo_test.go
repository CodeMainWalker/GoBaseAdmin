package sysinfo

import (
	"runtime"
	"testing"
	"time"
)

// TestDisk 校验磁盘容量采集：总量为正、已用不超过总量、使用率在 0-100。
func TestDisk(t *testing.T) {
	d, ok := Disk()
	if !ok {
		t.Skipf("%s 上无法采集磁盘容量，跳过", runtime.GOOS)
	}
	if d.Total == 0 {
		t.Fatal("磁盘总容量不应为 0")
	}
	if d.Used > d.Total {
		t.Fatalf("已用 %d 大于总容量 %d", d.Used, d.Total)
	}
	if d.Usage < 0 || d.Usage > 100 {
		t.Fatalf("使用率越界: %v", d.Usage)
	}
	if d.Path == "" {
		t.Fatal("统计路径不应为空")
	}
	t.Logf("磁盘 %s: 总 %.2f GB / 已用 %.2f GB / 可用 %.2f GB / 使用率 %.2f%%",
		d.Path, gb(d.Total), gb(d.Used), gb(d.Free), d.Usage)
}

// TestCpu 校验 CPU 使用率：连续两次采样都在合法区间内，并验证忙碌时能测出非零值。
func TestCpu(t *testing.T) {
	first, ok := Cpu()
	if !ok {
		t.Skipf("%s 上无法采集 CPU 使用率，跳过", runtime.GOOS)
	}
	if first < 0 || first > 100 {
		t.Fatalf("首次采样越界: %v", first)
	}

	// 制造一段确定性负载，让第二次采样（差值口径）一定大于 0
	deadline := time.Now().Add(300 * time.Millisecond)
	x := 0.0
	for time.Now().Before(deadline) {
		x += 1.000001
	}
	runtime.KeepAlive(x)

	second, ok := Cpu()
	if !ok {
		t.Fatal("第二次采样失败")
	}
	if second < 0 || second > 100 {
		t.Fatalf("第二次采样越界: %v", second)
	}
	if second == 0 {
		t.Fatalf("300ms 忙等后 CPU 使用率仍为 0（first=%v）", first)
	}
	t.Logf("CPU 使用率: 首次 %.2f%% / 采样区间 %.2f%%", first, second)
}

// TestLoad 校验平均负载：支持的平台应返回非负值，不支持的平台 Available 为 false。
func TestLoad(t *testing.T) {
	l := Load()
	if !l.Available {
		t.Logf("%s 不提供 load average（Available=false）", runtime.GOOS)
		if l.One != 0 || l.Five != 0 || l.Fifteen != 0 {
			t.Fatalf("不可用时三个数值应全为 0，实际 %+v", l)
		}
		return
	}
	if l.One < 0 || l.Five < 0 || l.Fifteen < 0 {
		t.Fatalf("负载不应为负: %+v", l)
	}
	t.Logf("平均负载: %.2f %.2f %.2f", l.One, l.Five, l.Fifteen)
}

func gb(v uint64) float64 { return float64(v) / 1024 / 1024 / 1024 }
