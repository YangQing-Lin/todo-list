package model

import (
	"time"
)

// Todo 表示一个待办事项
type Todo struct {
	ID          int        `json:"id"`
	Version     int        `json:"version"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`   // pending, completed
	// Deprecated: Priority is kept only for DB compatibility and legacy API responses.
	Priority    int        `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// NewTodo 创建一个新的待办事项
func NewTodo(title, description string) *Todo {
	now := time.Now()
	return &Todo{
		Version:     1,
		Title:       title,
		Description: description,
		Status:      "pending",
		Priority:    1, // 默认低优先级
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Complete 标记待办事项为完成
func (t *Todo) Complete() {
	now := time.Now()
	t.Status = "completed"
	t.UpdatedAt = now
	t.CompletedAt = &now
}

// Reactivate 重新激活待办事项
func (t *Todo) Reactivate() {
	t.Status = "pending"
	t.UpdatedAt = time.Now()
	t.CompletedAt = nil
}

// SetDueDate 设置截止日期
func (t *Todo) SetDueDate(dueDate time.Time) {
	t.DueDate = &dueDate
	t.UpdatedAt = time.Now()
}
