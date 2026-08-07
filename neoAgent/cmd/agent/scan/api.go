package scan

import (
	"context"
	"fmt"
	"time"

	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/spf13/cobra"
)

// NewApiScanCmd 创建 api_scan 命令骨架。
//
// 风险提示文案（下载外链 JS 文件是主动网络行为）由 Web-JS接口提取实施
// 文档.md 第二节补充完整，本步骤先给出可编译运行的最小骨架，Short/Long
// 占位文本会在那份文档实施时被替换，不是最终文案。
func NewApiScanCmd() *cobra.Command {
	opts := options.NewApiScanOptions()

	cmd := &cobra.Command{
		Use:   "api",
		Short: "API 接口扫描（骨架阶段，JS 提取逻辑待实现）",
		Long:  "对目标页面（及可选的深度爬取子页面）提取接口调用地址。当前为骨架版本，尚未接入真实的 JS 提取逻辑。",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			task := opts.ToTask()
			manager := runner.NewRunnerManager()

			fmt.Printf("[*] Starting API Scan against %s (Ports: %s)...\n", opts.Target, opts.Ports)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			results, err := manager.Execute(ctx, task)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

			if globalOutputOptions.OutputJson != "" {
				saveJsonResult(globalOutputOptions.OutputJson, results)
			}
			if globalOutputOptions.OutputCsv != "" {
				if err := reporter.SaveCsvResult(globalOutputOptions.OutputCsv, results); err != nil {
					fmt.Printf("[-] Failed to save csv: %v\n", err)
				}
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "目标 URL/IP")
	flags.StringVarP(&opts.Ports, "ports", "p", opts.Ports, "端口范围")
	flags.StringVar(&opts.Crawl, "crawl", opts.Crawl, "是否深度爬取子页面（auto/true/false）")
	flags.IntVar(&opts.CrawlDepth, "crawl-depth", opts.CrawlDepth, "深度爬取的层数（仅 crawl=true 时生效）")

	cmd.MarkFlagRequired("target")
	return cmd
}
