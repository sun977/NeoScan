package scan

import (
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewSubdomainScanCmd() *cobra.Command {
	opts := options.NewSubdomainScanOptions()

	cmd := &cobra.Command{
		Use:   "subdomain",
		Short: "子域名扫描",
		Long:  `使用字典进行子域名枚举.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// TODO: SubdomainScanner 尚未实现。在真正的扫描器接入 RunnerManager 之前，
			// 明确报错而不是假装成功地只打印 Task JSON，避免误导用户以为已完成扫描。
			return fmt.Errorf("subdomain scan not implemented yet")
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Domain, "domain", "d", opts.Domain, "目标域名")
	flags.StringVar(&opts.Dict, "dict", opts.Dict, "字典文件路径")
	flags.IntVar(&opts.Threads, "threads", opts.Threads, "并发线程数")

	cmd.MarkFlagRequired("domain")

	return cmd
}
