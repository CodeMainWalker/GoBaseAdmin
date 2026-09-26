package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/install"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/router"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/store"
)

// debugMode 是否开启调试模式（GO_DEBUG=false 可关闭）
func debugMode() bool {
	if os.Getenv("GO_DEBUG") == "false" {
		return false
	}
	return true
}

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "启动 HTTP 服务",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe()
		},
	}
}

func runServe() error {
	if err := config.Load(); err != nil {
		return err
	}
	// JWT 密钥必须显式配置：缺失就拒绝启动，而不是悄悄用默认值
	if miss := config.C.MissingJWTSecrets(); len(miss) > 0 {
		return fmt.Errorf("缺少 JWT 密钥配置: %s\n请执行以下任一操作后重试：\n"+
			"  1. gobaseadmin secret --write   # 生成密钥并写入 %s\n"+
			"  2. 在 .env 中手动填写（64 位十六进制字符串）",
			strings.Join(miss, ", "), config.LocalEnvPath())
	}
	addr := ":" + config.C.ServerPort

	// 数据库连不上时**不退出**：可能尚未安装，先把服务起起来，
	// 由 middleware.InstallGuard 统一返回「尚未安装」的 JSON 提示。
	if err := store.Init(); err != nil {
		log.Printf("数据库未就绪: %v", err)
		if config.LocalEnvExists() {
			log.Printf("已存在配置文件 %s，请检查数据库配置或确认数据库已启动", config.LoadedFrom())
		} else {
			log.Printf("检测到尚未安装，请执行 gobaseadmin install --database <库名> --username <用户> --password <密码> 完成安装")
		}
	} else {
		log.Printf("数据库已连接: %s@%s:%s/%s",
			config.C.DBUser, config.C.DBHost, config.C.DBPort, config.C.DBName)
		if p := config.LoadedFrom(); p != "" {
			log.Printf("配置来源: %s", p)
		}
		if !install.Installed() {
			log.Printf("警告: 数据库 %s 中没有 sa_system_menu 表，请先执行 gobaseadmin install", config.C.DBName)
		}
	}

	srv := &http.Server{Addr: addr, Handler: router.Setup(debugMode())}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	log.Printf("GoBaseAdmin 服务已启动: http://127.0.0.1%s", addr)

	// 等待 Ctrl+C / SIGTERM，收到后优雅关闭（10 秒超时）
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("服务启动失败: %w", err)
		}
	case <-ctx.Done():
		log.Printf("收到退出信号，正在关闭服务…")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("服务关闭异常: %v", err)
		}
	}

	store.Reset()
	log.Printf("服务已退出")
	return nil
}
