package scan

import (
	"github.com/spf13/cobra"
)

// NewScanCmd 创建 scan 父命令
func NewScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "执行扫描任务",
		Long: `执行各类扫描任务，如资产发现、端口扫描、Web 扫描等。
请使用具体的子命令。`,
	}

	// 注册子命令
	cmd.AddCommand(NewAssetScanCmd())
	cmd.AddCommand(NewPortScanCmd())
	cmd.AddCommand(NewWebScanCmd())
	cmd.AddCommand(NewDirScanCmd())
	cmd.AddCommand(NewSubdomainScanCmd())
	cmd.AddCommand(NewVulnScanCmd())

	return cmd
}
