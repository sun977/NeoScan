package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
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

			// 4. 输出结果 (使用 ConsoleReporter)
			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

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
	flags.StringVar(&opts.Strategy, "strategy", opts.Strategy, "探测策略: auto (自动), manual (手动)")
	flags.BoolVar(&opts.EnableArp, "arp", opts.EnableArp, "手动模式: 启用 ARP 探测")
	flags.BoolVar(&opts.EnableIcmp, "icmp", opts.EnableIcmp, "手动模式: 启用 ICMP 探测")
	flags.BoolVar(&opts.EnableTcp, "tcp", opts.EnableTcp, "手动模式: 启用 TCP 全连接探测")
	flags.IntSliceVar(&opts.TcpPorts, "tcp-ports", opts.TcpPorts, "TCP 探测端口列表")
	flags.IntVarP(&opts.Concurrency, "concurrency", "c", opts.Concurrency, "并发数")

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
