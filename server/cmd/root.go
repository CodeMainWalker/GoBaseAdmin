// Package cmd 提供 gobaseadmin 的命令行入口（cobra）。
//
// 命令一览：
//
//	gobaseadmin serve    启动 HTTP 服务
//	gobaseadmin install  初始化数据库并生成 .env（替代原网页安装向导）
//	gobaseadmin secret   生成 JWT 签名密钥
//	gobaseadmin version  打印版本
//	gobaseadmin config   打印当前生效的配置
//
// 无参数时只打印帮助（退出码 0），不会隐式启动服务。
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
)

// envFile 对应全局参数 --env-file（也可用 GO_ENV_FILE 环境变量）
var envFile string

// newRootCmd 构造根命令。
// 每次调用都返回全新的命令树，便于测试里反复构造互不干扰。
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gobaseadmin",
		Short: "GoBaseAdmin 后端服务",
		Long: "GoBaseAdmin 后端服务（基于上游 SaiAdmin 6.x 接口契约的 Go 重写版）。\n\n" +
			"启动服务使用 serve 子命令；首次部署先执行 install 完成数据库初始化。",
		// 出错时不打印用法与错误（由 Execute 统一输出，避免重复）
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       appVersion,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if envFile == "" {
				envFile = os.Getenv("GO_ENV_FILE")
			}
			// 只登记路径，不做存在性校验：install 要写的文件本来就还不存在
			config.SetEnvFile(envFile)
			return nil
		},
	}
	// 不注册 cobra 自带的 completion 子命令
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().StringVar(&envFile, "env-file", "",
		"指定 .env 配置文件路径（默认 server/.env，也可用环境变量 GO_ENV_FILE）")

	root.AddCommand(newServeCmd(), newInstallCmd(), newSecretCmd(), newVersionCmd(), newConfigCmd())
	return root
}

// Execute 运行根命令并返回进程退出码。
//
// 根命令刻意**不实现 Run/RunE**：cobra 对不可运行的根命令在无参数时会打印帮助
// 并返回 flag.ErrHelp（退出码 0）；而未知子命令会走 Find 报错（退出码 1）。
func Execute() int {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		return 1
	}
	return 0
}
