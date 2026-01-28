/*
 * @author: Sun977
 * @date: 2026.01.21
 * @description: Cobra Root Command 定义
 */

package main

import (
	"fmt"
	"neoagent/cmd/agent/proxy"
	"neoagent/cmd/agent/scan"
	"neoagent/internal/config"
	"neoagent/internal/pkg/logger"
	"os"

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
	if flag == nil || !flag.Changed {
		// 如果没有显式设置，不初始化日志，或者初始化为 Fatal 级别以静默输出
		// 这里我们选择初始化为 Fatal，这样只有 Fatal 级别的日志会输出（几乎等于静默）
		// 同时这避免了 LoggerInstance 为 nil 导致的潜在 panic
		logConfig := &config.LogConfig{
			Level:  "fatal",
			Format: "text",
			Output: "stdout",
			Caller: false,
		}
		logger.InitLogger(logConfig)
		return
	}

	logLevel := viper.GetString("log.level")
	if logLevel == "" {
		logLevel = "info"
	}

	// 构造一个简单的 LogConfig 用于 CLI
	logConfig := &config.LogConfig{
		Level:  logLevel,
		Format: "text",
		Output: "stdout",
		Caller: false, // CLI 模式通常不需要调用者信息，除非 debug
	}

	if logLevel == "debug" {
		logConfig.Caller = true
	}

	// 初始化日志
	if _, err := logger.InitLogger(logConfig); err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
	}
}
