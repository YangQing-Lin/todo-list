package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"todo-list/model"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

var ErrVersionConflict = errors.New("todo version conflict")

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.initSchema(); err != nil {
		return nil, err
	}

	log.Printf("Database initialized at %s", dbPath)
	return db, nil
}

// initSchema 初始化数据库表
func (db *DB) initSchema() error {
	schema := `
  	CREATE TABLE IF NOT EXISTS todos (
  		id INTEGER PRIMARY KEY AUTOINCREMENT,
  		version INTEGER NOT NULL DEFAULT 1,
  		title TEXT NOT NULL,
  		description TEXT,
  		status TEXT NOT NULL DEFAULT 'pending',
  		priority INTEGER NOT NULL DEFAULT 1,
  		due_date TEXT,
  		created_at DATETIME NOT NULL,
  		updated_at DATETIME NOT NULL,
  		completed_at DATETIME
  	);

  	CREATE INDEX IF NOT EXISTS idx_status ON todos(status);
  	CREATE INDEX IF NOT EXISTS idx_created_at ON todos(created_at DESC);
	`

	if _, err := db.conn.Exec(schema); err != nil {
		return err
	}

	return db.ensureVersionColumn()
}

func (db *DB) ensureVersionColumn() error {
	rows, err := db.conn.Query(`PRAGMA table_info(todos);`)
	if err != nil {
		return fmt.Errorf("failed to inspect todos table: %w", err)
	}
	defer rows.Close()

	hasVersionColumn := false
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			return fmt.Errorf("failed to scan todos schema: %w", err)
		}
		if name == "version" {
			hasVersionColumn = true
			break
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate todos schema: %w", err)
	}

	if hasVersionColumn {
		return nil
	}

	alterStmt := `ALTER TABLE todos ADD COLUMN version INTEGER NOT NULL DEFAULT 1`
	if _, err := db.conn.Exec(alterStmt); err != nil {
		return fmt.Errorf("failed to add version column: %w", err)
	}

	if _, err := db.conn.Exec(`UPDATE todos SET version = 1 WHERE version IS NULL`); err != nil {
		return fmt.Errorf("failed to backfill version column: %w", err)
	}

	return nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateTodo 创建待办事项
func (db *DB) CreateTodo(todo *model.Todo) error {
	query := `
  		INSERT INTO todos (title, description, status, due_date, created_at, updated_at, version)
  		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(
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

// TodoFilter 查询过滤器
type TodoFilter struct {
	Status string
	Search string
	Sort   string
	Order  string
	Limit  int
	Offset int
}

// ListTodos 获取待办事项列表（支持筛选、搜索、分页）
func (db *DB) ListTodos(filter TodoFilter) ([]model.Todo, int, error) {
	// 设置默认值
	if filter.Sort == "" {
		filter.Sort = "created_at"
	}
	if filter.Order == "" {
		filter.Order = "DESC"
	} else {
		filter.Order = strings.ToUpper(filter.Order) // 转换大写
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

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM todos WHERE 1=1"
	countArgs := []interface{}{}

	// 复制筛选条件到计数查询
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
	err := db.conn.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 添加排序和分页（验证 sort 和 order 防止 SQL 注入）
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

	// 这里 sort 和 order 已经验证过，可以安全拼接
	baseQuery += fmt.Sprintf(" ORDER BY %s %s LIMIT ? OFFSET ?", filter.Sort, filter.Order)
	args = append(args, filter.Limit, filter.Offset)

	// 执行查询
	rows, err := db.conn.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var todos []model.Todo
	for rows.Next() {
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

	return todos, total, nil
}

// GetTodoByID 根据ID获取待办事项
func (db *DB) GetTodoByID(id int) (*model.Todo, error) {
	query := `
  		SELECT id, version, title, description, status, due_date,
  		       created_at, updated_at, completed_at
  		FROM todos
  		WHERE id = ?
	`

	var todo model.Todo

	err := db.conn.QueryRow(query, id).Scan(
		&todo.ID,
		&todo.Version,
		&todo.Title,
		&todo.Description,
		&todo.Status,
		&todo.DueDate,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&todo.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}

	return &todo, nil
}

// UpdateTodo 更新待办事项
func (db *DB) UpdateTodo(todo *model.Todo) error {
	query := `
  		UPDATE todos
  		SET title = ?, description = ?, status = ?,
  		    due_date = ?, updated_at = ?, completed_at = ?, version = version + 1
  		WHERE id = ? AND version = ?
	`

	todo.UpdatedAt = time.Now()

	result, err := db.conn.Exec(
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

	// 查询上面语句匹配的行数
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

// DeleteTodo 删除待办事项
func (db *DB) DeleteTodo(id int) error {
	query := `DELETE FROM todos WHERE id = ?`

	result, err := db.conn.Exec(query, id)
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

// TodoStats 统计信息
type TodoStats struct {
	Total     int `json:"total"`     // 总数量
	Pending   int `json:"pending"`   // 未完成
	Completed int `json:"completed"` // 已完成
	Overdue   int `json:"overdue"`   // 已逾期
	Today     int `json:"today"`     // 今天到期
	ThisWeek  int `json:"this_week"` // 本周到期
}

// GetStats 获取待办事项统计信息
func (db *DB) GetStats() (*TodoStats, error) {
	// 获取 UTC 时间
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

	err := db.conn.QueryRow(query, now, today, today, weekLater).Scan(
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

	// 处理 NULL 值（空表时 SUM 返回 NULL）
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
