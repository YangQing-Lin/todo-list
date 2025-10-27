package model

import (
	"time"
)

// Todo 表示一个待办事项
type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // pending, completed
	Priority    int       `json:"priority"` // 1=低, 2=中, 3=高
	DueDate     *string   `json:"due_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt *string   `json:"completed_at,omitempty"`
}

// NewTodo 创建一个新的待办事项
func NewTodo(title, description string) *Todo {
	now := time.Now()
	return &Todo{
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
	t.Status = "completed"
	t.UpdatedAt = time.Now()
	completedAt := t.UpdatedAt.Format(time.RFC3339)
	t.CompletedAt = &completedAt
}

// Reactivate 重新激活待办事项
func (t *Todo) Reactivate() {
	t.Status = "pending"
	t.UpdatedAt = time.Now()
	t.CompletedAt = nil
}

// SetPriority 设置优先级
func (t *Todo) SetPriority(priority int) {
	if priority >= 1 && priority <= 3 {
		t.Priority = priority
		t.UpdatedAt = time.Now()
	}
}

// SetDueDate 设置截止日期
func (t *Todo) SetDueDate(dueDate string) {
	t.DueDate = &dueDate
	t.UpdatedAt = time.Now()
}

// IsOverdue 检查是否过期
func (t *Todo) IsOverdue() bool {
	if t.DueDate == nil || t.Status == "completed" {
		return false
	}
	dueTime, err := time.Parse(time.RFC3339, *t.DueDate)
	if err != nil {
		return false
	}
	return time.Now().After(dueTime)
}