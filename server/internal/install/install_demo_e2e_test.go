package install

import (
	"os"
	"strconv"
	"testing"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// TestInstallDemoEndToEnd 验收 demo 版初始数据（比 pure 多 3 张 sa_article* 表）。
//
// 与 TestInstallEndToEnd 一样需要真实 MySQL 并默认跳过：
//
//	$env:SAI_E2E=1
//	$env:SAI_TEST_DB_HOST='127.0.0.1'; $env:SAI_TEST_DB_PORT='3306'
//	$env:SAI_TEST_DB_USER='root';      $env:SAI_TEST_DB_PASSWORD='123456'
//	$env:SAI_TEST_DB_NAME='saiadmin_demo_e2e'   # 会被清空并在结束时删除
//	go test ./internal/install -run TestInstallDemoEndToEnd -v
//
// demo 与 pure 共用同一套建表语句，差异只在演示数据，
// 这里重点确认「demo 的 INSERT 能被整包执行」以及表数量与 pure 一致（+3）。
func TestInstallDemoEndToEnd(t *testing.T) {
	if os.Getenv("SAI_E2E") == "" {
		t.Skip("跳过端到端安装测试：未设置 SAI_E2E=1")
	}
	t.Setenv("SAI_SQL_LOG", "off")

	host := envOr("SAI_TEST_DB_HOST", "127.0.0.1")
	port := envOr("SAI_TEST_DB_PORT", "3306")
	user := envOr("SAI_TEST_DB_USER", "root")
	pass := os.Getenv("SAI_TEST_DB_PASSWORD")
	// 默认库名与 pure 用例区分开，避免两个用例互相清库
	scratch := envOr("SAI_TEST_DB_NAME_DEMO", "saiadmin_demo_e2e")

	// 安装器把 .env 写在当前工作目录，切到临时目录避免污染 server/
	origin, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origin) }()

	config.SetEnvFile("")
	resetInstalledCache()
	store.Reset()

	requireScratchDB(t, host, port, user, pass, scratch)

	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("SAI_TEST_DB_PORT 不是端口号: %q", port)
	}
	p := Params{
		Host: host, Port: portNum, Database: scratch,
		Username: user, Password: pass,
		DbType: "mysql", DataType: DataTypeDemo,
	}
	if err := Run(p); err != nil {
		t.Fatalf("demo 安装失败: %v", err)
	}
	t.Cleanup(func() {
		store.Reset()
		tryDropScratch(t, host, port, user, pass, scratch)
	})

	var tableCount int
	if err := store.DB.Raw(
		"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", scratch,
	).Scan(&tableCount).Error; err != nil {
		t.Fatalf("统计表数量失败: %v", err)
	}
	// demo 版应为 21 张：pure 的 18 张 + 3 张 sa_article*
	if tableCount != 21 {
		t.Fatalf("demo 表数量异常，期望 21，实际 %d", tableCount)
	}

	var articleRows int
	if err := store.DB.Raw("SELECT COUNT(*) FROM sa_article").Scan(&articleRows).Error; err != nil {
		t.Fatalf("查询 sa_article 失败: %v", err)
	}
	if articleRows == 0 {
		t.Fatal("demo 版 sa_article 没有演示数据")
	}

	var menuCount int
	if err := store.DB.Raw("SELECT COUNT(*) FROM sa_system_menu").Scan(&menuCount).Error; err != nil {
		t.Fatalf("查询 sa_system_menu 失败: %v", err)
	}
	if menuCount != 76 {
		t.Fatalf("菜单数量异常，期望 76，实际 %d", menuCount)
	}

	var toolCount int
	if err := store.DB.Raw(
		"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME LIKE 'sa_tool%'",
		scratch,
	).Scan(&toolCount).Error; err != nil {
		t.Fatalf("统计工具表失败: %v", err)
	}
	if toolCount != 0 {
		t.Fatalf("demo 安装数据里仍含已下线的工具表: %d 张", toolCount)
	}

	t.Logf("demo 安装完成：%d 张表 / %d 条菜单 / sa_article %d 行", tableCount, menuCount, articleRows)
}
