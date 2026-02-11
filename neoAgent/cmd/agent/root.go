/*
 * @author: Sun977
 * @date: 2026.01.21
 * @description: Cobra Root Command 定义
 */

package main

import (
	"fmt"
	"io"
	"neoagent/cmd/agent/proxy"
	"neoagent/cmd/agent/scan"
	"neoagent/internal/config"
	"neoagent/internal/pkg/logger"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "NeoScan-Agent",
	Short: "NeoScan-Agent 分布式安全扫描代理",
	Long: `NeoScan-Agent 是 NeoScan 系统的执行单元。
它可以作为分布式 Worker 连接到 Master 集群,也可以作为独立的CLI扫描工具运行.

示例:
  1.启动服务模式(默认)
	NeoSacn server
  2.加入服务集群
	NeoScan server --master 10.0.0.1:8080 --token mysecrettoken
  3.单机运行扫描
	NeoSacn scan [scan_mode] [mode_ops] -t <target_ip>
	NeoScan scan port -t 192.168.1.1 -p 80,443,1-1000 -s --oj output.json
`,
	// PersistentPreRun: 全局初始化逻辑，确保所有子命令都能使用日志
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initCLILogger(cmd)
	},
}

func Execute() {
	// 全局 Panic Recovery (Linus Style: Catch everything, even stupid user errors)
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\n[FATAL] Agent crashed unexpectedly: %v\n", r)
			// 在 Debug 模式下打印堆栈，但在生产环境只显示友好的错误
			// 避免吓坏用户
			// debug.Stack() // 如果需要堆栈
			os.Exit(1)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局 Flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认: ./configs/config.yaml)")
	rootCmd.PersistentFlags().String("log-level", "", "日志级别 (debug, info, warn, error)")

	// 绑定 Viper
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))

	// 注册子命令
	rootCmd.AddCommand(proxy.NewProxyCmd())
	rootCmd.AddCommand(scan.NewScanCmd())
}

// initConfig 读取配置文件和环境变量
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("configs")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // 读取环境变量

	if err := viper.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// initCLILogger 初始化 CLI 模式下的日志
// 这确保了 CLI 命令也能输出格式化的日志，并且受 --log-level 控制
func initCLILogger(cmd *cobra.Command) {
	// 检查 log-level 标志是否被显式设置
	flag := cmd.Flags().Lookup("log-level")
	level := "fatal" // 默认只输出 Fatal
	if flag != nil && flag.Changed {
		level = flag.Value.String()
	}

	// 配置 pterm
	switch level {
	case "debug":
		pterm.EnableDebugMessages()
		// pterm 没有 EnableInfoMessages，它是默认开启的，除非被 Disable
		// 如果我们之前 disable 了，现在需要 enable 吗？pterm 似乎没有提供直接的 API
		// 但 pterm.Info.Printer.Writer = os.Stdout 可以恢复
		// 简单起见，我们只控制 Debug。Info 默认开启。
	case "info":
		pterm.DisableDebugMessages()
	case "warn", "error", "fatal":
		pterm.DisableDebugMessages()
		// 禁用 Info 输出
		// pterm.Info 是 PrefixPrinter，没有直接暴露 Printer 或 Writer
		// 但我们可以设置 DisableOutput = true
		// 注意: pterm 全局没有 DisableInfoMessages，但可以通过设置 pterm.Info 的属性来禁用
		// 或者，我们只需要知道 Info 是用于展示过程的，如果不需要看过程，直接禁用
		// 实际上 pterm 提供了 DisableOutput() 方法来全局禁用
		// 但我们只想禁用 Info
		// 查阅文档/源码：pterm.Info.Writer = io.Discard (如果 Writer 是公开的)
		// 如果没有，我们可能无法简单禁用 Info 除非不调用它。
		// 鉴于我们是在 Scanner 里面调用的，那里有 pterm.PrintInfoMessages 的逻辑吗？没有。

		// 替代方案：pterm.Info = *pterm.Info.WithWriter(io.Discard)
		pterm.Info = *pterm.Info.WithWriter(io.Discard)
	}

	logConfig := &config.LogConfig{
		Level:  level,
		Format: "text",
		Output: "stdout",
		Caller: false,
	}

	// 初始化日志
	if _, err := logger.InitLogger(logConfig); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
	}
}
