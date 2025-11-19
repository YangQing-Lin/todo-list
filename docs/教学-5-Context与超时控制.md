# 教学 Part 5: Context 与超时控制

> **📌 学习阶段**: 中级 / Go 进阶特性
> **前置要求**: 已掌握基础 CRUD、HTTP 处理
> **学习目标**: 理解 Context、实现超时控制、防止资源泄漏
> **时间投入**: 2-3 小时(理解原理 + 实现 + 测试)

---

## 🎯 功能分析 - Linus 式思考

### Linus 的三个问题

**1. "这是个真问题还是臆想出来的?"**

✅ **真实问题**:
- 数据库查询卡住 → 整个 HTTP 请求永远不返回 → 客户端超时重试 → 雪崩
- 用户关闭浏览器 → 服务器还在查询数据库 → 浪费资源
- 慢查询(搜索 1000 条记录) → 占用数据库连接 → 其他请求被阻塞

**为什么需要超时控制?**
- **防止资源泄漏**: 取消无用的操作,释放连接
- **提高响应速度**: 快速失败,不让用户干等
- **级联取消**: 请求取消时,停止所有子操作(数据库、API调用)

**2. "有更简单的方法吗?"**

💡 **核心思路**: Context 上下文传播

**错误方法**(全局超时):
```go
// ❌ 所有请求统一5秒超时,太死板
http.TimeoutHandler(handler, 5*time.Second, "Timeout")
```

**正确方法**(Context传递):
```go
// ✅ 请求级别的超时,可以灵活控制
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()
result := db.QueryContext(ctx, "SELECT ...")
```

**3. "会破坏什么吗?"**

⚠️ **潜在风险**:
- 超时时间设置太短 → 正常查询也会失败
- 忘记 `defer cancel()` → 内存泄漏(goroutine 不会退出)
- 数据库驱动不支持 Context → 超时控制无效

✅ **安全设计**:
- 合理的超时时间(根据 P95 延迟设置)
- 统一的超时配置(可调整)
- 错误日志记录(区分"真正失败"和"超时取消")

---

## 📚 核心知识讲解

### 1. Context 是什么?

**官方定义**: Context 在 goroutine 之间传递截止时间、取消信号和请求范围的值。

**类比**: Context 就像一个"任务控制器":
```
你(主goroutine): "去超市买牛奶,如果10分钟没回来就别买了"
孩子(子goroutine): "好的" (带着Context出发)

9分钟后...
孩子: "到了超市,正在排队" (检查Context.Done())
你: "取消任务,我改主意了" (调用cancel())
孩子: "收到,马上回来" (停止操作)
```

---

### 2. Context 的四种创建方式

**方式 1: context.Background()** - 根 Context
```go
// 程序启动时,main 函数中使用
ctx := context.Background()
```

**方式 2: context.TODO()** - 占位符
```go
// 暂时不知道用什么 Context 时使用
ctx := context.TODO()
```

**方式 3: context.WithTimeout()** - 超时控制
```go
// 5 秒后自动取消
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()
```

**方式 4: context.WithCancel()** - 手动取消
```go
// 手动调用 cancel() 取消
ctx, cancel := context.WithCancel(parent)
defer cancel()
```

**关键点**:
- 所有 Context 都是从 `Background()` 派生的
- `cancel()` 必须调用(即使没有取消),用 `defer` 确保执行
- Context 是不可变的,每次派生会创建新 Context

---

### 3. Context 在 HTTP 中的应用

**HTTP 请求自带 Context**:
```go
func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
    // r.Context() 包含:
    // - 客户端断开连接时自动取消
    // - HTTP 服务器的超时设置
    ctx := r.Context()

    // 添加自定义超时
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // 传递给数据库查询
    todos, err := h.db.ListTodosContext(ctx, filter)
}
```

**Context 的生命周期**:
```
客户端请求 → HTTP服务器创建Context → Handler接收
    ↓
Handler派生新Context(加超时) → 传递给数据库
    ↓
数据库查询中... → 检查Context.Done()
    ↓
[选项A] 查询完成 → 返回结果 → Context结束
[选项B] 超时 → Context取消 → 数据库停止查询 → 返回超时错误
[选项C] 客户端断开 → Context取消 → 停止所有操作
```

---

### 4. 数据库操作的 Context 集成

**database/sql 包支持 Context**:
```go
// 不带 Context(老式写法)
rows, err := db.Query("SELECT * FROM todos")

// 带 Context(推荐写法)
rows, err := db.QueryContext(ctx, "SELECT * FROM todos")
```

**区别**:
- `Query`: 即使客户端断开,查询也会继续执行
- `QueryContext`: Context 取消时,数据库驱动会尝试中断查询

**SQLite 的限制**:
- SQLite 是嵌入式数据库,不支持真正的"查询中断"
- `QueryContext` 会在查询开始前检查 Context,但无法中断正在执行的查询
- 仍然推荐使用 `QueryContext`,防止启动无用的查询

---

## 💻 完整代码示例

### 文件 1: `database/db.go` - 添加 Context 支持

```go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"todo-list/model"
)

// ListTodosContext 获取待办事项列表(支持 Context)
func (db *DB) ListTodosContext(ctx context.Context, filter TodoFilter) ([]model.Todo, int, error) {
	// 设置默认值
	if filter.Sort == "" {
		filter.Sort = "created_at"
	}
	if filter.Order == "" {
		filter.Order = "DESC"
	} else {
		filter.Order = strings.ToUpper(filter.Order)
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Status == "" {
		filter.Status = "all"
	}

	baseQuery := "SELECT id, version, title, description, status, due_date, created_at, updated_at, completed_at FROM todos WHERE 1=1"
	args := []interface{}{}

	// 动态添加查询条件
	if filter.Status != "" && filter.Status != "all" {
		baseQuery += " AND status = ?"
		args = append(args, filter.Status)
	}

	if filter.Search != "" {
		baseQuery += " AND (title LIKE ? OR description LIKE ?)"
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// 查询总数(带 Context)
	countQuery := "SELECT COUNT(*) FROM todos WHERE 1=1"
	countArgs := []interface{}{}

	if filter.Status != "" && filter.Status != "all" {
		countQuery += " AND status = ?"
		countArgs = append(countArgs, filter.Status)
	}
	if filter.Search != "" {
		countQuery += " AND (title LIKE ? OR description LIKE ?)"
		searchPattern := "%" + filter.Search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern)
	}

	var total int
	// 使用 QueryRowContext 而不是 QueryRow
	err := db.conn.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 添加排序和分页
	allowedSortFields := map[string]bool{
		"created_at": true,
		"due_date":   true,
		"status":     true,
	}
	allowedOrders := map[string]bool{
		"ASC":  true,
		"DESC": true,
	}

	if !allowedSortFields[filter.Sort] {
		filter.Sort = "created_at"
	}
	if !allowedOrders[filter.Order] {
		filter.Order = "DESC"
	}

	baseQuery += fmt.Sprintf(" ORDER BY %s %s LIMIT ? OFFSET ?", filter.Sort, filter.Order)
	args = append(args, filter.Limit, filter.Offset)

	// 执行查询(带 Context)
	rows, err := db.conn.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var todos []model.Todo
	for rows.Next() {
		// 检查 Context 是否已取消(可选,SQLite 可能不会自动检查)
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}

		var todo model.Todo
		var dueDate, completedAt sql.NullString

		err := rows.Scan(
			&todo.ID,
			&todo.Version,
			&todo.Title,
			&todo.Description,
			&todo.Status,
			&dueDate,
			&todo.CreatedAt,
			&todo.UpdatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描失败: %w", err)
		}

		if dueDate.Valid {
			t, _ := time.Parse(time.RFC3339, dueDate.String)
			todo.DueDate = &t
		}

		if completedAt.Valid {
			t, _ := time.Parse(time.RFC3339, completedAt.String)
			todo.CompletedAt = &t
		}

		todos = append(todos, todo)
	}

	// 检查迭代过程中的错误
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return todos, total, nil
}

// CreateTodoContext 创建待办事项(支持 Context)
func (db *DB) CreateTodoContext(ctx context.Context, todo *model.Todo) error {
	query := `
		INSERT INTO todos (title, description, status, due_date, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.ExecContext(
		ctx,
		query,
		todo.Title,
		todo.Description,
		todo.Status,
		todo.DueDate,
		todo.CreatedAt,
		todo.UpdatedAt,
		todo.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to create todo: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	todo.ID = int(id)
	return nil
}

// UpdateTodoContext 更新待办事项(支持 Context)
func (db *DB) UpdateTodoContext(ctx context.Context, todo *model.Todo) error {
	query := `
		UPDATE todos
		SET title = ?, description = ?, status = ?,
		    due_date = ?, updated_at = ?, completed_at = ?, version = version + 1
		WHERE id = ? AND version = ?
	`

	todo.UpdatedAt = time.Now()

	result, err := db.conn.ExecContext(
		ctx,
		query,
		todo.Title,
		todo.Description,
		todo.Status,
		todo.DueDate,
		todo.UpdatedAt,
		todo.CompletedAt,
		todo.ID,
		todo.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrVersionConflict
	}

	todo.Version++

	return nil
}

// DeleteTodoContext 删除待办事项(支持 Context)
func (db *DB) DeleteTodoContext(ctx context.Context, id int) error {
	query := `DELETE FROM todos WHERE id = ?`

	result, err := db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("todo not found")
	}

	return nil
}

// GetStatsContext 获取统计信息(支持 Context)
func (db *DB) GetStatsContext(ctx context.Context) (*TodoStats, error) {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	weekLater := now.AddDate(0, 0, 7).Format("2006-01-02")

	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND due_date < ? THEN 1 ELSE 0 END) as overdue,
			SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND date(due_date) = ? THEN 1 ELSE 0 END) as today,
			SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND date(due_date) BETWEEN ? AND ? THEN 1 ELSE 0 END) as this_week
		FROM todos
	`

	var stats TodoStats
	var pending, completed, overdue, todayCount, thisWeek sql.NullInt64

	err := db.conn.QueryRowContext(ctx, query, now, today, today, weekLater).Scan(
		&stats.Total,
		&pending,
		&completed,
		&overdue,
		&todayCount,
		&thisWeek,
	)

	if err != nil {
		return nil, fmt.Errorf("查询统计信息失败: %w", err)
	}

	// 处理 NULL 值
	if pending.Valid {
		stats.Pending = int(pending.Int64)
	}
	if completed.Valid {
		stats.Completed = int(completed.Int64)
	}
	if overdue.Valid {
		stats.Overdue = int(overdue.Int64)
	}
	if todayCount.Valid {
		stats.Today = int(todayCount.Int64)
	}
	if thisWeek.Valid {
		stats.ThisWeek = int(thisWeek.Int64)
	}

	return &stats, nil
}
```

---

### 文件 2: `handler/handler.go` - Handler 使用 Context

```go
package handler

import (
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

// 超时配置
const (
	DefaultTimeout = 10 * time.Second // 默认超时
	ListTimeout    = 5 * time.Second  // 列表查询超时
	CreateTimeout  = 3 * time.Second  // 创建超时
	UpdateTimeout  = 3 * time.Second  // 更新超时
	DeleteTimeout  = 2 * time.Second  // 删除超时
	StatsTimeout   = 5 * time.Second  // 统计查询超时
)

// ListTodos 获取待办事项列表(带超时控制)
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
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

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
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "查询超时,请稍后重试")
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

	response := Response{
		Success: true,
		Data: map[string]interface{}{
			"todos":  todos,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
		Message: "查询成功",
	}
	h.sendJSON(w, http.StatusOK, response)
}

// CreateTodo 创建待办事项(带超时控制)
func (h *Handler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), CreateTimeout)
	defer cancel()

	defer r.Body.Close()

	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", fmt.Sprintf("JSON 解析失败: %v", err))
		return
	}

	if req.Title == "" {
		h.sendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "标题不能为空")
		return
	}

	todo := model.NewTodo(req.Title, req.Description)

	if err := h.db.CreateTodoContext(ctx, todo); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("CreateTodo timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "创建超时,请稍后重试")
			return
		}
		log.Printf("Failed to create todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "创建失败")
		return
	}

	response := Response{
		Success: true,
		Data:    todo,
		Message: "创建成功",
	}
	h.sendJSON(w, http.StatusCreated, response)
}

// UpdateTodo 更新待办事项(带超时控制)
func (h *Handler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), UpdateTimeout)
	defer cancel()

	defer r.Body.Close()

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", "无效的 ID")
		return
	}

	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", fmt.Sprintf("JSON 解析失败: %v", err))
		return
	}

	// 获取现有待办
	existingTodo, err := h.db.GetTodoByID(id)
	if err != nil {
		log.Printf("Failed to get todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "获取待办失败")
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
		if *req.Status == "completed" && existingTodo.CompletedAt == nil {
			now := time.Now()
			existingTodo.CompletedAt = &now
		} else if *req.Status == "pending" {
			existingTodo.CompletedAt = nil
		}
	}
	if req.DueDate != nil {
		existingTodo.DueDate = req.DueDate
	}

	// 处理乐观锁
	if req.Version != nil {
		existingTodo.Version = *req.Version
	}

	if err := h.db.UpdateTodoContext(ctx, existingTodo); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("UpdateTodo timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "更新超时,请稍后重试")
			return
		}
		if errors.Is(err, database.ErrVersionConflict) {
			h.sendError(w, http.StatusConflict, "VERSION_CONFLICT", "版本冲突,请刷新后重试")
			return
		}
		log.Printf("Failed to update todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "更新失败")
		return
	}

	response := Response{
		Success: true,
		Data:    existingTodo,
		Message: "更新成功",
	}
	h.sendJSON(w, http.StatusOK, response)
}

// DeleteTodo 删除待办事项(带超时控制)
func (h *Handler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), DeleteTimeout)
	defer cancel()

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_ID", "无效的 ID")
		return
	}

	if err := h.db.DeleteTodoContext(ctx, id); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("DeleteTodo timeout: %v", err)
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "删除超时,请稍后重试")
			return
		}
		log.Printf("Failed to delete todo: %v", err)
		h.sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", "删除失败")
		return
	}

	response := Response{
		Success: true,
		Message: "删除成功",
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
			h.sendError(w, http.StatusRequestTimeout, "TIMEOUT", "统计查询超时,请稍后重试")
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
```

---

## ⚠️ 关键点解析

### 1. 超时时间的选择

**如何确定超时时间?**
```go
// ❌ 太短 - 正常查询也会超时
ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)

// ✅ 合理 - 根据实际测试的 P95 延迟
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
```

**推荐策略**:
1. 测试正常请求的耗时(取 P95 或 P99)
2. 设置超时为 `P95 * 2`(留有余地)
3. 不同操作设置不同超时:
   - 查询列表: 5s(可能需要扫描多行)
   - 创建/更新: 3s(单行操作)
   - 删除: 2s(最快)
   - 统计查询: 5s(聚合计算)

---

### 2. defer cancel() 的重要性

**问题代码**:
```go
// ❌ 忘记 cancel,导致内存泄漏
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
todos, err := h.db.ListTodosContext(ctx, filter)
// cancel 没有被调用 → Context 不会被释放 → goroutine 泄漏
```

**正确代码**:
```go
// ✅ 立即 defer,确保 cancel 被调用
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()  // 即使提前 return,也会执行
todos, err := h.db.ListTodosContext(ctx, filter)
```

**为什么必须 cancel?**
- `WithTimeout` 内部启动了一个 timer goroutine
- 如果不调用 `cancel()`,timer 会一直运行到超时
- 即使操作已经完成,timer 也不会停止
- 多次调用 `cancel()` 是安全的(幂等)

---

### 3. 错误处理的区分

**Context 的三种错误**:
```go
err := h.db.ListTodosContext(ctx, filter)

// 错误类型 1: 超时
if errors.Is(err, context.DeadlineExceeded) {
    // 操作太慢,超过了设定的时间
    return 408 Request Timeout
}

// 错误类型 2: 取消
if errors.Is(err, context.Canceled) {
    // 客户端主动取消(关闭浏览器等)
    // 不需要响应,客户端已经不在了
    return
}

// 错误类型 3: 其他错误
// 数据库连接失败、SQL 语法错误等
return 500 Internal Server Error
```

---

### 4. SQLite 的 Context 限制

**SQLite 不支持真正的查询中断**:
```go
// PostgreSQL/MySQL:
// Context 取消 → 发送 "KILL QUERY" 到数据库 → 查询立即停止

// SQLite:
// Context 取消 → 数据库驱动返回错误 → 但查询仍在执行
```

**为什么仍然要用 Context?**
1. 防止启动新的查询(Context 在 `QueryContext` 开始前检查)
2. 代码可移植性(切换到 PostgreSQL 时不需要改代码)
3. 统一的超时处理逻辑

**优化建议**:
```go
// 在循环中手动检查 Context
for rows.Next() {
    select {
    case <-ctx.Done():
        return ctx.Err()  // 快速退出
    default:
    }

    // 处理行数据...
}
```

---

## 🧪 测试示例

### 测试超时控制

```bash
#!/bin/bash

# 测试 1: 正常请求(应该成功)
echo "=== 测试 1: 正常查询 ==="
time curl -s "http://localhost:7789/api/v1/todos?limit=10" | jq '.success'

# 测试 2: 模拟慢查询(在数据库中插入大量数据后测试)
echo -e "\n=== 测试 2: 大量数据查询 ==="
# 先插入 10000 条数据
for i in {1..10000}; do
  curl -s -X POST "http://localhost:7789/api/v1/todos" \
    -H "Content-Type: application/json" \
    -d "{\"title\": \"Task $i\"}" > /dev/null
done

# 查询全部(可能触发超时)
time curl -s "http://localhost:7789/api/v1/todos?limit=10000" | jq '.error.code'

# 测试 3: 客户端取消请求
echo -e "\n=== 测试 3: 客户端取消 ==="
# 启动请求,1 秒后 Ctrl+C 取消
timeout 1 curl "http://localhost:7789/api/v1/todos?limit=10000" || echo "已取消"

# 检查服务器日志,应该看到 "context canceled" 日志
```

---

## ✅ 验证清单

实现完成后,请检查:

**功能正确性**:
- [ ] 所有数据库操作使用 `*Context` 方法
- [ ] 每个 Handler 都创建了带超时的 Context
- [ ] 超时错误返回 `408 Request Timeout`
- [ ] 取消错误不返回响应(客户端已断开)

**资源管理**:
- [ ] 所有 `cancel()` 都通过 `defer` 调用
- [ ] 没有 Context 泄漏(使用 pprof 检查 goroutine 数量)

**错误处理**:
- [ ] 区分 `DeadlineExceeded` 和 `Canceled`
- [ ] 日志记录包含错误类型
- [ ] 用户友好的错误提示

**性能**:
- [ ] 超时时间设置合理(根据实际测试)
- [ ] 没有不必要的 Context 创建

---

## 💡 常见问题 FAQ

### Q1: 为什么不用 `http.TimeoutHandler`?

**答**:
- `http.TimeoutHandler` 是全局超时,所有请求统一时间
- 无法针对不同操作设置不同超时
- 无法传递 Context 到数据库层
- 推荐使用 `context.WithTimeout` 更灵活

### Q2: Context.Value() 什么时候用?

**答**:
```go
// 用于传递请求范围的元数据(如用户ID、请求ID)
ctx = context.WithValue(ctx, "user_id", 123)
userID := ctx.Value("user_id").(int)
```

**注意**:
- 不要滥用 `Value`,只存储请求范围的数据
- 不要存储可选参数,用函数参数代替
- Key 应该是自定义类型,避免冲突:
  ```go
  type contextKey string
  const UserIDKey contextKey = "user_id"
  ctx = context.WithValue(ctx, UserIDKey, 123)
  ```

### Q3: 如何测试 Context 超时?

**答**:
```go
func TestListTodosTimeout(t *testing.T) {
    // 创建一个已经超时的 Context
    ctx, cancel := context.WithTimeout(context.Background(), 0)
    defer cancel()

    _, _, err := db.ListTodosContext(ctx, filter)
    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("expected timeout error, got %v", err)
    }
}
```

### Q4: Context 会影响性能吗?

**答**:
- Context 的开销极小(纳秒级)
- 主要开销是 timer goroutine(使用 `WithTimeout` 时)
- 使用 `defer cancel()` 可以提前释放资源
- 相比数据库查询的耗时,Context 开销可以忽略

---

## 🚀 实现步骤建议

**第 1 步**: 修改 `database/db.go`,为所有数据库方法添加 Context 参数

**第 2 步**: 修改 `handler/handler.go`,在 Handler 中创建带超时的 Context

**第 3 步**: 添加超时错误处理

**第 4 步**: 测试超时控制是否生效

---

**现在,开始实现 Context 超时控制吧!记住:defer cancel(),错误区分,合理超时。** 🚀
