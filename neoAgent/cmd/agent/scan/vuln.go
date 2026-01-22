package scan

import (
	"encoding/json"
	"fmt"

	"neoagent/internal/core/options"

	"github.com/spf13/cobra"
)

func NewVulnScanCmd() *cobra.Command {
	opts := options.NewVulnScanOptions()

	cmd := &cobra.Command{
		Use:   "vuln",
		Short: "漏洞扫描 (YAML模板)",
		Long:  `使用 Nuclei 模板进行漏洞扫描。`, // 原生支持 YAML 模板漏洞扫描，同时支持调用 Nuclei 进行扫描(工具调用不属于原生支持)
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			// 注入全局输出参数
			opts.Output = globalOutputOptions

			task := opts.ToTask()
			taskJSON, _ := json.MarshalIndent(task, "", "  ")
			fmt.Printf("Vuln Scan Task Created:\n%s\n", string(taskJSON))
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "扫描目标")
	flags.StringVar(&opts.Templates, "templates", opts.Templates, "模板路径")
	flags.StringVar(&opts.Severity, "severity", opts.Severity, "漏洞等级过滤")

	cmd.MarkFlagRequired("target")

	return cmd
}
