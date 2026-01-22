package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

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

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()

			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("OS Scan Task Created:\n%s\n", string(taskJSON))
			return nil
		},
	}

	// 绑定 Flags
	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (IP)")
	cmd.MarkFlagRequired("target")

	return cmd
}
