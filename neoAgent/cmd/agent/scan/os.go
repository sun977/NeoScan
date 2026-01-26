package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"
	"neoagent/internal/core/scanner/os"

	"github.com/spf13/cobra"
)

// NewOsScanCmd 创建操作系统扫描子命令
func NewOsScanCmd() *cobra.Command {
	opts := options.NewOsScanOptions()

	var cmd = &cobra.Command{
		Use:   "os",
		Short: "操作系统识别",
		Long:  `通过 TCP/IP 协议栈指纹识别目标操作系统类型.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			fmt.Printf("Starting OS Scan against %s (Mode: %s)...\n", opts.Target, opts.Mode)

			// 实例化扫描器
			scanner := os.NewScanner()

			// 执行扫描
			result, err := scanner.Scan(cmd.Context(), opts.Target, opts.Mode)
			if err != nil {
				return fmt.Errorf("scan failed: %v", err)
			}

			// 输出结果
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			fmt.Printf("Scan Result:\n%s\n", string(resultJSON))
			return nil
		},
	}

	// 绑定 Flags
	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (IP)")
	flags.StringVarP(&opts.Mode, "mode", "m", "auto", "扫描模式 (fast, deep, auto)")
	cmd.MarkFlagRequired("target")

	return cmd
}
