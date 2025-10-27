package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"todo-list/model"
)

// Response 统一响应格式
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Handler 处理器结构体
type Handler struct {
	// 后续会添加数据库连接
}

// NewHandler 创建新的处理器
func NewHandler() *Handler {
	return &Handler{}
}

// sendJSON 发送JSON响应
func (h *Handler) sendJSON(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// ListTodos 获取待办事项列表
func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	// 临时返回空列表，后续会从数据库读取
	todos := []model.Todo{}

	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"todos": todos,
			"total": len(todos),
		},
		Message: "获取待办事项成功",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// CreateTodo 创建新的待办事项
func (h *Handler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	// 解析请求体
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON format")
		return
	}

	// 验证数据
	if req.Title == "" {
		h.sendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "标题不能为空")
		return
	}

	// 创建Todo
	todo := model.NewTodo(req.Title, req.Description)

	// 临时模拟创建成功，后续会保存到数据库
	todo.ID = 1

	response := Response{
		Success: true,
		Data:    todo,
		Message: "创建待办事项成功",
	}

	h.sendJSON(w, http.StatusCreated, response)
}

// sendError 发送错误响应
func (h *Handler) sendError(w http.ResponseWriter, status int, code, message string) {
	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
	h.sendJSON(w, status, response)
}

// HealthCheck 健康检查
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"status":    "ok",
			"timestamp": "server-time",
		},
		Message: "服务运行正常",
	}
	h.sendJSON(w, http.StatusOK, response)
}
