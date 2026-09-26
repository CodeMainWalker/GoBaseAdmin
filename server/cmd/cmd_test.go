package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCmd 构造全新的命令树并执行 args，返回标准输出/错误输出与错误
func runCmd(args ...string) (string, string, error) {
	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errBuf.String(), err
}

// 无参数只打印帮助，且不算失败（退出码 0）
func TestRootNoArgsShowsHelp(t *testing.T) {
	out, _, err := runCmd()
	if err != nil {
		t.Fatalf("无参数执行应返回 nil，实际: %v", err)
	}
	for _, want := range []string{"Usage:", "serve", "install", "secret", "version", "config"} {
		if !strings.Contains(out, want) {
			t.Fatalf("帮助输出缺少 %q:\n%s", want, out)
		}
	}
}

// 未知子命令必须报错（退出码 1）
func TestRootUnknownCommand(t *testing.T) {
	_, _, err := runCmd("definitely-not-a-command")
	if err == nil {
		t.Fatal("未知子命令应返回错误")
	}
}

func TestVersionCmd(t *testing.T) {
	out, _, err := runCmd("version")
	if err != nil {
		t.Fatalf("version 执行失败: %v", err)
	}
	if !strings.Contains(out, "gobaseadmin "+appVersion) {
		t.Fatalf("版本输出不正确: %s", out)
	}
}

// install 缺少必填库名时，应在触达数据库之前就失败
func TestInstallRequiresDatabase(t *testing.T) {
	_, _, err := runCmd("install")
	if err == nil || !strings.Contains(err.Error(), "--database") {
		t.Fatalf("应提示 --database 必填，实际: %v", err)
	}
}

func TestInstallRejectsBadDataType(t *testing.T) {
	_, _, err := runCmd("install", "--database", "whatever", "--data-type", "bogus")
	if err == nil || !strings.Contains(err.Error(), "--data-type") {
		t.Fatalf("应提示 --data-type 非法，实际: %v", err)
	}
}

func TestInstallRejectsBadDbType(t *testing.T) {
	_, _, err := runCmd("install", "--database", "whatever", "--db-type", "postgres")
	if err == nil || !strings.Contains(err.Error(), "MySQL") {
		t.Fatalf("应提示仅支持 MySQL，实际: %v", err)
	}
}

// --env-file 指向不存在的文件时必须报错，而不是静默回退到别的 .env
func TestEnvFileMissingFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.env")
	_, _, err := runCmd("config", "--env-file", missing)
	if err == nil || !strings.Contains(err.Error(), "配置文件不存在") {
		t.Fatalf("应提示配置文件不存在，实际: %v", err)
	}
}

// secret 默认只打印，且两个密钥必须是不同的 64 位十六进制串
func TestSecretCmdPrintsKeys(t *testing.T) {
	out, _, err := runCmd("secret")
	if err != nil {
		t.Fatalf("secret 执行失败: %v", err)
	}
	var access, refresh string
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(line, "JWT_ACCESS_SECRET="); ok {
			access = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "JWT_REFRESH_SECRET="); ok {
			refresh = strings.TrimSpace(v)
		}
	}
	if len(access) != 64 || len(refresh) != 64 {
		t.Fatalf("密钥长度应为 64，实际 access=%d refresh=%d\n%s", len(access), len(refresh), out)
	}
	if access == refresh {
		t.Fatal("access 与 refresh 密钥必须不同")
	}
}

// secret --write 必须就地替换已有的 JWT_* 行，且不动其它内容
func TestSecretWriteReplacesExistingKeys(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	original := "DB_NAME = demo\nJWT_ACCESS_SECRET = old\nJWT_REFRESH_SECRET = old\nGO_SERVER_PORT = 9090\n"
	if err := os.WriteFile(envPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runCmd("secret", "--write", "--env-file", envPath); err != nil {
		t.Fatalf("secret --write 执行失败: %v", err)
	}

	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, keep := range []string{"DB_NAME = demo", "GO_SERVER_PORT = 9090"} {
		if !strings.Contains(got, keep) {
			t.Fatalf("原有配置被破坏，缺少 %q:\n%s", keep, got)
		}
	}
	if strings.Contains(got, "old") {
		t.Fatalf("旧密钥未被替换:\n%s", got)
	}
	if strings.Count(got, "JWT_ACCESS_SECRET") != 1 || strings.Count(got, "JWT_REFRESH_SECRET") != 1 {
		t.Fatalf("密钥行数不正确:\n%s", got)
	}
}

// JWT 密钥缺失时 serve 必须拒绝启动，并给出生成命令提示
func TestServeRequiresJWTSecrets(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("DB_NAME = demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := runCmd("serve", "--env-file", envPath)
	if err == nil {
		t.Fatal("缺少 JWT 密钥时 serve 应返回错误")
	}
	for _, want := range []string{"JWT_ACCESS_SECRET", "secret --write"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("错误提示缺少 %q: %v", want, err)
		}
	}
}
