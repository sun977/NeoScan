package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"neoagent/internal/core/options"
	"neoagent/internal/core/runner"
	"neoagent/internal/core/scanner/alive"

	"github.com/spf13/cobra"
)

// NewIpAliveScanCmd 创建 IP存活扫描 命令
func NewIpAliveScanCmd() *cobra.Command {
	opts := options.NewIpAliveScanOptions()

	var cmd = &cobra.Command{
		Use:   "alive",             // 改为 alive，更贴切
		Short: "IP存活扫描 (ICMP/ARP)", // 后续实现多种协议探测 SYN 等
		Long:  `仅对目标进行存活探测.使用 ICMP 或 ARP 协议。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()

			// 1. 初始化 RunnerManager
			manager := runner.NewRunnerManager()
			// 2. 注册 IpAliveScanner
			manager.Register(alive.NewIpAliveScanner())

			// 3. 执行任务
			fmt.Printf("[*] Starting IP Alive Scan on %s...\n", task.Target)
			results, err := manager.Execute(context.Background(), task)
			if err != nil {
				return err
			}

			// 4. 输出结果
			if len(results) == 0 {
				fmt.Println("[-] No alive hosts found.")
			} else {
				fmt.Printf("[+] Found %d alive hosts:\n", len(results))
				for _, res := range results {
					jsonBytes, _ := json.Marshal(res.Result)
					fmt.Printf("    %s\n", string(jsonBytes))
				}
			}

			// TODO: 实现 OutputReporter (CSV/JSON/Excel)
			if opts.Output.OutputJson != "" {
				saveJsonResult(opts.Output.OutputJson, results)
			}

			return nil
		},
	}

	// 绑定 Flags
	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (IP/CIDR)")
	flags.BoolVar(&opts.Ping, "ping", opts.Ping, "启用 Ping 存活探测")

	return cmd
}

func saveJsonResult(path string, data interface{}) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("[-] Failed to create output file: %v\n", err)
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		fmt.Printf("[-] Failed to write json output: %v\n", err)
	}
	fmt.Printf("[+] Results saved to %s\n", path)
}
