package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
		Addr:    ":7789",
		Handler: mux,
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

	log.Println("Shutting down server...")
	if err := server.Close(); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
}
