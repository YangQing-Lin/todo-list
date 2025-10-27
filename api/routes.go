package api

import (
	"net/http"
	"todo-list/handler"
)

// Router 路由器结构体
type Router struct {
	handler *handler.Handler
}

// NewRouter 创建新的路由器
func NewRouter() *Router {
	return &Router{
		handler: handler.NewHandler(),
	}
}

// SetupRoutes 设置所有路由
func (r *Router) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// API路由
	mux.HandleFunc("/", r.handler.HealthCheck)
	mux.HandleFunc("/health", r.handler.HealthCheck)

	// Todo相关路由
	mux.HandleFunc("/api/todos", r.handleTodos)

	return mux
}

// handleTodos 处理todos相关的请求
func (r *Router) handleTodos(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.handler.ListTodos(w, req)
	case http.MethodPost:
		r.handler.CreateTodo(w, req)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}