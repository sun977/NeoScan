package proxy

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

// NewProxyCmd 创建 proxy 命令
func NewProxyCmd() *cobra.Command {
	opts := options.NewProxyOptions()

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "启动代理或端口转发服务",
		Long: `启动代理服务，支持 Socks5、HTTP 代理和端口转发模式。
此模式通常作为后台服务运行。

示例:
  neoAgent proxy --mode socks5 --listen :1080
  neoAgent proxy --mode http --listen :8080 --auth user:pass
  neoAgent proxy --mode port_forward --listen :8080 --forward 192.168.1.100:80`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 验证参数
			if err := opts.Validate(); err != nil {
				return err
			}

			// 2. 转换为 Task 模型
			task := opts.ToTask()

			// 3. 打印调试信息 (模拟执行)
			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Proxy Task Created:\n%s\n", string(taskJSON))

			// TODO: 调用真正的 Runner
			return nil
		},
	}

	// 绑定 Flags
	flags := cmd.Flags()
	flags.StringVar(&opts.Mode, "mode", opts.Mode, "代理模式: socks5, http, port_forward")
	flags.StringVarP(&opts.Listen, "listen", "l", opts.Listen, "监听地址")
	flags.StringVar(&opts.Auth, "auth", opts.Auth, "认证信息 (user:pass)")
	flags.StringVarP(&opts.Forward, "forward", "f", opts.Forward, "转发目标 (仅 port_forward 模式)")

	return cmd
}
