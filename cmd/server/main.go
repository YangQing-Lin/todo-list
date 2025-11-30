// @title Todo List API
// @version 1.0
// @description Todo List RESTful API
// @host localhost:7789
// @BasePath /api/v1
// @schemes http
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	"todo-list/api"
	"todo-list/database"
	_ "todo-list/docs"
	"todo-list/handler"
)

func main() {
	// 支持环境变量配置数据库路径
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./todos.db"
	}

	// 初始化数据库
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 创建处理器
	h := handler.NewHandler(db)

	// 设置路由
	mux := api.SetupRoutes(h)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// 配置 HTTP 服务器
	server := &http.Server{
		Addr:           ":7789",
		Handler:        mux,
		ReadTimeout:    15 * time.Second, // 读请求超时
		WriteTimeout:   15 * time.Second, // 写响应超时
		IdleTimeout:    60 * time.Second, // Keep-Alive 空闲超时
		MaxHeaderBytes: 1 << 20,          // 1MB 头部限制
	}

	// 优雅关闭 - 在 goroutine 中启动服务器
	go func() {
		log.Println("Server started on http://localhost:7789")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 等待中断信号
	/*
		信号类型详解:
		| 信号              | 触发方式     | 数值 | 说明             |
		|-----------------|---------------|-----|------------------|
		| syscall.SIGINT  | Ctrl+C        | 2   | INTerrupt (中断) |
		| syscall.SIGTERM | kill <pid>    | 15  | TERMinate (终止) |
		| syscall.SIGKILL | kill -9 <pid> | 9   | 无法捕获,强制杀死 |
	*/
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit // 保存信号,用于日志

	// 记录收到的信号类型
	log.Printf("收到信号 %v，开始优雅关闭...", sig)

	// 等待最多30秒让现有请求完成
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 区分优雅关闭成功/失败，添加强制关闭逻辑
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭超时：%v，强制关闭", err)
		// 强制关闭(立即中断所有连接)
		if err := server.Close(); err != nil {
			log.Printf("强制关闭失败：%v", err)
		}
	} else {
		log.Println("HTTP 服务器已优雅关闭")
	}

	// 显式关闭数据库,记录详细日志
	if err := db.Close(); err != nil {
		log.Printf("数据库关闭失败：%v", err)
	} else {
		log.Println("数据库连接已关闭")
	}

	log.Println("服务器已完全停止")
}
