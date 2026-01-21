package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewPortScanCmd() *cobra.Command {
	opts := options.NewPortScanOptions()

	cmd := &cobra.Command{
		Use:   "port",
		Short: "详细端口扫描",
		Long:  `对指定目标的特定端口进行详细扫描和服务版本识别。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()

			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Port Scan Task Created:\n%s\n", string(taskJSON))

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "扫描目标")
	flags.StringVarP(&opts.Port, "port", "p", opts.Port, "端口范围")
	flags.IntVar(&opts.Rate, "rate", opts.Rate, "扫描速率")
	flags.BoolVar(&opts.ServiceDetect, "service-detect", opts.ServiceDetect, "启用服务版本识别")

	cmd.MarkFlagRequired("target")
	cmd.MarkFlagRequired("port")

	return cmd
}
