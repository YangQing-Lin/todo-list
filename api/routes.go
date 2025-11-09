package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"todo-list/handler"
)

// corsMiddleware 处理 CORS 跨域请求
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// 处理预检请求
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// recoverMiddleware 捕获 panic 防止服务崩溃
func recoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// chain 链接多个中间件
func chain(f http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		f = middlewares[i](f)
	}
	return f
}

func SetupRoutes(h *handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// API路由 - 使用中间件链
	mux.HandleFunc("/api/todos", chain(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListTodos(w, r)
		case http.MethodPost:
			h.CreateTodo(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, corsMiddleware, recoverMiddleware))

	// 单个Todo操作: /api/todos/{id}
	mux.HandleFunc("/api/todos/", chain(func(w http.ResponseWriter, r *http.Request) {
		// 提取并验证ID
		idStr := strings.TrimPrefix(r.URL.Path, "/api/todos/")

		// 检查ID是否为空
		if idStr == "" {
			http.Error(w, "ID required", http.StatusBadRequest)
			return
		}

		// 验证ID格式 - 必须是正整数
		if id, err := strconv.ParseInt(idStr, 10, 64); err != nil || id <= 0 {
			http.Error(w, "Invalid ID format", http.StatusBadRequest)
			return
		}

		// 检查是否有多余的路径（防止 /api/todos/123/xxx）
		if strings.Contains(idStr, "/") {
			http.Error(w, "Invalid ID format", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPut:
			h.UpdateTodo(w, r)
		case http.MethodDelete:
			h.DeleteTodo(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, corsMiddleware, recoverMiddleware))

	mux.HandleFunc("/health", h.HealthCheck)

	return mux
}
