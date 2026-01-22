/*
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent主程序入口
 * @func: 初始化应用、启动服务器、等待中断信号
 * @architecture: 参考Master的架构模式，简化main函数职责
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"neoagent/internal/app/agent"
)

// AgentOptions Agent启动选项
type AgentOptions struct {
	ConfigPath string // 配置文件路径
	LogLevel   string // 日志级别
	ShowHelp   bool   // 显示帮助信息
	ShowVersion bool  // 显示版本信息
	Daemon     bool   // 后台运行模式
}

func main() {
	// 解析命令行参数
	opts := parseFlags()
	
	// 处理特殊参数
	if opts.ShowHelp {
		flag.Usage()
		return
	}
	
	if opts.ShowVersion {
		showVersion()
		return
	}

	// 创建Agent应用实例
	app, err := agent.NewApp()
	if err != nil {
		log.Fatalf("Failed to create agent app: %v", err)
	}

	// 启动Agent应用
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start agent app: %v", err)
	}

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down NeoAgent server...")

	// 给服务器5秒钟的时间来完成现有请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 停止Agent应用
	if err := app.Stop(ctx); err != nil {
		log.Fatal("Agent forced to shutdown:", err)
	}

	log.Println("NeoAgent exiting")
}

// parseFlags 解析命令行参数
// 遵循Unix哲学：做一件事并做好
func parseFlags() *AgentOptions {
	opts := &AgentOptions{}

	flag.StringVar(&opts.ConfigPath, "config", "", "配置文件路径 (默认: configs/config.yaml)")
	flag.StringVar(&opts.LogLevel, "log-level", "", "日志级别 (debug, info, warn, error)")
	flag.BoolVar(&opts.ShowHelp, "help", false, "显示帮助信息")
	flag.BoolVar(&opts.ShowVersion, "version", false, "显示版本信息")
	flag.BoolVar(&opts.Daemon, "daemon", false, "后台运行模式")

	// 简化版本的help参数
	flag.BoolVar(&opts.ShowHelp, "h", false, "显示帮助信息 (简写)")
	flag.BoolVar(&opts.ShowVersion, "v", false, "显示版本信息 (简写)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "NeoScan Agent - 分布式安全扫描代理\n\n")
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s                           # 使用默认配置启动\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -config=/path/config.yaml # 指定配置文件\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -log-level=debug          # 设置日志级别\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -daemon                   # 后台运行\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version                  # 显示版本信息\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n更多信息请访问: https://github.com/sun977/NeoScan\n")
	}

	flag.Parse()
	return opts
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Printf("NeoScan Agent\n")
	fmt.Printf("Version: %s\n", getVersion())
	fmt.Printf("Build Time: %s\n", getBuildTime())
	fmt.Printf("Git Commit: %s\n", getGitCommit())
	fmt.Printf("Go Version: %s\n", getGoVersion())
	fmt.Printf("Platform: %s\n", getPlatform())
}

// getVersion 获取版本号
func getVersion() string {
	// TODO: 从构建时注入版本信息
	return "1.0.0"
}

// getBuildTime 获取构建时间
func getBuildTime() string {
	// TODO: 从构建时注入构建时间
	return "2025-01-14"
}

// getGitCommit 获取Git提交哈希
func getGitCommit() string {
	// TODO: 从构建时注入Git提交信息
	return "unknown"
}

// getGoVersion 获取Go版本
func getGoVersion() string {
	// TODO: 从构建时注入Go版本信息
	return "go1.21+"
}

// getPlatform 获取平台信息
func getPlatform() string {
	// TODO: 从构建时注入平台信息
	return "linux/amd64"
}
