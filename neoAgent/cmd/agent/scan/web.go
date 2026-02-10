package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"neoagent/internal/core/factory"
	"neoagent/internal/core/options"

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

			// 1. 创建 Scanner
			scanner := factory.NewWebScanner()

			// 2. 构造 Task
			task := opts.ToTask()
			// 注入截图参数 (CLI 参数 -> Task Params)
			task.Params["screenshot"] = screenshot

			fmt.Printf("[*] Starting Web Scan against %s (Ports: %s)...\n", opts.Target, opts.Ports)
			
			// 3. 执行扫描
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			results, err := scanner.Run(ctx, task)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// 4. 输出结果
			for _, res := range results {
				resJSON, _ := json.MarshalIndent(res.Result, "", "  ")
				fmt.Printf("\n[+] Scan Result:\n%s\n", string(resJSON))
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

	cmd.MarkFlagRequired("target")

	return cmd
}
