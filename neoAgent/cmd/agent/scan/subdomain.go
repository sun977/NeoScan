package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewSubdomainScanCmd() *cobra.Command {
	opts := options.NewSubdomainScanOptions()

	cmd := &cobra.Command{
		Use:   "subdomain",
		Short: "子域名扫描",
		Long:  `使用字典或 API 进行子域名枚举。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}
			task := opts.ToTask()
			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Subdomain Scan Task Created:\n%s\n", string(taskJSON))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Domain, "domain", "d", opts.Domain, "目标域名")
	flags.StringVar(&opts.Dict, "dict", opts.Dict, "字典文件路径")
	flags.IntVar(&opts.Threads, "threads", opts.Threads, "并发线程数")

	cmd.MarkFlagRequired("domain")

	return cmd
}
