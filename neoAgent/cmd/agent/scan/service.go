package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

// NewServiceScanCmd 创建服务扫描子命令
func NewServiceScanCmd() *cobra.Command {
	opts := options.NewServiceScanOptions()

	var cmd = &cobra.Command{
		Use:   "service",
		Short: "服务识别",
		Long:  `对指定目标的开放端口进行深度服务识别 (Banner Grab/Protocol Handshake).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()

			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Service Scan Task Created:\n%s\n", string(taskJSON))
			return nil
		},
	}

	// 绑定 Flags
	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (IP)")
	flags.StringVarP(&opts.Port, "port", "p", "", "目标端口 (e.g. 80,443)")
	cmd.MarkFlagRequired("target")

	return cmd
}
