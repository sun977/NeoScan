package scan

import (
	"context"
	"fmt"

	"neoagent/internal/core/pipeline"

	"github.com/spf13/cobra"
)

// RunOptions 定义 run 命令的参数
type RunOptions struct {
	Target      string
	Concurrency int
	AutoMode    bool
}

// NewRunScanCmd 创建 scan run 命令
func NewRunScanCmd() *cobra.Command {
	opts := &RunOptions{
		Concurrency: 5, // 默认并发 5 个 IP
		AutoMode:    true,
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "自动化全流程扫描",
		Long: `自动串联各个扫描模块，实现从主机发现到服务识别的全流程扫描。
支持 CIDR、IP 范围、IP 列表等多种目标输入。

流程: Target -> Alive -> Port -> Service -> OS -> Report`,
		Example: `  neoAgent scan run -t 192.168.1.0/24
  neoAgent scan run -t targets.txt --concurrency 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Target == "" {
				return fmt.Errorf("target is required")
			}

			fmt.Printf("[*] Starting Auto Pipeline on %s (Concurrency: %d)...\n", opts.Target, opts.Concurrency)

			// 初始化 AutoRunner
			runner := pipeline.NewAutoRunner(opts.Target, opts.Concurrency)

			// 执行
			if err := runner.Run(context.Background()); err != nil {
				return err
			}

			fmt.Println("[*] Pipeline completed.")
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (CIDR/IP/Range/File)")
	flags.IntVarP(&opts.Concurrency, "concurrency", "c", opts.Concurrency, "并发扫描的 IP 数量")
	flags.BoolVar(&opts.AutoMode, "auto", opts.AutoMode, "启用自动模式 (默认开启)")

	return cmd
}
