package database

import (
	"context"
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

	// 查询总数(带 Context)
	countQuery := "SELECT COUNT(*) FROM todos WHERE 1=1"

	// 动态添加查询条件
	if filter.Status != "" && filter.Status != "all" {
		whereClause := " AND status = ?"
		baseQuery += whereClause
		countQuery += whereClause
		args = append(args, filter.Status)
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		whereClause := " AND (title LIKE ? OR description LIKE ?)"
		baseQuery += whereClause
		countQuery += whereClause
		args = append(args, searchPattern, searchPattern)
	}

	var total int
	// 使用 QueryRowContext 而不是 QueryRow
	err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询总数失败：%w", err)
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
		return nil, 0, fmt.Errorf("查询失败：%w", err)
	}
	defer rows.Close()

	var todos []model.Todo
	for rows.Next() {
		/*
			检查 Context 是否已取消(可选, SQLite 可能不会自动检查)
			✅ 应该使用：
			- 长时间运行的循环（如处理大量数据）
			- I/O 密集型操作（如批量文件读取）
			- 可能被用户中断的任务（如 HTTP 请求）

			❌ 不需要使用：
			- 短时间操作（< 100ms）
			- CPU 密集型计算（Context 取消不能中断计算，反而增加开销）
			- 数据库驱动已经自动支持 Context 的场景
		*/
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
			// 不阻塞，继续执行
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
			return nil, 0, fmt.Errorf("扫描失败：%w", err)
		}

		if dueDate.Valid {
			if t, err := time.Parse(time.RFC3339, dueDate.String); err == nil {
				todo.DueDate = &t
			} else {
				return nil, 0, fmt.Errorf("解析 due_date 失败：%w", err)
			}
		}

		if completedAt.Valid {
			if t, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
				todo.CompletedAt = &t
			} else {
				return nil, 0, fmt.Errorf("解析 completed_at 失败：%w", err)
			}

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
		return nil, fmt.Errorf("查询统计信息失败：%w", err)
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

// BatchCompleteTodosContext 批量完成待办事项（全有或全无）
// 注意：使用命名返回值 (err error)，让 defer 能访问到错误
func (db *DB) BatchCompleteTodosContext(ctx context.Context, ids []int) (err error) {
	// 验证输入
	if len(ids) == 0 {
		return nil
	}

	if len(ids) > 100 {
		return fmt.Errorf("批量操作最多支持100个项目，当前：%d", len(ids))
	}

	// （使用 BeginTx 支持 Context）
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败：%w", err)
	}

	// 使用 defer 确保事务被处理
	defer func() {
		if err != nil {
			// 回滚时也记录错误
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("事务回滚失败：%v（原始错误：%v）", rbErr, err)
			}
			log.Println("事务回滚成功")
		}
	}()

	// 预先声明变量，避免在循环中使用 := 导致变量遮蔽
	var result sql.Result
	var rows int64
	now := time.Now().UTC()

	// 批量更新
	for _, id := range ids {
		// 检查 Context 是否已取消
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		default:
		}

		// 这里不能使用 := 来接收返回内容
		// 因为 err := 会遮蔽循环外部的 err 参数，导致 defer func() 无法正常回滚
		result, err = tx.ExecContext(ctx, `
            UPDATE todos
            SET status = 'completed',
                completed_at = ?,
                updated_at = ?
            WHERE id = ? AND status = 'pending'
		`, now, now, id)

		if err != nil {
			return fmt.Errorf("更新 ID %d 失败：%w", id, err)
		}

		// 检查是否真的更新了
		// 同样不能使用 :=
		rows, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("获取影响行数失败：%w", err)
		}

		if rows == 0 {
			return fmt.Errorf("待办事项 %d 不存在或已完成", id)
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败：%w", err)
	}

	return nil
}

// BatchDeleteTodosContext 批量删除待办事项（全有或全无）
// 注意：使用命名返回值 (err error)，让 defer 能访问到错误
func (db *DB) BatchDeleteTodosContext(ctx context.Context, ids []int) (err error) {
	// 验证输入
	if len(ids) == 0 {
		return nil
	}

	if len(ids) > 100 {
		return fmt.Errorf("批量操作最多支持100个项目，当前：%d", len(ids))
	}

	// 开启事务（使用 BeginTx 支持 Context）
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败：%w", err)
	}

	// 使用 defer 确保事务被处理
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("回滚失败：%v（原始错误：%v）", rbErr, err)
			}
		}
	}()

	// 预先声明变量，避免在循环中使用 := 导致变量遮蔽
	var result sql.Result
	var rows int64

	// 批量删除
	for _, id := range ids {
		// 检查 Context 是否已取消
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return err
		default:
		}

		result, err = tx.ExecContext(ctx, "DELETE FROM todos WHERE id = ?", id)

		if err != nil {
			return fmt.Errorf("删除 ID %d 失败：%w", id, err)
		}

		// 检查是否真的删除了
		rows, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("获取影响行数失败：%w", err)
		}

		if rows == 0 {
			err = fmt.Errorf("待办事项 %d 不存在", id)
			return err
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败：%w", err)
	}

	return nil
}

// BatchError 批量操作中的单个错误
type BatchError struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}

// BatchResult 批量操作结果
type BatchResult struct {
	SuccessCount int          `json:"success_count"`
	FailedCount  int          `json:"failed_count"`
	Errors       []BatchError `json:"errors,omitempty"`
}

// BatchCompleteTodosPartialContext 批量完成待办事项（部分成功策略）
// 与教学-5的 BatchCompleteTodosContext（全有或全无）不同，
// 本方法允许部分成功，记录失败的 ID 并返回给调用者。
func (db *DB) BatchCompleteTodosPartialContext(ctx context.Context, ids []int) (result *BatchResult, err error) {
	if len(ids) == 0 {
		return &BatchResult{}, nil
	}

	// 限制批量大小
	if len(ids) > 100 {
		return nil, fmt.Errorf("批量操作最多支持 100 个 ID，当前：%d", len(ids))
	}

	// 使用 BeginTx 支持 Context
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 使用 defer 确保事务被处理
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("回滚失败: %v (原始错误: %v)", rbErr, err)
			}
		}
	}()

	result = &BatchResult{
		Errors: make([]BatchError, 0),
	}

	// 预先声明变量，避免在循环中使用 := 导致变量遮蔽
	var res sql.Result
	var rowsAffected int64

	for _, id := range ids {
		// 检查 Context 是否已取消
		select {
		case <-ctx.Done():
			err = ctx.Err() // 赋值给命名返回值，触发 defer 回滚
			return nil, err
		default:
		}

		// 在 Go 层生成时间戳（统一使用 UTC）
		now := time.Now().UTC()

		res, err = tx.ExecContext(ctx, `
			UPDATE todos
			SET status = 'completed',
			    completed_at = ?,
			    updated_at = ?
			WHERE id = ? AND status = 'pending'
		`, now, now, id)

		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{
				ID:    id,
				Error: err.Error(),
			})
			err = nil // 重置 err，避免触发 defer 回滚（部分成功策略）
			continue
		}

		// 检查是否真的更新了（可能 ID 不存在或已经 completed）
		rowsAffected, err = res.RowsAffected()
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{
				ID:    id,
				Error: fmt.Sprintf("获取受影响行数失败：%v", err),
			})
			err = nil // 重置 err
			continue
		}
		if rowsAffected == 0 {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{
				ID:    id,
				Error: "待办事项不存在或已完成",
			})
		} else {
			result.SuccessCount++
		}
	}

	// 提交事务（即使有部分失败，成功的也要提交）
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// BatchDeleteTodosPartialContext 批量删除待办事项（部分成功策略）
// 注意：使用命名返回值 (err error)，让 defer 能访问到错误
func (db *DB) BatchDeleteTodosPartialContext(ctx context.Context, ids []int) (result *BatchResult, err error) {
	if len(ids) == 0 {
		return &BatchResult{}, nil
	}

	// 限制批量大小
	if len(ids) > 100 {
		return nil, fmt.Errorf("批量操作最多支持 100 个 ID，当前: %d", len(ids))
	}

	// 使用 BeginTx 支持 Context
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 使用 defer 确保事务被处理
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("回滚失败: %v (原始错误: %v)", rbErr, err)
			}
		}
	}()

	result = &BatchResult{
		Errors: make([]BatchError, 0),
	}

	// 预先声明变量，避免在循环中使用 := 导致变量遮蔽
	var res sql.Result
	var rowsAffected int64

	for _, id := range ids {
		// 检查 Context 是否已取消
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return nil, err
		default:
		}

		res, err = tx.ExecContext(ctx, `DELETE FROM todos WHERE id = ?`, id)

		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{
				ID:    id,
				Error: err.Error(),
			})
			err = nil // 重置 err，避免触发 defer 回滚（部分成功策略）
			continue
		}

		// 检查是否真的删除了
		rowsAffected, err = res.RowsAffected()
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{
				ID:    id,
				Error: fmt.Sprintf("获取受影响行数失败: %v", err),
			})
			err = nil // 重置 err
			continue
		}
		if rowsAffected == 0 {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{
				ID:    id,
				Error: "待办事项不存在",
			})
		} else {
			result.SuccessCount++
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}
