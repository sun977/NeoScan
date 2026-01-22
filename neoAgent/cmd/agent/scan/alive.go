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
		Use:   "alive",                 // 改为 alive，更贴切
		Short: "IP存活扫描 (ICMP/ARP/TCP)", // 后续实现多种协议探测 SYN 等
		Long:  `对目标进行存活探测(支持根据TTL映射OS).`,
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
	// 不带协议参数 -- 自动模式(根据是否同网段智能选择协议探测帧)
	// 带协议参数 -- 使用指定协议的探测帧
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (IP/CIDR)")
	flags.BoolVarP(&opts.EnableArp, "arp", "A", opts.EnableArp, "使用 ARP 探测 (仅同广播域有效)")
	flags.BoolVarP(&opts.EnableIcmp, "icmp", "I", opts.EnableIcmp, "使用 ICMP 探测 (可根据TTL映射OS)")
	flags.BoolVarP(&opts.EnableTcp, "tcp", "T", opts.EnableTcp, "使用 TCP Connect 探测")
	// 默认值传入 nil 以避免 Help 信息显示冗长的端口列表
	// 实际默认值会在 parseOptions 阶段注入 (见 internal/core/options/scan_alive.go)
	flags.IntSliceVarP(&opts.TcpPorts, "tcp-ports", "p", nil, "TCP 探测端口列表 (默认内置 Top 端口)")
	flags.IntVarP(&opts.Concurrency, "concurrency", "c", opts.Concurrency, "并发数")
	flags.BoolVarP(&opts.ResolveHostname, "resolve-hostname", "r", opts.ResolveHostname, "启用 Hostname 反向解析 (DNS PTR)")

	// NeoScan-agent scan alive -t 127.0.0.1 -T -p 90,445
	// 隐式逻辑：如果指定参数 -p,但是没有指定 -T 的情况下隐式开启 TCP 探测,此时 NeoScan-agent scan alive -t 127.0.0.1 -p 90,445 等同于 NeoScan-agent scan alive -t 127.0.0.1 -T -p 90,445
	// 如果指定参数 -p 和其他协议参数(icmp/arp) 则表示TCP协议和指定参数协议同时开启探测,但是结果取决于谁先返回。但这个时候TCP探测的端口只有 -p 参数指定的端口。
	// 例如：NeoScan-agent scan alive -t 127.0.0.1 -p 90,445 -I 表示同时开启 TCP 和 ICMP 探测,但是结果取决于谁先返回。
	// 如果携带了多个协议参数,则表示同时开启多个协议探测,但是结果取决于谁先返回。
	// IP存活探测会根据TTL粗略的映射OS,但仅作为参考。只有ICMP协议(ping)会携带TTL,其他协议(ARP/TCP)不会携带TTL,因此使用别的协议是大概率OS为空。OS探测交给后续单独的扫描。

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
