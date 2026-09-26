package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
)

// newConfigCmd 打印当前生效的配置。
// 排查「到底读了哪个 .env」时最有用；密码只显示是否已设置。
func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "打印当前生效的配置（密码脱敏）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Load(); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			src := config.LoadedFrom()
			if src == "" {
				src = "(无 .env，使用环境变量与默认值)"
			}
			pass := "(空)"
			if config.C.DBPassword != "" {
				pass = "***"
			}
			mask := func(v string) string {
				if v == "" {
					return "(未配置)"
				}
				return "***"
			}

			fmt.Fprintf(out, "配置来源: %s\n", src)
			fmt.Fprintf(out, "DB_TYPE=%s\n", config.C.DBType)
			fmt.Fprintf(out, "DB_HOST=%s\n", config.C.DBHost)
			fmt.Fprintf(out, "DB_PORT=%s\n", config.C.DBPort)
			fmt.Fprintf(out, "DB_NAME=%s\n", config.C.DBName)
			fmt.Fprintf(out, "DB_USER=%s\n", config.C.DBUser)
			fmt.Fprintf(out, "DB_PASSWORD=%s\n", pass)
			fmt.Fprintf(out, "DB_CHARSET=%s\n", config.C.DBCharset)
			fmt.Fprintf(out, "CACHE_MODE=%s\n", config.C.CacheMode)
			fmt.Fprintf(out, "CAPTCHA_MODE=%s\n", config.C.CaptchaMode)
			fmt.Fprintf(out, "GO_SERVER_PORT=%s\n", config.C.ServerPort)
			fmt.Fprintf(out, "JWT_ACCESS_SECRET=%s\n", mask(config.C.JWTAccessSecret))
			fmt.Fprintf(out, "JWT_REFRESH_SECRET=%s\n", mask(config.C.JWTRefreshSecret))
			return nil
		},
	}
}
