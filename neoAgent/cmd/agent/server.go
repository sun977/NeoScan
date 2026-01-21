/*
 * @author: Sun977
 * @date: 2026.01.21
 * @description: Server 模式子命令 (Worker Mode)
 */

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"neoagent/internal/app/agent"

	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动 Agent 服务模式 (Cluster Worker)",
	Long:  `以守护进程方式启动 Agent，连接 Master 节点并监听任务下发。`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

// runServer 封装了原 main.go 的逻辑
func runServer() {
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

	// 强制退出以防万一
	if err == context.DeadlineExceeded {
		log.Println("Timeout exceeded, forcing shutdown")
	}

	log.Println("NeoAgent exiting")
}
