package scan

import (
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewDirScanCmd() *cobra.Command {
	opts := options.NewDirScanOptions()

	cmd := &cobra.Command{
		Use:   "dir",
		Short: "目录扫描",
		Long:  `使用字典进行 Web 目录爆破扫描.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// TODO: DirScanner 尚未实现。在真正的扫描器接入 RunnerManager 之前，
			// 明确报错而不是假装成功地只打印 Task JSON，避免误导用户以为已完成扫描。
			return fmt.Errorf("dir scan not implemented yet")
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "目标 URL")
	flags.StringVarP(&opts.Dict, "dict", "d", opts.Dict, "字典文件路径")
	flags.StringVarP(&opts.Extensions, "extensions", "e", opts.Extensions, "文件后缀 (e.g. php,jsp)")
	flags.IntVar(&opts.Threads, "threads", opts.Threads, "并发线程数")

	cmd.MarkFlagRequired("target")

	return cmd
}
