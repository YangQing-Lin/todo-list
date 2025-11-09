package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"todo-list/api"
	"todo-list/database"
	"todo-list/handler"
)

func main() {
	// 初始化数据库
	db, err := database.New("./todos.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 创建处理器
	h := handler.NewHandler(db)

	// 设置路由
	mux := api.SetupRoutes(h)

	// 启动服务器
	server := &http.Server{
		Addr:           ":7789",
		Handler:        mux,
		ReadTimeout:    15 * time.Second, // 读请求超时
		WriteTimeout:   15 * time.Second, // 写响应超时
		IdleTimeout:    60 * time.Second, // Keep-Alive 空闲超时
		MaxHeaderBytes: 1 << 20,          // 1MB 头部限制
	}

	// 优雅关闭
	go func() {
		log.Println("Server started on http://localhost:7789")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 等待最多30秒让现有请求完成
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Shutting down server...")
	// Shutdown() 等待现有请求完成再关闭
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
