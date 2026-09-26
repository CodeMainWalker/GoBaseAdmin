package store

import (
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局 GORM 实例。
//
// 未安装（或数据库暂时不可用）时为 nil —— 调用方必须先判断 Ready()，
// 否则会空指针 panic。HTTP 层由中间件 InstallGuard 统一拦截。
var DB *gorm.DB

// ready 标记全局连接是否可用
var ready atomic.Bool

func gormConfig() *gorm.Config {
	level := logger.Warn
	// 跑测试/搬数据时不想被逐条 SQL 刷屏，可设 SAI_SQL_LOG=off
	if os.Getenv("SAI_SQL_LOG") == "off" {
		level = logger.Silent
	}
	return &gorm.Config{
		Logger: logger.New(log.New(log.Writer(), "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
		}),
		// 表名不做复数化转换，模型里已显式指定 TableName()
		NamingStrategy: nil,
		NowFunc:        func() time.Time { return time.Now().Local() },
	}
}

// Open 用指定 DSN 打开连接（不校验连通性）。
// 安装器建库阶段需要 connect-without-database，故单独暴露此函数。
func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), gormConfig())
}

// Init 依据当前配置建立全局连接，并探测可用性。
// 失败时不会留下半可用的 DB，Ready() 保持 false。
func Init() error {
	return InitWithDSN(config.C.DSN())
}

// InitWithDSN 用指定 DSN 初始化全局连接
func InitWithDSN(dsn string) error {
	ready.Store(false)

	db, err := Open(dsn)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return err
	}

	DB = db
	ready.Store(true)
	return nil
}

// Ready 数据库是否已就绪（已安装且可连通）
func Ready() bool { return ready.Load() && DB != nil }

// Reset 关闭并清空全局连接（重新安装时使用）
func Reset() {
	ready.Store(false)
	if DB != nil {
		if sqlDB, err := DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	DB = nil
}
