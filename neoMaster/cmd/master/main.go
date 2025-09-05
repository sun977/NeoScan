/*
 * @author: sun977
 * @date: 2025.09.05
 * @description: 主程序入口
 * @func: 初始化应用、配置路由、启动服务器、等待中断信号
 */

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"neomaster/internal/app/master"
)

func main() {
	// 创建应用实例
	app, err := master.NewApp()
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// 获取配置和Gin引擎
	config := app.GetConfig()
	engine := app.GetRouter().GetEngine()

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        engine,
		ReadTimeout:    config.Server.ReadTimeout,
		WriteTimeout:   config.Server.WriteTimeout,
		IdleTimeout:    config.Server.IdleTimeout,
		MaxHeaderBytes: config.Server.MaxHeaderBytes,
	}

	// 启动服务器的goroutine
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 给服务器5秒钟的时间来完成现有请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server exiting")
}
