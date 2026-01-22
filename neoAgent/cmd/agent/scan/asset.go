package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewAssetScanCmd() *cobra.Command {
	opts := options.NewAssetScanOptions()

	cmd := &cobra.Command{
		Use:   "asset",
		Short: "资产发现扫描",
		Long:  `对指定网段或 IP 进行资产存活探测、端口开放检测及指纹识别。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()

			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Asset Scan Task Created:\n%s\n", string(taskJSON))

			return nil
		},
	}

	flags := cmd.Flags()
	// 参数绑定：&变量地址，长参数名，默认值，默认值，帮助说明
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "扫描目标 (IP/CIDR)")
	flags.StringVarP(&opts.Port, "port", "p", opts.Port, "端口范围")
	flags.IntVar(&opts.Rate, "rate", opts.Rate, "扫描速率")
	flags.BoolVar(&opts.OSDetect, "os-detect", opts.OSDetect, "启用操作系统探测")
	flags.BoolVar(&opts.Ping, "ping", opts.Ping, "启用 Ping 存活探测")

	// 标记 target 为必填项
	cmd.MarkFlagRequired("target")

	return cmd
}
