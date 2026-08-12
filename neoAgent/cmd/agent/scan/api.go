package scan

import (
	"context"
	"fmt"

	"neoagent/internal/core/model"
	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/spf13/cobra"
)

// NewApiScanCmd 创建 api_scan 命令。
//
// ApiScan 总是深度爬取子页面（无 --crawl 开关，见 API扫描功能设计.md 第五节），
// 参数细节见 docs/API扫描-js提取/API扫描实施文档.md 第十四节。
func NewApiScanCmd() *cobra.Command {
	opts := options.NewApiScanOptions()

	cmd := &cobra.Command{
		Use:   "api",
		Short: "API 接口扫描",
		Long:  "对目标页面（及深度爬取子页面）提取接口调用地址。",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			task := opts.ToTask()
			manager := runner.NewRunnerManager()

			fmt.Printf("[*] Starting API Scan against %s (Ports: %s)...\n", opts.Target, opts.Ports)

			// 超时用 task.Timeout（ToTask() 里已设好），原因同 web.go：写死的短超时
			// 会在开启深度爬取时把任务中途砍断，与 --crawl-depth 语义相悖。
			ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
			defer cancel()

			results, err := manager.Execute(ctx, task)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// 打印汇总统计行
			printApiScanSummary(results)

			// 表格输出（通过 TabularData 接口）
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
	flags.IntVar(&opts.CrawlDepth, "crawl-depth", opts.CrawlDepth, "深度爬取的层数（ApiScan 总是深度爬取，无需 --crawl 开关）")
	flags.IntVar(&opts.MaxJSFiles, "max-js-files", opts.MaxJSFiles, "单页最多下载的外链 JS 文件数")

	cmd.MarkFlagRequired("target")
	return cmd
}

// printApiScanSummary 在表格前打印一行扫描汇总信息：
// 爬取页面数、提取接口总数（按置信度分层统计）、是否有截断。
func printApiScanSummary(results []*model.TaskResult) {
	if len(results) == 0 {
		return
	}

	var pages, totalAPIs, highCount, medCount, lowCount int
	var hasTruncated bool

	for _, r := range results {
		ar, ok := r.Result.(*model.ApiResult)
		if !ok {
			continue
		}
		pages++
		for _, api := range ar.APIs {
			totalAPIs++
			switch api.Confidence {
			case "high":
				highCount++
			case "medium":
				medCount++
			case "low":
				lowCount++
			}
		}
		if ar.APIsTruncated {
			hasTruncated = true
		}
	}

	truncStr := ""
	if hasTruncated {
		truncStr = " [some pages truncated, increase --max-js-files to scan more]"
	}

	fmt.Printf("[+] API Scan done: %d pages crawled, %d APIs found (high=%d medium=%d low=%d)%s\n",
		pages, totalAPIs, highCount, medCount, lowCount, truncStr)
}
