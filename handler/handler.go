package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"todo-list/database"
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
	db *database.DB
}

// NewHandler 创建新的处理器
func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// sendJSON 发送JSON响应
func (h *Handler) sendJSON(w http.ResponseWriter, status int, response Response) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		h.sendError(w, http.StatusInternalServerError, "ENCODE_ERROR", "内部错误")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
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

// ListTodos 获取待办事项列表
func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	todos, err := h.db.ListTodos()
	if err != nil {
		log.Printf("Failed to list todos: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "获取待办事项失败")
		return
	}

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
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 限制1MB

	// 解析请求体
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", fmt.Sprintf("JSON解析失败: %v", err))
		return
	}

	// 验证数据
	if req.Title == "" {
		h.sendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "标题不能为空")
		return
	}

	// 创建Todo
	todo := model.NewTodo(req.Title, req.Description)

	if err := h.db.CreateTodo(todo); err != nil {
		log.Printf("Failed to create todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", fmt.Sprintf("创建待办事项失败: %v", err))
		return
	}

	response := Response{
		Success: true,
		Data:    todo,
		Message: "创建待办事项成功",
	}

	h.sendJSON(w, http.StatusCreated, response)
}

// UpdateTodo 更新待办事项
func (h *Handler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodPut {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", "无效的ID")
		return
	}

	id, err := strconv.Atoi(parts[3])
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", fmt.Sprintf("无效的ID格式: %v", err))
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Status      string  `json:"status"`
		Priority    int     `json:"priority"`
		DueDate     *string `json:"due_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", fmt.Sprintf("Invalid JSON format: %v", err))
		return
	}

	todo, err := h.db.GetTodoByID(id)
	if err != nil {
		log.Printf("failed to get todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "获取待办事项失败")
		return
	}
	if todo == nil {
		h.sendError(w, http.StatusNotFound, "NOT_FOUND", "待办事项不存在")
		return
	}

	// 更新字段
	if req.Title != "" {
		todo.Title = req.Title
	}
	todo.Description = req.Description
	if req.Status != "" {
		todo.Status = req.Status
	}
	if req.Priority > 0 {
		todo.SetPriority(req.Priority)
	}
	if req.DueDate != nil {
		todo.SetDueDate(*req.DueDate)
	}

	if err := h.db.UpdateTodo(todo); err != nil {
		log.Printf("Failed to update todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "更新待办事项失败")
		return
	}

	response := Response{
		Success: true,
		Data:    todo,
		Message: "更新待办事项成功",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// DeleteTodo 删除待办事项
func (h *Handler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != http.MethodDelete {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", "无效的ID")
		return
	}

	id, err := strconv.Atoi(parts[3])
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", fmt.Sprintf("无效的Id格式: %v", err))
		return
	}

	if err := h.db.DeleteTodo(id); err != nil {
		log.Printf("Failed to delete todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "删除待办事项失败")
		return
	}

	response := Response{
		Success: true,
		Message: "删除待办事项成功",
	}

	h.sendJSON(w, http.StatusOK, response)
}
