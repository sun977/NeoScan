package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewWebScanCmd() *cobra.Command {
	opts := options.NewWebScanOptions()

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Web 综合扫描",
		Long:  `对 Web 服务进行综合扫描，包括指纹、路径、漏洞等。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}
			task := opts.ToTask()
			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Web Scan Task Created:\n%s\n", string(taskJSON))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "目标 URL/IP")
	flags.StringVarP(&opts.Ports, "ports", "p", opts.Ports, "端口范围")
	flags.StringVar(&opts.Path, "path", opts.Path, "扫描路径")
	flags.StringVarP(&opts.Method, "method", "m", opts.Method, "HTTP 方法")

	cmd.MarkFlagRequired("target")

	return cmd
}
