package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"todo-list/model"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

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

	_, err := db.conn.Exec(schema)
	return err
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateTodo 创建待办事项
func (db *DB) CreateTodo(todo *model.Todo) error {
	query := `
  		INSERT INTO todos (title, description, status, priority, due_date, created_at, updated_at)
  		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(
		query,
		todo.Title,
		todo.Description,
		todo.Status,
		todo.Priority,
		todo.DueDate,
		todo.CreatedAt,
		todo.UpdatedAt,
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

// ListTodos获取所有待办事项
func (db *DB) ListTodos() ([]model.Todo, error) {
	query := `
  		SELECT id, title, description, status, priority, due_date,
  		       created_at, updated_at, completed_at
  		FROM todos
  		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query todos: %w", err)
	}
	defer rows.Close()

	todos := make([]model.Todo, 0)
	for rows.Next() {
		var todo model.Todo
		var dueDate, completedAt sql.NullString

		err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Description,
			&todo.Status,
			&todo.Priority,
			&dueDate,
			&todo.CreatedAt,
			&todo.UpdatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo: %w", err)
		}

		if dueDate.Valid {
			todo.DueDate = &dueDate.String
		}
		if completedAt.Valid {
			todo.CompletedAt = &completedAt.String
		}

		todos = append(todos, todo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return todos, nil
}

// GetTodoByID 根据ID获取待办事项
func (db *DB) GetTodoByID(id int) (*model.Todo, error) {
	query := `
  		SELECT id, title, description, status, priority, due_date,
  		       created_at, updated_at, completed_at
  		FROM todos
  		WHERE id = ?
	`

	var todo model.Todo
	var dueDate, completedAt sql.NullString

	err := db.conn.QueryRow(query, id).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Description,
		&todo.Status,
		&todo.Priority,
		&dueDate,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}

	if dueDate.Valid {
		todo.DueDate = &dueDate.String
	}
	if completedAt.Valid {
		todo.CompletedAt = &dueDate.String
	}

	return &todo, nil
}

// UpdateTodo 更新待办事项
func (db *DB) UpdateTodo(todo *model.Todo) error {
	query := `
  		UPDATE todos
  		SET title = ?, description = ?, status = ?, priority = ?,
  		    due_date = ?, updated_at = ?, completed_at = ?
  		WHERE id = ?
	`

	todo.UpdatedAt = time.Now()

	result, err := db.conn.Exec(
		query,
		todo.Title,
		todo.Description,
		todo.Status,
		todo.Priority,
		todo.DueDate,
		todo.UpdatedAt,
		todo.CompletedAt,
		todo.ID,
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
		return fmt.Errorf("todo not found")
	}

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
