/*
 * @author: Sun977
 * @date: 2026.01.21
 * @description: Cobra Root Command 定义
 */

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "neoAgent",
	Short: "NeoScan 分布式安全扫描代理",
	Long: `NeoAgent 是 NeoScan 系统的执行单元。
它可以作为分布式 Worker 连接到 Master 集群，
也可以作为独立的命令行扫描工具运行。

示例:
  # 启动服务模式 (默认)
  neoAgent server

  # 运行单次扫描
  neoAgent scan -t 192.168.1.1 -p 80,443

  # 加入集群
  neoAgent join 10.0.0.1:8080 --token abcdef
`,
	// 默认行为：如果不带参数，显示帮助信息，而不是启动 Server。
	// 这是一个设计变更：显式优于隐式。
	// 但为了兼容旧脚本，如果没有任何参数，我们暂时在此处可以做特殊处理，
	// 或者强制用户使用 `neoAgent server`。
	// 考虑到 Linus 原则 "Never break userspace"，我们将在 Execute 中处理向后兼容性。
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
	rootCmd.PersistentFlags().String("log-level", "info", "日志级别 (debug, info, warn, error)")

	// 绑定 Viper
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
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
