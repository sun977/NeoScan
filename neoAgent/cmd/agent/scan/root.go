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
		Long: `执行各类扫描任务，如资产发现、端口扫描、Web 扫描等。
请使用具体的子命令。`,
	}

	// 定义持久化 Flags (所有子命令都可用)
	pFlags := cmd.PersistentFlags()
	// 注意: Shorthand 必须是单个字符。这里我们只注册长参数。
	pFlags.StringVar(&globalOutputOptions.OutputCsv, "outputCsv", "", "指定保存csv文件路径[以.csv结尾] (alias: --oc)")
	pFlags.StringVar(&globalOutputOptions.OutputJson, "outputJson", "", "指定保存json文件路径[以.json结尾]，格式为JSON Lines (alias: --oj)")

	// 注册别名 (Hidden flags) 方便用户使用简短命令
	pFlags.StringVar(&globalOutputOptions.OutputCsv, "oc", "", "outputCsv 简写")
	pFlags.Lookup("oc").Hidden = true
	pFlags.StringVar(&globalOutputOptions.OutputJson, "oj", "", "outputJson 简写")
	pFlags.Lookup("oj").Hidden = true

	// 注册子命令
	cmd.AddCommand(NewIpAliveScanCmd())
	cmd.AddCommand(NewPortScanCmd())
	cmd.AddCommand(NewWebScanCmd())
	cmd.AddCommand(NewDirScanCmd())
	cmd.AddCommand(NewSubdomainScanCmd())
	cmd.AddCommand(NewVulnScanCmd())

	return cmd
}
