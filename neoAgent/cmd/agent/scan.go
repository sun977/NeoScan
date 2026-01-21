/*
 * @author: Sun977
 * @date: 2026.01.21
 * @description: Scan 模式子命令 (Standalone Mode)
 */

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	target    string
	portRange string
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "执行单次扫描任务 (Standalone)",
	Long: `在不连接 Master 的情况下执行一次性的扫描任务。
支持端口扫描、Web 指纹识别、漏洞检测等。

示例:
  neoAgent scan --target 192.168.1.1 --port 80,443
  neoAgent scan -t example.com -p 1-1000
`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: 这里的逻辑将在 Phase 3 实现
		fmt.Printf("Starting scan for target: %s, ports: %s\n", target, portRange)
		fmt.Println("Error: Scan logic not implemented yet (Phase 3)")
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Scan 命令特有的 Flags
	scanCmd.Flags().StringVarP(&target, "target", "t", "", "扫描目标 (IP/Domain/CIDR)")
	scanCmd.Flags().StringVarP(&portRange, "port", "p", "", "端口范围 (e.g. 80,443,1-1000)")

	// 必选参数标记
	scanCmd.MarkFlagRequired("target")
}
