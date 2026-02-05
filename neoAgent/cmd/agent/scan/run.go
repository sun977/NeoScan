package scan

import (
	"context"

	"neoagent/internal/core/options"
	"neoagent/internal/core/pipeline"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// NewRunScanCmd 创建 scan run 命令
func NewRunScanCmd() *cobra.Command {
	// 使用 Core Options
	opts := options.NewScanRunOptions()

	cmd := &cobra.Command{
		Use:   "run",
		Short: "自动化全流程扫描",
		Long: `自动串联各个扫描模块，实现从主机发现到服务识别的全流程扫描。
支持 CIDR、IP 范围、IP 列表等多种目标输入。

流程: Target -> Alive -> Port -> Service -> OS -> [Brute] -> Report`,
		Example: `  neoAgent scan run -t 192.168.1.0/24
  neoAgent scan run -t 10.0.0.1 --brute
  neoAgent scan run -t 10.0.0.1 --brute --users root,admin --pass 123456`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 处理默认端口逻辑，用于显示
			displayPort := opts.PortRange
			if displayPort == "top1000" {
				displayPort = "top1000 (default)"
			}

			pterm.Info.Printf("开始全流程扫描: %s (Concurrency: %d, Ports: %s)...\n", opts.Target, opts.Concurrency, displayPort)
			if opts.EnableBrute {
				pterm.Info.Println("爆破模块: 已启用")
				if opts.BruteUsers != "" {
					pterm.Info.Printf("自定义用户: %s\n", opts.BruteUsers)
				}
				if opts.BrutePass != "" {
					pterm.Info.Printf("自定义密码: %s\n", opts.BrutePass)
				}
			} else {
				pterm.Info.Println("爆破模块: 未启用 (使用 --brute 开启)")
			}

			// 初始化 AutoRunner
			runner := pipeline.NewAutoRunner(opts)

			// 执行
			if err := runner.Run(context.Background()); err != nil {
				return err
			}

			pterm.Info.Println("全流程扫描完成.")
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (CIDR/IP/Range/File)")
	flags.IntVarP(&opts.Concurrency, "concurrency", "c", opts.Concurrency, "并发扫描的 IP 数量")
	flags.StringVarP(&opts.PortRange, "port", "p", opts.PortRange, "端口范围 (默认 top1000)")
	flags.BoolVar(&opts.ShowSummary, "summary", false, "显示扫描结果汇总 (默认关闭)")

	// 爆破参数
	flags.BoolVar(&opts.EnableBrute, "brute", false, "启用弱口令爆破 (默认关闭)")
	flags.StringVar(&opts.BruteUsers, "users", "", "自定义爆破用户名 (逗号分隔或文件)")
	flags.StringVar(&opts.BrutePass, "pass", "", "自定义爆破密码 (逗号分隔或文件)")

	return cmd
}
