package scan

import (
	"context"
	"fmt"

	"neoagent/internal/core/model"
	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// NewOsScanCmd 创建操作系统扫描子命令
func NewOsScanCmd() *cobra.Command {
	opts := options.NewOsScanOptions()

	var cmd = &cobra.Command{
		Use:   "os",
		Short: "操作系统识别",
		Long:  `通过 TCP/IP 协议栈指纹识别目标操作系统类型.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			pterm.Info.Printf("Starting OS detection: %s (Mode: %s)...\n", opts.Target, opts.Mode)

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()

			// 初始化 RunnerManager（工厂内已完成全部原子扫描器的统一注册，
			// CLI 与 Master 调度共用同一份注册表，避免能力不一致）
			manager := runner.NewRunnerManager()

			results, err := manager.Execute(context.Background(), task)
			if err != nil {
				return fmt.Errorf("scan failed: %v", err)
			}

			// 提取结果供 JSON/CSV 输出复用（保持原有输出行为不变）
			// 注意：OsRunner 将扫描期间的错误封装进了 TaskResult.Error 而非直接返回 error，
			// 这里显式检查一次，避免静默吞掉失败状态。
			var result interface{}
			if len(results) > 0 {
				result = results[0].Result
				if results[0].Status == model.TaskStatusFailed {
					return fmt.Errorf("scan failed: %s", results[0].Error)
				}
			}

			// 输出结果 (使用 ConsoleReporter)
			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

			// 保存 JSON 结果
			if opts.Output.OutputJson != "" {
				saveJsonResult(opts.Output.OutputJson, result)
			}
			// 保存 CSV 结果 (OsScanner 结果结构可能需要适配)
			// OsInfo 现已实现 TabularData 接口，可以被 CSV Reporter 支持
			if opts.Output.OutputCsv != "" {
				if err := reporter.SaveCsvResult(opts.Output.OutputCsv, results); err != nil {
					fmt.Printf("[-] Failed to save csv: %v\n", err)
				}
			}

			return nil
		},
	}

	// 绑定 Flags
	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (IP)")
	flags.StringVarP(&opts.Mode, "mode", "m", "auto", "扫描模式 (fast, deep, auto)")
	cmd.MarkFlagRequired("target")

	return cmd
}
