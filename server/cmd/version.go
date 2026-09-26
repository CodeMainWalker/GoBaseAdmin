package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// 版本信息集中在这里，是全局唯一的版本来源
const (
	appName    = "gobaseadmin"
	appVersion = "6.1.5"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "打印版本信息",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (%s %s/%s)\n",
				appName, appVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
}
