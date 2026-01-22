package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

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

			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("IP Alive Scan Task Created:\n%s\n", string(taskJSON))
			return nil
		},
	}

	// 绑定 Flags
	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", "", "扫描目标 (IP/CIDR)")
	flags.BoolVar(&opts.Ping, "ping", opts.Ping, "启用 Ping 存活探测")

	return cmd
}
