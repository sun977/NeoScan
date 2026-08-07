package scan

import (
	"context"
	"fmt"

	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/spf13/cobra"
)

func NewWebScanCmd() *cobra.Command {
	opts := options.NewWebScanOptions()
	var screenshot bool

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Web 综合扫描",
		Long:  `对 Web 服务进行综合扫描，包括指纹、路径、漏洞等。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 构造 Task
			task := opts.ToTask()
			// 注入截图参数 (CLI 参数 -> Task Params)
			task.Params["screenshot"] = screenshot

			// 初始化 RunnerManager（工厂内已完成全部原子扫描器的统一注册，
			// CLI 与 Master 调度共用同一份注册表，避免能力不一致）
			manager := runner.NewRunnerManager()

			fmt.Printf("[*] Starting Web Scan against %s (Ports: %s)...\n", opts.Target, opts.Ports)

			// 超时必须用 task.Timeout（ToTask() 里已经根据任务类型设好），不能在这里
			// 另外写死一个数字：深度爬取的耗时随 --crawl-depth 指数增长，固定 2 分钟会导致
			// 深度调高后爬虫在中途被 ctx 取消、结果不完整，且没有任何报错提示用户。
			ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
			defer cancel()

			results, err := manager.Execute(ctx, task)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// 输出结果
			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

			// 保存 JSON 结果 (如果指定)
			if globalOutputOptions.OutputJson != "" {
				saveJsonResult(globalOutputOptions.OutputJson, results)
			}
			// 保存 CSV 结果 (如果指定)
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
	flags.StringVar(&opts.Path, "path", opts.Path, "扫描路径")
	flags.StringVarP(&opts.Method, "method", "m", opts.Method, "HTTP 方法")

	// 添加截图参数
	flags.BoolVar(&screenshot, "screenshot", false, "启用网页截图")

	// 深度爬取参数
	flags.StringVar(&opts.Crawl, "crawl", opts.Crawl, "是否启用深度爬取: auto(默认，自动判断)/true/false")
	flags.IntVar(&opts.CrawlDepth, "crawl-depth", opts.CrawlDepth, "爬取深度（仅 --crawl=true 时生效）")

	cmd.MarkFlagRequired("target")

	return cmd
}
