package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Cfg 全局配置，字段与 server/.env 对齐
type Cfg struct {
	DBType     string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBPrefix   string
	DBCharset  string

	CacheMode   string
	RedisHost   string
	RedisPort   string
	RedisPass   string
	RedisDB     string
	CaptchaMode string

	// 服务自身监听端口
	ServerPort string

	// JWT 签名密钥（HS256）。access / refresh 必须不同，且必须显式配置，
	// 否则启动直接报错 —— 避免用默认弱密钥上线。
	JWTAccessSecret  string
	JWTRefreshSecret string
}

var C Cfg

// loadedFrom 记录本次实际加载的 .env 路径（空表示全部候选都不存在，退化为环境变量）
var loadedFrom string

// forcedEnvFile 由命令行 --env-file / GO_ENV_FILE 指定。
// 一旦指定就不再回退到其它候选文件：找不到就报错，避免「以为读了 A、其实读了 B」。
var forcedEnvFile string

// SetEnvFile 指定强制使用的 .env 路径，传空字符串表示清除。
//
// 这里刻意**不校验文件是否存在**：install 子命令要写的目标文件本来就还不存在，
// 存在性由 Load() 负责（serve 等命令在启动阶段报错即可）。
func SetEnvFile(p string) {
	if strings.TrimSpace(p) == "" {
		forcedEnvFile = ""
		return
	}
	if abs, err := filepath.Abs(p); err == nil {
		forcedEnvFile = abs
		return
	}
	forcedEnvFile = p
}

// ForcedEnvFile 返回当前强制指定的 .env 路径（空表示未指定）
func ForcedEnvFile() string { return forcedEnvFile }

// envCandidates 返回 .env 的候选路径，按优先级排列。
//
// Go 版有**自己独立的 .env**（server/.env，由安装器写入），排在最前。
func envCandidates() []string {
	list := []string{".env"}

	// 已编译的二进制可能在别的 cwd 下启动，exe 同目录的 .env 也认。
	// 但要跳过 `go run` 的临时构建目录（否则会误认临时目录里的文件）。
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if !strings.Contains(filepath.ToSlash(dir), "go-build") {
			list = append(list, filepath.Join(dir, ".env"))
		}
	}
	return list
}

// Load 读取 .env（server/.env，其次 exe 同目录，再退化为环境变量）
//
// 若通过 SetEnvFile 指定了 --env-file，则只认该文件，不存在直接报错。
func Load() error {
	if forcedEnvFile != "" {
		if _, err := os.Stat(forcedEnvFile); err != nil {
			return fmt.Errorf("配置文件不存在: %s", forcedEnvFile)
		}
		_ = godotenv.Load(forcedEnvFile)
		loadedFrom = forcedEnvFile
	} else {
		for _, p := range envCandidates() {
			if _, err := os.Stat(p); err == nil {
				_ = godotenv.Load(p)
				loadedFrom = p
				break
			}
		}
	}

	C = Cfg{
		DBType:      env("DB_TYPE", "mysql"),
		DBHost:      env("DB_HOST", "127.0.0.1"),
		DBPort:      env("DB_PORT", "3306"),
		DBName:      env("DB_NAME", "gobaseadmin"),
		DBUser:      env("DB_USER", "root"),
		DBPassword:  env("DB_PASSWORD", "123456"),
		DBPrefix:    env("DB_PREFIX", ""),
		DBCharset:   env("DB_CHARSET", "utf8mb4"),
		CacheMode:   env("CACHE_MODE", "file"),
		RedisHost:   env("REDIS_HOST", "127.0.0.1"),
		RedisPort:   env("REDIS_PORT", "6379"),
		RedisPass:   env("REDIS_PASSWORD", ""),
		RedisDB:     env("REDIS_DB", "0"),
		CaptchaMode: env("CAPTCHA_MODE", "cache"),
		ServerPort:  env("GO_SERVER_PORT", "8080"),

		JWTAccessSecret:  env("JWT_ACCESS_SECRET", ""),
		JWTRefreshSecret: env("JWT_REFRESH_SECRET", ""),
	}
	return nil
}

// NewJWTSecret 生成一个 64 位十六进制随机密钥（32 字节熵），供 install/secret 命令使用
func NewJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// MissingJWTSecrets 返回当前缺失的 JWT 密钥名（用于启动前校验）
func (c Cfg) MissingJWTSecrets() []string {
	var miss []string
	if strings.TrimSpace(c.JWTAccessSecret) == "" {
		miss = append(miss, "JWT_ACCESS_SECRET")
	}
	if strings.TrimSpace(c.JWTRefreshSecret) == "" {
		miss = append(miss, "JWT_REFRESH_SECRET")
	}
	return miss
}

// LoadedFrom 返回本次实际加载的 .env 路径
func LoadedFrom() string { return loadedFrom }

// LocalEnvPath 返回 Go 自己 .env 的绝对路径（安装器写入的目标）。
// 通过 --env-file 指定时以指定路径为准。
func LocalEnvPath() string {
	if forcedEnvFile != "" {
		return forcedEnvFile
	}
	abs, err := filepath.Abs(".env")
	if err != nil {
		return ".env"
	}
	return abs
}

// LocalEnvExists 判断 Go 自己的 .env 是否存在。
// 这是安装命令「程序已经安装」的判定依据。
func LocalEnvExists() bool {
	_, err := os.Stat(LocalEnvPath())
	return err == nil
}

// Reload 重新读取 .env 并刷新配置（安装完成后调用）
func Reload() error { return Load() }

// DSN 组装 MySQL 连接串（带库名）
func (c Cfg) DSN() string {
	return c.dsn(c.DBName)
}

// DSNWithoutDB 组装不带库名的连接串，用于「建库」阶段。
// MySQL 允许不指定 database 连接，这样即使目标库还不存在也能连上服务器。
func (c Cfg) DSNWithoutDB() string {
	return c.dsn("")
}

func (c Cfg) dsn(dbName string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&multiStatements=true",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, dbName, c.DBCharset,
	)
}

// BuildDSN 用显式参数组装 DSN。
// 安装器在建库前使用 —— 此时 server/.env 还不存在，拿不到 config.C。
func BuildDSN(host, port, user, pass, dbName, charset string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&multiStatements=true",
		user, pass, host, port, dbName, charset,
	)
}

func env(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	// .env 里可能存在 `KEY = value` 的空格形式，godotenv 已处理；这里再去掉一层引号
	v = strings.Trim(v, `"'`)
	return v
}

// EnvInt 读取整型环境变量
func EnvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
