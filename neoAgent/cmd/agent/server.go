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
	"github.com/spf13/viper"
)

var (
	masterAddr string
	authToken  string
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动 Agent 服务模式 (Cluster Worker)",
	Long: `以守护进程方式启动 Agent，连接 Master 节点并监听任务下发。

可以通过命令行参数指定 Master 地址和认证 Token，也可以通过配置文件指定。
命令行参数优先级高于配置文件。

示例:
  neoAgent server --master 10.0.0.1:8080 --token mysecrettoken`,
	Run: func(cmd *cobra.Command, args []string) {
		// 绑定 Flags 到 Viper，这样 App 内部可以直接读取 Viper
		if masterAddr != "" {
			viper.Set("master.address", masterAddr)
		}
		if authToken != "" {
			viper.Set("auth.token", authToken)
		}
		runServer()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// 定义 Flags
	serverCmd.Flags().StringVar(&masterAddr, "master", "", "Master 节点地址 (e.g. 127.0.0.1:8080)")
	serverCmd.Flags().StringVar(&authToken, "token", "", "集群认证 Token")
}

// runServer 封装了原 main.go 的逻辑
func runServer() {
	// 创建Agent应用实例
	app, err := agent.NewApp()
	if err != nil {
		log.Fatalf("Failed to create agent app: %v", err)
	}

	// 启动Agent应用
	if err2 := app.Start(); err2 != nil {
		log.Fatalf("Failed to start agent app: %v", err2)
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
	if err1 := app.Stop(ctx); err1 != nil {
		log.Fatal("Agent forced to shutdown:", err1)
	}

	// 强制退出以防万一
	if err == context.DeadlineExceeded {
		log.Println("Timeout exceeded, forcing shutdown")
	}

	log.Println("NeoAgent exiting")
}
