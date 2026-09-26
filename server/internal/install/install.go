// Package install 实现首次部署的**命令行**安装（原网页向导已下线）。
//
// 当前实现要点：
//   - 建表与初始数据直接执行内嵌的 SQL dump（sql/*.sql），
//     dump 里已含建表语句与初始数据（包括管理员账号），不依赖迁移框架
//   - 安装产物只有一个 .env（数据库连接、缓存方式、服务端口、JWT 密钥）
//   - 装完在当前进程内重载配置并重建连接，立即可用，无需重启进程
package install

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"

	"gorm.io/gorm"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

//go:embed sql/*.sql
var sqlFS embed.FS

// 数据来源类型
const (
	DataTypeDemo = "demo" // 含演示数据（部门/岗位/用户等）
	DataTypePure = "pure" // 仅基础数据（一个管理员 + 菜单权限）
)

// DBTypePorts 支持的数据库类型与默认端口。
// 当前仅实现 MySQL —— SQL dump 是 MySQL 方言（反引号、SET NAMES utf8mb4）。
var DBTypePorts = map[string]int{"mysql": 3306}

// Params 安装参数
type Params struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	DbType   string
	DataType string
}

var identRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// installedFlag 只在**确认已安装**时置位（正向缓存）。
//
// 刻意不做负向缓存：未安装时每次探一次 information_schema 的代价可忽略，
// 但一旦缓存「未安装」，另一个进程执行完 install 后本进程就再也感知不到，
// 必须重启 —— 那正是网页向导时代才有的限制。
var installedFlag atomic.Bool

// NormalizeDbType 归一化数据库类型，只放行 mysql
func NormalizeDbType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if t == "mysql" || t == "" {
		return "mysql"
	}
	return t
}

// Installed 判断程序是否已安装（未安装时服务只返回提示，不处理业务请求）。
//
// 判定顺序：
//  1. Go 自己的 .env（server/.env，或 exe 同目录的 .env / 显式指定的 --env-file）
//     存在即视为已安装 —— 配置也可能全部来自环境变量，此时必须走第 2 步；
//  2. 当前连接可用时，再确认库中确实有 sa_system_menu，
//     避免出现「连着一个空库却提示程序已经安装」的情况。
func Installed() bool {
	if config.LocalEnvExists() {
		return true
	}
	if !store.Ready() {
		return false
	}
	if installedFlag.Load() {
		return true
	}
	if tableExists(store.DB, "sa_system_menu") {
		installedFlag.Store(true)
		return true
	}
	return false
}

// resetInstalledCache 清掉正向缓存（仅测试使用）
func resetInstalledCache() { installedFlag.Store(false) }

// Run 执行安装。全程紧凑：建库 → 建表灌数据 → 写 .env → 重建连接。
//
// 校验顺序与上游 SaiAdmin 6.x 的安装接口保持一致，
// 保证同一场景下返回的提示文案完全相同（命令行安装直接透出这些文案）。
func Run(p Params) error {
	// 1. 已安装 / 配置目录可写
	if config.LocalEnvExists() {
		return errors.New("管理后台已经安装！如需重新安装，请删除根目录env配置文件并重启")
	}
	if !isWritable(filepath.Dir(config.LocalEnvPath())) {
		return errors.New("权限认证失败")
	}

	// 2. 参数校验
	// 库名要拼进建库语句（标识符无法参数绑定），只放行字母数字下划线
	if !identRe.MatchString(p.Database) {
		return errors.New("数据库名只能包含字母、数字和下划线")
	}
	if strings.TrimSpace(p.Host) == "" || strings.TrimSpace(p.Username) == "" {
		return errors.New("数据库地址和用户名不能为空")
	}
	if p.DbType != "mysql" {
		return errors.New("当前仅支持 MySQL 数据库")
	}
	if p.Port <= 0 {
		p.Port = DBTypePorts["mysql"]
	}
	charset := "utf8mb4"

	// 3. 不指定库连接服务器，按需建库
	if err := ensureDatabase(p, charset); err != nil {
		return err
	}

	// 4. 连到目标库
	target := config.BuildDSN(p.Host, itoa(p.Port), p.Username, p.Password, p.Database, charset)
	db, err := store.Open(target)
	if err != nil {
		return errors.New("数据库连接失败：" + err.Error())
	}

	// 5. 已装检查（库中已有 sa_system_menu 即视为已安装）
	if tableExists(db, "sa_system_menu") {
		return errors.New("数据库已经安装，请勿重复安装")
	}

	// 6. 建表 + 灌初始数据
	if err := execSQLDump(db, p.DataType); err != nil {
		return errors.New("数据表初始化失败：" + err.Error())
	}

	// 7. 写 Go 自己的 .env
	if err := writeEnv(p, charset); err != nil {
		return errors.New("配置文件写入失败：" + err.Error())
	}

	// 8. 刷新配置并建立全局连接（装完立即可用，无需重启）
	if err := config.Reload(); err != nil {
		return err
	}
	if err := store.InitWithDSN(target); err != nil {
		return errors.New("安装完成但数据库连接失败：" + err.Error())
	}
	return nil
}

// isWritable 判断目录是否可写。
// 需要写入的是 .env，所以检查它所在的目录能否创建临时文件。
func isWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".saiadmin_write_test_*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// ensureDatabase 连接服务器并确保目标库存在
func ensureDatabase(p Params, charset string) error {
	// MySQL 允许不指定 database 连接，便于先建库
	rootDSN := config.BuildDSN(p.Host, itoa(p.Port), p.Username, p.Password, "", charset)
	db, err := store.Open(rootDSN)
	if err != nil {
		return translateDBError(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return errors.New("数据库连接失败：" + err.Error())
	}
	// Ping 一次以尽早暴露认证/网络类错误，从而给出与上游一致的提示文案
	if err := sqlDB.Ping(); err != nil {
		return translateDBError(err)
	}
	defer sqlDB.Close()

	if databaseExists(db, p.Database) {
		return nil
	}
	stmt := fmt.Sprintf("CREATE DATABASE `%s` CHARSET utf8mb4 COLLATE utf8mb4_general_ci", p.Database)
	if err := db.Exec(stmt).Error; err != nil {
		return translateDBError(err)
	}
	return nil
}

func databaseExists(db *gorm.DB, name string) bool {
	var n int
	err := db.Raw("SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", name).Scan(&n).Error
	if err != nil {
		return false
	}
	return n > 0
}

func tableExists(db *gorm.DB, table string) bool {
	var n int
	err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?",
		table,
	).Scan(&n).Error
	if err != nil {
		return false
	}
	return n > 0
}

// execSQLDump 执行内嵌的 SQL dump（建表 + 初始数据）。
// 依赖 DSN 里的 multiStatements=true，可整包执行。
func execSQLDump(db *gorm.DB, dataType string) error {
	file := "sql/saiadmin-pure.sql"
	if dataType == DataTypeDemo {
		file = "sql/saiadmin-demo.sql"
	}
	raw, err := sqlFS.ReadFile(file)
	if err != nil {
		return fmt.Errorf("内置 SQL 文件缺失: %s", file)
	}
	sql := string(raw)
	if strings.TrimSpace(sql) == "" {
		return errors.New("内置 SQL 文件为空")
	}
	return db.Exec(sql).Error
}

// writeEnv 写入 Go 自己的 .env（server/.env），并随机生成一对 JWT 密钥
func writeEnv(p Params, charset string) error {
	access, err := config.NewJWTSecret()
	if err != nil {
		return err
	}
	refresh, err := config.NewJWTSecret()
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`# 由安装器生成 —— GoBaseAdmin 后端配置
# 数据库配置
DB_TYPE = mysql
DB_HOST = %s
DB_PORT = %d
DB_NAME = %s
DB_USER = %s
DB_PASSWORD = %s
DB_PREFIX =
DB_CHARSET = %s

# 缓存方式
CACHE_MODE = file

# Redis配置
REDIS_HOST = 127.0.0.1
REDIS_PORT = 6379
REDIS_PASSWORD = ''
REDIS_DB = 0

# 验证码配置
CAPTCHA_MODE = cache

# 服务监听端口
GO_SERVER_PORT = 8080

# JWT 签名密钥（access / refresh 必须不同，泄露后请重新生成）
JWT_ACCESS_SECRET = %s
JWT_REFRESH_SECRET = %s
`, p.Host, p.Port, p.Database, p.Username, p.Password, charset, access, refresh)

	return os.WriteFile(config.LocalEnvPath(), []byte(content), 0o644)
}

// translateDBError 把驱动错误转成与上游 SaiAdmin 6.x 一致的提示文案。
//
// 注意判定顺序：MySQL 的 1045（密码错）与 1044（无库权限）报错里
// 都含 "Access denied for user"，必须先用错误码把两者分开。
func translateDBError(err error) error {
	msg := err.Error()

	// 认证失败：MySQL 1045 / PostgreSQL 28P01；无库权限：MySQL 1044。
	// 两者都以 "Access denied for user" 开头，必须先按错误码区分（1044 含 "to database"）。
	if strings.Contains(msg, "Error 1044") ||
		strings.Contains(msg, "to database") ||
		strings.Contains(msg, "permission denied to create database") {
		return errors.New("当前数据库用户没有建库权限，请手动创建数据库后重试")
	}
	if strings.Contains(msg, "Error 1045") ||
		strings.Contains(msg, "(using password") ||
		strings.Contains(msg, "password authentication failed") ||
		strings.Contains(msg, "Access denied for user") {
		return errors.New("数据库用户名或密码错误")
	}
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connectex") ||
		strings.Contains(msg, "No connection could be made") {
		return errors.New("Connection refused. 请确认数据库IP端口是否正确，数据库已经启动")
	}
	if strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "timeout") {
		return errors.New("数据库连接超时，请确认数据库IP端口是否正确，安全组及防火墙已经放行端口")
	}
	if strings.Contains(msg, "Unknown database") || strings.Contains(msg, "Error 1049") {
		return errors.New("数据库不存在，请确认库名或由安装程序自动创建")
	}
	return errors.New("数据库连接失败：" + msg)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
