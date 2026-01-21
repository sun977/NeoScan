package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewDirScanCmd() *cobra.Command {
	opts := options.NewDirScanOptions()

	cmd := &cobra.Command{
		Use:   "dir",
		Short: "目录扫描",
		Long:  `使用字典进行 Web 目录爆破扫描。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}
			task := opts.ToTask()
			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Dir Scan Task Created:\n%s\n", string(taskJSON))
			return nil
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
