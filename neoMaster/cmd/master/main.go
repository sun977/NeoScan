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

	// 获取Gin引擎
	engine := app.GetRouter().GetEngine()

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":8080",
		Handler: engine,
	}

	// 启动服务器的goroutine
	go func() {
		log.Println("Starting server on :8080")
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
