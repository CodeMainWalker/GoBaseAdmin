package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/CodeMainWalker/GoBaseAdmin/internal/config"
	"github.com/CodeMainWalker/GoBaseAdmin/internal/install"
)

// newInstallCmd 命令行安装：flag → install.Params → install.Run → 打印结果/错误。
//
// 原网页安装向导（/core/install）已下线，首次部署与重新部署都走这里。
func newInstallCmd() *cobra.Command {
	var (
		params   install.Params
		dataType string
	)

	c := &cobra.Command{
		Use:   "install",
		Short: "初始化数据库并生成 .env（命令行安装）",
		Long: "初始化数据库并生成 .env。\n\n" +
			"安装向导：校验参数 → 按需建库 → 建表灌数据 → 写 .env → 建立连接。\n" +
			"数据库已装过时会拒绝执行；如需重装，请先删除 .env。",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// 安装要整包执行 SQL dump，逐条 SQL 日志会刷屏几百行且毫无意义，
			// 失败时的错误信息仍会从 install.Run 原样返回，不影响排查。
			if os.Getenv("SAI_SQL_LOG") == "" {
				_ = os.Setenv("SAI_SQL_LOG", "off")
			}

			if strings.TrimSpace(params.Database) == "" {
				return errors.New("请用 --database 指定数据库名")
			}
			params.DbType = install.NormalizeDbType(params.DbType)
			if params.DbType != "mysql" {
				return errors.New("当前仅支持 MySQL 数据库")
			}
			switch dataType {
			case "", install.DataTypePure:
				params.DataType = install.DataTypePure
			case install.DataTypeDemo:
				params.DataType = install.DataTypeDemo
			default:
				return errors.New("--data-type 只能是 pure 或 demo")
			}
			if params.Port <= 0 {
				params.Port = install.DBTypePorts["mysql"]
			}

			// 失败文案由 install 包给出（与上游 SaiAdmin 6.x 逐字一致），原样透出并以退出码 1 结束
			if err := install.Run(params); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"安装成功\n"+
					"  数据库: %s:%d/%s\n"+
					"  配置文件: %s（已随机生成 JWT 密钥）\n"+
					"  后台账号: admin / 123456\n"+
					"下一步: 执行 gobaseadmin serve 启动服务\n",
				params.Host, params.Port, params.Database, config.LocalEnvPath())
			return nil
		},
	}

	f := c.Flags()
	f.StringVar(&params.Host, "host", "127.0.0.1", "数据库地址")
	f.IntVar(&params.Port, "port", 3306, "数据库端口")
	f.StringVar(&params.Database, "database", "", "数据库名（必填，不存在时自动创建）")
	f.StringVar(&params.Username, "username", "root", "数据库用户名")
	f.StringVar(&params.Password, "password", "", "数据库密码")
	f.StringVar(&params.DbType, "db-type", "mysql", "数据库类型（当前仅支持 mysql）")
	f.StringVar(&dataType, "data-type", install.DataTypePure, "初始数据：pure=仅基础数据，demo=含演示数据")
	return c
}
