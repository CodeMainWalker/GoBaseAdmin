package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
)

// newSecretCmd 生成 JWT 签名密钥。
//
// 默认只打印到标准输出；--write 会把两个密钥写入 .env
// （已有的 JWT_* 行会被就地替换，其余内容保持不变）。
func newSecretCmd() *cobra.Command {
	var write bool

	cmd := &cobra.Command{
		Use:   "secret",
		Short: "生成 JWT 签名密钥（--write 直接写入 .env）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			access, err := config.NewJWTSecret()
			if err != nil {
				return err
			}
			refresh, err := config.NewJWTSecret()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()

			if !write {
				fmt.Fprintf(out, "JWT_ACCESS_SECRET=%s\n", access)
				fmt.Fprintf(out, "JWT_REFRESH_SECRET=%s\n", refresh)
				fmt.Fprintf(out, "\n把它们写入 %s 后重启服务即可生效。\n", config.LocalEnvPath())
				return nil
			}

			path := config.LocalEnvPath()
			raw, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("读取配置文件失败: %w", err)
			}
			updated := setEnvLine(string(raw), "JWT_ACCESS_SECRET", access)
			updated = setEnvLine(updated, "JWT_REFRESH_SECRET", refresh)
			if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("写入配置文件失败: %w", err)
			}
			fmt.Fprintf(out, "已写入 %s\n", path)
			fmt.Fprintf(out, "JWT_ACCESS_SECRET=%s\n", access)
			fmt.Fprintf(out, "JWT_REFRESH_SECRET=%s\n", refresh)
			return nil
		},
	}

	cmd.Flags().BoolVar(&write, "write", false, "直接写入 .env（默认只打印）")
	return cmd
}

// setEnvLine 替换 .env 中某个键的值；键不存在时追加到末尾。
func setEnvLine(content, key, value string) string {
	lines := strings.Split(content, "\n")
	replaced := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, key) {
			continue
		}
		if rest := strings.TrimSpace(strings.TrimPrefix(trimmed, key)); strings.HasPrefix(rest, "=") {
			lines[i] = fmt.Sprintf("%s = %s", key, value)
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, fmt.Sprintf("%s = %s", key, value))
	}
	return strings.Join(lines, "\n")
}
