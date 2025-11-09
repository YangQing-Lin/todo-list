package api

import (
	"net/http"
	"strings"
	"todo-list/handler"
)

func SetupRoutes(h *handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// API路由
	mux.HandleFunc("/api/todos", func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.ListTodos(w, r)
		case http.MethodPost:
			h.CreateTodo(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 单个Todo操作: /api/todos/{id}
	mux.HandleFunc("/api/todos/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 检查是否有ID
		if strings.TrimPrefix(r.URL.Path, "/api/todos/") == "" {
			http.Error(w, "ID required", http.StatusBadRequest)
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
	})

	mux.HandleFunc("/health", h.HealthCheck)

	return mux
}
