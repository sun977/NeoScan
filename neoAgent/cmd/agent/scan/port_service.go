package scan

import (
	"context"
	"fmt"

	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"
	"neoagent/internal/core/scanner/port_service"

	"github.com/spf13/cobra"
)

func NewPortScanCmd() *cobra.Command {
	opts := options.NewPortScanOptions()

	cmd := &cobra.Command{
		Use:   "port",
		Short: "详细端口扫描",
		Long:  `对指定目标的特定端口进行详细扫描和服务版本识别.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()

			// 1. 初始化 RunnerManager
			manager := runner.NewRunnerManager()
			// 2. 注册 PortServiceScanner
			manager.Register(port_service.NewPortServiceScanner())

			// 3. 执行任务
			fmt.Printf("[*] Starting Port Scan on %s (Ports: %s)...\n", task.Target, task.PortRange)
			results, err := manager.Execute(context.Background(), task)
			if err != nil {
				return err
			}

			// 4. 输出结果 (使用 ConsoleReporter)
			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

			// 保存 JSON 结果
			if opts.Output.OutputJson != "" {
				saveJsonResult(opts.Output.OutputJson, results)
			}

			// 保存 CSV 结果
			if opts.Output.OutputCsv != "" {
				if err := reporter.SaveCsvResult(opts.Output.OutputCsv, results); err != nil {
					fmt.Printf("[-] Failed to save csv: %v\n", err)
				}
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "扫描目标")
	flags.StringVarP(&opts.Port, "port", "p", opts.Port, "端口范围 (e.g., 80,443,1-1000)")
	flags.IntVarP(&opts.Rate, "rate", "r", opts.Rate, "扫描速率 (并发数)")
	flags.BoolVarP(&opts.ServiceDetect, "service-detect", "s", opts.ServiceDetect, "启用服务版本识别")

	cmd.MarkFlagRequired("target")
	cmd.MarkFlagRequired("port")

	return cmd
}
