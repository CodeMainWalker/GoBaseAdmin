package config

import (
	"os"
	"path/filepath"
	"testing"
)

// 每个用例结束都要复位包级状态，避免相互污染
func cleanupForced(t *testing.T) {
	t.Helper()
	SetEnvFile("")
	loadedFrom = ""
}

func TestSetEnvFileMissingPathStillRecorded(t *testing.T) {
	cleanupForced(t)
	defer cleanupForced(t)

	missing := filepath.Join(t.TempDir(), "not-yet.env")
	// 目标文件还不存在也要能登记（install 需要写它）
	SetEnvFile(missing)
	if got := LocalEnvPath(); got != missing {
		t.Fatalf("LocalEnvPath() = %q, 期望 %q", got, missing)
	}
	if err := Load(); err == nil {
		t.Fatal("加载不存在的 --env-file 应报错")
	}
}

func TestSetEnvFileReadsOnlyThatFile(t *testing.T) {
	cleanupForced(t)
	defer cleanupForced(t)

	dir := t.TempDir()
	// 造一个「更优先」的 cwd .env，确认强制指定时不会被读到
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_NAME = wrong\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	forced := filepath.Join(dir, "custom.env")
	if err := os.WriteFile(forced, []byte("DB_NAME = right\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	SetEnvFile(forced)
	if err := Load(); err != nil {
		t.Fatalf("加载失败: %v", err)
	}
	if C.DBName != "right" {
		t.Fatalf("DB_NAME = %q, 期望 right", C.DBName)
	}
	if LoadedFrom() != forced {
		t.Fatalf("LoadedFrom() = %q, 期望 %q", LoadedFrom(), forced)
	}
	if !LocalEnvExists() {
		t.Fatal("强制路径存在时 LocalEnvExists() 应为 true")
	}
}

func TestSetEnvFileEmptyResets(t *testing.T) {
	cleanupForced(t)
	defer cleanupForced(t)

	SetEnvFile("/tmp/whatever.env")
	SetEnvFile("")
	if ForcedEnvFile() != "" {
		t.Fatalf("清空后 ForcedEnvFile() 应为空, 实际 %q", ForcedEnvFile())
	}
	if LocalEnvPath() == "/tmp/whatever.env" {
		t.Fatal("清空后不应再用旧的强制路径")
	}
}
