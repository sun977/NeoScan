package scan

import (
	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

var globalOutputOptions options.OutputOptions

// NewScanCmd 创建 scan 父命令
func NewScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "执行扫描任务",
		Long: `执行各类扫描任务,如IP探活,端口扫描,服务扫描,OS识别,Web扫描,漏洞扫描,目录/子域名挖掘等。
请使用具体的子命令.`,
	}

	// 定义持久化 Flags (所有子命令都可用)
	pFlags := cmd.PersistentFlags()
	// 注意: Shorthand 必须是单个字符。这里我们只注册长参数。
	pFlags.StringVar(&globalOutputOptions.OutputCsv, "oc", "", "指定保存csv文件路径")
	pFlags.StringVar(&globalOutputOptions.OutputJson, "oj", "", "指定保存json文件路径")

	// // 注册别名 (Hidden flags) 方便用户使用简短命令
	// pFlags.StringVar(&globalOutputOptions.OutputCsv, "oc", "", "outputCsv 简写")
	// pFlags.Lookup("oc").Hidden = true
	// pFlags.StringVar(&globalOutputOptions.OutputJson, "oj", "", "outputJson 简写")
	// pFlags.Lookup("oj").Hidden = true

	// 注册子命令
	cmd.AddCommand(NewIpAliveScanCmd())   // IP存活扫描 (ICMP/ARP)
	cmd.AddCommand(NewPortScanCmd())      // 端口扫描
	cmd.AddCommand(NewServiceScanCmd())   // 服务识别
	cmd.AddCommand(NewOsScanCmd())        // 操作系统识别
	cmd.AddCommand(NewWebScanCmd())       // Web综合扫描
	cmd.AddCommand(NewVulnScanCmd())      // 漏洞扫描
	cmd.AddCommand(NewDirScanCmd())       // 目录/文件挖掘
	cmd.AddCommand(NewSubdomainScanCmd()) // 子域名挖掘

	return cmd
}
