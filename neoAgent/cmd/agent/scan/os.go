package scan

import (
	"fmt"

	"neoagent/internal/core/model"
	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/scanner/os"

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

			// 实例化扫描器
			scanner := os.NewScanner()

			// 执行扫描
			result, err := scanner.Scan(cmd.Context(), opts.Target, opts.Mode)
			if err != nil {
				return fmt.Errorf("scan failed: %v", err)
			}

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			// 构造 TaskResult (为了复用 Reporter)
			taskResult := &model.TaskResult{
				TaskID: "", // 临时 ID
				Status: model.TaskStatusSuccess,
				Result: result, // result 是 *model.OsInfo
			}
			results := []*model.TaskResult{taskResult}

			// 4. 输出结果 (使用 ConsoleReporter)
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
