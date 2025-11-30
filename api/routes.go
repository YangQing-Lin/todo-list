package api

import (
	"log"
	"net/http"
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

	withMiddlewares := func(f http.HandlerFunc) http.HandlerFunc {
		return chain(f, corsMiddleware, recoverMiddleware)
	}

	optionsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	registerTodoRoutes := func(base string) {
		mux.HandleFunc("GET "+base, withMiddlewares(h.ListTodos))
		mux.HandleFunc("POST "+base, withMiddlewares(h.CreateTodo))
		mux.HandleFunc("OPTIONS "+base, withMiddlewares(optionsHandler))

		mux.HandleFunc("GET "+base+"/stats", withMiddlewares(h.GetStats))

		// 批量操作的路由
		mux.HandleFunc("POST "+base+"/batch/complete", withMiddlewares(h.BatchCompleteTodos))
		mux.HandleFunc("POST "+base+"/batch/delete", withMiddlewares(h.BatchDeleteTodos))

		mux.HandleFunc("PUT "+base+"/{id}", withMiddlewares(h.UpdateTodo))
		mux.HandleFunc("DELETE "+base+"/{id}", withMiddlewares(h.DeleteTodo))
		mux.HandleFunc("OPTIONS "+base+"/{id}", withMiddlewares(optionsHandler))
	}

	// Versioned routes with legacy aliases for backward compatibility
	registerTodoRoutes("/api/v1/todos")
	registerTodoRoutes("/api/todos")

	mux.HandleFunc("/health", h.HealthCheck)

	return mux
}
