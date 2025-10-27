package main

import (
	"log"
	"net/http"
	"todo-list/api"
)

func main() {
	// 创建路由器并设置路由
	router := api.NewRouter()
	handler := router.SetupRoutes()

	// 配置服务器
	port := ":7789"
	log.Printf("Go Todo List API Server")
	log.Printf("========================")
	log.Printf("Server starting on port %s", port)
	log.Printf("\nAvailable endpoints:")
	log.Printf("  GET  /             - Health check")
	log.Printf("  GET  /health       - Health check")
	log.Printf("  GET  /api/todos    - List todos")
	log.Printf("  POST /api/todos    - Create todo")
	log.Printf("\nTry these commands:")
	log.Printf("  curl http://localhost%s/", port)
	log.Printf("  curl http://localhost%s/api/todos", port)

	// 启动服务器
	server := &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  10 * 1000000000, // 10 seconds
		WriteTimeout: 10 * 1000000000, // 10 seconds
	}

	log.Printf("\nServer started successfully!")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
