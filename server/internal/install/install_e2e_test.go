package install

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// TestInstallEndToEnd 是端到端安装验收，需要真实 MySQL，默认跳过。
//
// 运行方式（PowerShell）：
//
//	$env:SAI_E2E=1
//	$env:SAI_TEST_DB_HOST='127.0.0.1'; $env:SAI_TEST_DB_PORT='3306'
//	$env:SAI_TEST_DB_USER='root';      $env:SAI_TEST_DB_PASSWORD='123456'
//	go test ./internal/install -run TestInstallEndToEnd -v
//
// 会用 SAI_TEST_DB_NAME（默认 saiadmin_install_e2e）作为临时库，
// 测试开始先 DROP，结束再 DROP，不会影响既有数据。
func TestInstallEndToEnd(t *testing.T) {
	if os.Getenv("SAI_E2E") == "" {
		t.Skip("跳过端到端安装测试：未设置 SAI_E2E=1")
	}
	// 安装会执行整份 SQL dump，逐条日志没有意义
	t.Setenv("SAI_SQL_LOG", "off")

	host := envOr("SAI_TEST_DB_HOST", "127.0.0.1")
	port := envOr("SAI_TEST_DB_PORT", "3306")
	user := envOr("SAI_TEST_DB_USER", "root")
	pass := os.Getenv("SAI_TEST_DB_PASSWORD")
	scratch := envOr("SAI_TEST_DB_NAME", "saiadmin_install_e2e")
	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("SAI_TEST_DB_PORT 不是端口号: %q", port)
	}

	// 安装器把 .env 写在「当前工作目录」，测试里切到临时目录，
	// 避免污染 server/ 以及误判本机是否已安装。
	origin, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origin) }()

	// 走 cwd 的 .env，不受外部 --env-file / 缓存影响
	config.SetEnvFile("")
	resetInstalledCache()

	// 先确保临时库存在。这一步同时用来探测账号有没有建库权限 ——
	// 没有权限就直接 skip 并给出需要执行的授权 SQL，而不是给出误导性的失败。
	requireScratchDB(t, host, port, user, pass, scratch)

	p := Params{
		Host: host, Port: portNum, Database: scratch,
		Username: user, Password: pass,
		DbType: "mysql", DataType: DataTypePure,
	}

	// ---- 1. 首次安装 ----
	if err := Run(p); err != nil {
		t.Fatalf("安装失败: %v", err)
	}
	t.Cleanup(func() {
		store.Reset()
		tryDropScratch(t, host, port, user, pass, scratch)
	})

	if !store.Ready() {
		t.Fatal("安装后全局连接未就绪")
	}
	if !Installed() {
		t.Fatal("安装后 Installed() 应为 true")
	}

	// ---- 2. .env 已写入且指向临时库 ----
	envPath := filepath.Join(tmp, ".env")
	b, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("未生成 .env: %v", err)
	}
	if !strings.Contains(string(b), "DB_NAME = "+scratch) {
		t.Fatalf(".env 内容不正确:\n%s", string(b))
	}
	if !strings.Contains(string(b), "DB_PORT = "+port) {
		t.Fatalf(".env 端口不正确:\n%s", string(b))
	}

	// ---- 3. 表结构与初始数据 ----
	var tableCount int
	if err := store.DB.Raw(
		"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", scratch,
	).Scan(&tableCount).Error; err != nil {
		t.Fatalf("统计表数量失败: %v", err)
	}
	// pure 版应为 18 张 sa_system_* 表（工具表已随前端一起下线）
	if tableCount != 18 {
		t.Fatalf("表数量异常，期望 18，实际 %d", tableCount)
	}

	// 已下线的工具表不应再被创建（防止 SQL dump 回退）
	var toolCount int
	if err := store.DB.Raw(
		"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME LIKE 'sa_tool%'",
		scratch,
	).Scan(&toolCount).Error; err != nil {
		t.Fatalf("统计工具表失败: %v", err)
	}
	if toolCount != 0 {
		t.Fatalf("安装数据里仍含已下线的工具表: %d 张", toolCount)
	}

	var menuCount int
	if err := store.DB.Raw("SELECT COUNT(*) FROM sa_system_menu").Scan(&menuCount).Error; err != nil {
		t.Fatalf("查询 sa_system_menu 失败: %v", err)
	}
	// 与「工具/插件市场菜单已下线」后的现有库一致：76 条
	if menuCount != 76 {
		t.Fatalf("菜单数量异常，期望 76，实际 %d", menuCount)
	}

	var toolMenuCount int
	if err := store.DB.Raw(
		"SELECT COUNT(*) FROM sa_system_menu WHERE path LIKE '/tool%' OR code LIKE 'Tool%' OR code = 'Plugin'",
	).Scan(&toolMenuCount).Error; err != nil {
		t.Fatalf("查询工具菜单失败: %v", err)
	}
	if toolMenuCount != 0 {
		t.Fatalf("安装数据里仍含工具/插件市场菜单: %d 条", toolMenuCount)
	}

	var adminCount int
	if err := store.DB.Raw(
		"SELECT COUNT(*) FROM sa_system_user WHERE username = 'admin' AND delete_time IS NULL",
	).Scan(&adminCount).Error; err != nil {
		t.Fatalf("查询管理员失败: %v", err)
	}
	if adminCount != 1 {
		t.Fatalf("管理员账号数量异常: %d", adminCount)
	}
	t.Logf("安装完成：%d 张表 / %d 条菜单 / 管理员 %d 个", tableCount, menuCount, adminCount)

	// ---- 4. 重复安装：.env 存在时的提示（与上游 SaiAdmin 6.x 一致）----
	err = Run(p)
	if err == nil || !strings.Contains(err.Error(), "管理后台已经安装") {
		t.Fatalf("重复安装应提示「管理后台已经安装！」，实际: %v", err)
	}

	// ---- 5. 重复安装：.env 被删除但库里已有表 ----
	if err := os.Remove(envPath); err != nil {
		t.Fatal(err)
	}
	store.Reset()
	err = Run(p)
	if err == nil || !strings.Contains(err.Error(), "数据库已经安装") {
		t.Fatalf("库已存在表时应提示「数据库已经安装」，实际: %v", err)
	}
}

// requireScratchDB 确保临时库存在；账号没有建库权限且库也不存在时 skip 并给出授权 SQL。
//
// 两种情况都算可用：库已存在（只需账号对该库有权限），或账号能建库。
// 这样在受限账号（只有库级权限）下也能用「已经建好的空库」跑完整流程。
func requireScratchDB(t *testing.T, host, port, user, pass, name string) {
	t.Helper()
	db, sqlDB, err := openServer(t, host, port, user, pass)
	if err != nil {
		t.Fatalf("连接数据库服务器失败: %v", err)
	}
	defer sqlDB.Close()

	var exists int
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", name,
	).Scan(&exists).Error; err != nil {
		t.Fatalf("查询库是否存在失败: %v", err)
	}

	if exists == 0 {
		err = db.Exec("CREATE DATABASE `" + name + "` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci").Error
		if err != nil {
			t.Skipf("库 %s 不存在，且当前 MySQL 账号 %s 没有建库权限，跳过完整安装验证（%v）。\n"+
				"请在 MySQL 中执行后重跑：\n"+
				"  CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;\n"+
				"  GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost';\n"+
				"  FLUSH PRIVILEGES;",
				name, user, err, name, name, user)
		}
	}

	// 清空库中可能残留的表，保证是全新安装
	var tables []string
	if err := db.Raw(
		"SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", name,
	).Scan(&tables).Error; err == nil {
		for _, tb := range tables {
			_ = db.Exec("DROP TABLE IF EXISTS `" + name + "`.`" + tb + "`")
		}
	}
}

// tryDropScratch 尽力删除临时库；账号没有 DROP 权限时只告警，不影响断言结果。
func tryDropScratch(t *testing.T, host, port, user, pass, name string) {
	t.Helper()
	db, sqlDB, err := openServer(t, host, port, user, pass)
	if err != nil {
		t.Logf("清理临时库失败（连接不上）：%v", err)
		return
	}
	defer sqlDB.Close()
	if err := db.Exec("DROP DATABASE IF EXISTS `" + name + "`").Error; err != nil {
		t.Logf("清理临时库失败（可能没有 DROP 权限，请手工删除 %s）：%v", name, err)
	}
}

// openServer 连接「不指定库」的 MySQL 服务器连接
func openServer(t *testing.T, host, port, user, pass string) (*gorm.DB, *sql.DB, error) {
	t.Helper()
	db, err := store.Open(config.BuildDSN(host, port, user, pass, "", "utf8mb4"))
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, sqlDB, nil
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
