package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
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

// CreateTodoRequest 创建待办事项请求体
type CreateTodoRequest struct {
	Title       string `json:"title" example:"Buy groceries"`
	Description string `json:"description" example:"Milk, bread, and fruits"`
}

// UpdateTodoRequest 更新待办事项请求体
type UpdateTodoRequest struct {
	Version     *int       `json:"version,omitempty" example:"2"`
	Title       *string    `json:"title,omitempty" example:"Update weekly report"`
	Description *string    `json:"description,omitempty" example:"Finish and send by EOD"`
	Status      *string    `json:"status,omitempty" example:"DONE"`
	DueDate     *time.Time `json:"due_date,omitempty" example:"2024-05-30T16:00:00Z"`
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

// 超时配置
const (
	DefaultTimeout = 10 * time.Second // 默认超时
	ListTimeout    = 5 * time.Second  // 列表查询超时
	CreateTimeout  = 3 * time.Second  // 创建超时
	UpdateTimeout  = 3 * time.Second  // 更新超时
	DeleteTimeout  = 2 * time.Second  // 删除超时
	StatsTimeout   = 5 * time.Second  // 统计查询超时
)

// NewHandler 创建新的处理器
func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// sendJSON 发送JSON响应
func (h *Handler) sendJSON(w http.ResponseWriter, status int, response Response) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(response); err != nil {
		// JSON编码失败，直接返回纯文本错误，不要再尝试调用sendError（会递归）
		log.Printf("Failed to encode response: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error: Failed to encode response"))
		return
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
// @Summary 健康检查
// @Description 返回应用当前健康状态
// @Tags health
// @Produce json
// @Success 200 {object} handler.Response
// @Router /health [get]
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

// ListTodos 获取待办事项列表(带超时控制)
// @Summary 获取待办事项列表
// @Description 支持筛选、搜索、排序和分页的待办事项列表
// @Tags todos
// @Param status query string false "状态过滤"
// @Param search query string false "搜索关键字"
// @Param sort query string false "排序字段"
// @Param order query string false "排序方式" Enums(asc,desc)
// @Param limit query int false "返回条数" default(50)
// @Param offset query int false "偏移量" default(0)
// @Produce json
// @Success 200 {object} handler.Response
// @Failure 500 {object} handler.Response
// @Router /todos [get]
func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
	// 创建带超时的 Context
	ctx, cancel := context.WithTimeout(r.Context(), ListTimeout)
	defer cancel()

	// 解析查询参数
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if l, err := strconv.Atoi(l); err == nil && l > 0 {
			limit = l
			// 限制最大值，防止恶意请求
			if limit > 200 {
				limit = 200
			}
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if o, err := strconv.Atoi(o); err == nil && o >= 0 {
			offset = o
		}
	}

	// 构建过滤器
	filter := database.TodoFilter{
		Status: status,
		Search: search,
		Sort:   sort,
		Order:  order,
		Limit:  limit,
		Offset: offset,
	}

	// 调用带 Context 的数据库方法
	todos, total, err := h.db.ListTodosContext(ctx, filter)
	if err != nil {
		// 区分超时错误和其他错误
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("ListTodos timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "查询超时，请稍后重试")
			return
		}
		if errors.Is(err, context.Canceled) {
			log.Printf("ListTodos canceled: %v", err)
			// 客户端取消请求,不需要响应
			return
		}
		log.Printf("Failed to list todos: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "查询失败")
		return
	}

	// 返回结果（包含分页信息）
	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"todos":  todos,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
		Message: "获取待办事项成功",
	}
	h.sendJSON(w, http.StatusOK, response)
}

// CreateTodo 创建待办事项(带超时控制)
// @Summary 创建待办事项
// @Description 创建一个新的待办事项
// @Tags todos
// @Accept json
// @Produce json
// @Param todo body handler.CreateTodoRequest true "待办事项内容"
// @Success 201 {object} handler.Response
// @Failure 400 {object} handler.Response
// @Failure 500 {object} handler.Response
// @Router /todos [post]
func (h *Handler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), CreateTimeout)
	defer cancel()

	defer r.Body.Close()

	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 限制1MB

	// 解析请求体
	var req CreateTodoRequest

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

	if err := h.db.CreateTodoContext(ctx, todo); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("CreateTodo timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "创建超时，请稍后重试")
			return
		}
		if errors.Is(err, context.Canceled) {
			log.Printf("ListTodos canceled: %v", err)
			// 客户端取消请求,不需要响应
			return
		}
		log.Printf("Failed to create todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "创建失败")
		return
	}

	response := Response{
		Success: true,
		Data:    todo,
		Message: "创建待办事项成功",
	}

	h.sendJSON(w, http.StatusCreated, response)
}

// UpdateTodo 更新待办事项(带超时控制)
// @Summary 更新待办事项
// @Description 根据 ID 更新待办事项信息
// @Tags todos
// @Accept json
// @Produce json
// @Param id path int true "待办事项ID"
// @Param todo body handler.UpdateTodoRequest true "待办事项更新内容"
// @Success 200 {object} handler.Response
// @Failure 400 {object} handler.Response
// @Failure 404 {object} handler.Response
// @Failure 409 {object} handler.Response
// @Failure 500 {object} handler.Response
// @Router /todos/{id} [put]
func (h *Handler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), UpdateTimeout)
	defer cancel()

	defer r.Body.Close()

	if r.Method != http.MethodPut {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", "无效的ID")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", fmt.Sprintf("无效的ID格式: %v", err))
		return
	}

	if id <= 0 {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", "无效的ID")
		return
	}

	var req UpdateTodoRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", fmt.Sprintf("Invalid JSON format: %v", err))
		return
	}

	if req.Version != nil && *req.Version < 1 {
		h.sendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "版本号无效")
		return
	}

	existingTodo, err := h.db.GetTodoByID(id)
	if err != nil {
		log.Printf("failed to get todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "获取待办事项失败")
		return
	}
	if existingTodo == nil {
		h.sendError(w, http.StatusNotFound, "NOT_FOUND", "待办事项不存在")
		return
	}

	// 更新字段
	if req.Title != nil {
		existingTodo.Title = *req.Title
	}
	if req.Description != nil {
		existingTodo.Description = *req.Description
	}
	if req.Status != nil {
		existingTodo.Status = *req.Status
		switch *req.Status {
		case "completed":
			now := time.Now()
			existingTodo.CompletedAt = &now
		case "pending":
			existingTodo.CompletedAt = nil
		}
	}
	if req.DueDate != nil {
		existingTodo.SetDueDate(*req.DueDate)
	}

	// 处理乐观锁
	if req.Version != nil {
		existingTodo.Version = *req.Version
	}

	if err := h.db.UpdateTodoContext(ctx, existingTodo); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("UpdateTodo timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "更新超时，请稍后重试")
			return
		}
		if errors.Is(err, database.ErrVersionConflict) {
			h.sendError(w, http.StatusConflict, "VERSION_CONFLICT", "版本冲突，请刷新后重试")
			return
		}
		if errors.Is(err, context.Canceled) {
			log.Printf("ListTodos canceled: %v", err)
			// 客户端取消请求,不需要响应
			return
		}
		log.Printf("Failed to update todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "更新失败")
		return
	}

	response := Response{
		Success: true,
		Data:    existingTodo,
		Message: "更新待办事项成功",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// DeleteTodo 删除待办事项(带超时控制)
// @Summary 删除待办事项
// @Description 根据 ID 删除待办事项
// @Tags todos
// @Produce json
// @Param id path int true "待办事项ID"
// @Success 200 {object} handler.Response
// @Failure 400 {object} handler.Response
// @Failure 500 {object} handler.Response
// @Router /todos/{id} [delete]
func (h *Handler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DeleteTimeout)
	defer cancel()

	defer r.Body.Close()

	if r.Method != http.MethodDelete {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", fmt.Sprintf("无效的Id格式: %v", err))
		return
	}

	if id <= 0 {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", "无效的ID")
		return
	}

	if err := h.db.DeleteTodoContext(ctx, id); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("DeleteTodo timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "删除超时，请稍后重试")
			return
		}
		if errors.Is(err, context.Canceled) {
			log.Printf("ListTodos canceled: %v", err)
			// 客户端取消请求,不需要响应
			return
		}
		log.Printf("Failed to delete todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "删除失败")
		return
	}

	response := Response{
		Success: true,
		Message: "删除待办事项成功",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetStats 获取统计信息(带超时控制)
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), StatsTimeout)
	defer cancel()

	stats, err := h.db.GetStatsContext(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("GetStats timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "统计查询超时，请稍后重试")
			return
		}
		if errors.Is(err, context.Canceled) {
			log.Printf("ListTodos canceled: %v", err)
			// 客户端取消请求,不需要响应
			return
		}
		log.Printf("Failed to get stats: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "获取统计信息失败")
		return
	}

	response := Response{
		Success: true,
		Data:    stats,
		Message: "获取统计信息成功",
	}

	h.sendJSON(w, http.StatusOK, response)
}
