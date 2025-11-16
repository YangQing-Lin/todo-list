# 教学 Part 3: 数据库迁移与 Schema 清理

## 🎯 功能分析 - Linus 式思考

### Linus 的三个问题

**1. "这是个真问题还是臆想出来的？"**

✅ **真实问题**：
- 数据库 schema 中有 `priority INTEGER NOT NULL DEFAULT 1` 字段（`database/db.go:50`）
- 但 `model.Todo` 中完全没有 `Priority` 字段
- 导致数据不一致：数据库存储了从未被使用的数据
- 违反 YAGNI 原则："删除无用代码"

**为什么会出现这个问题？**
- 早期规划时可能考虑过优先级功能
- 后来发现不需要，但忘记清理数据库 schema
- **技术债累积**：时间越久，越难清理（可能有生产数据）

**2. "有更简单的方法吗？"**

💡 **核心思路**：数据库迁移（Migration）

**选项 A**：直接删除字段
```sql
-- ❌ SQLite 不支持！
ALTER TABLE todos DROP COLUMN priority;
```

**选项 B**：创建新表 + 迁移数据
```sql
-- ✅ SQLite 的标准方法
CREATE TABLE todos_new (...不含 priority...);
INSERT INTO todos_new SELECT ... FROM todos;
DROP TABLE todos;
ALTER TABLE todos_new RENAME TO todos;
```

**选项 C**：保留字段，标记为废弃
```go
// ⚠️ 技术债继续累积，不推荐
type Todo struct {
    // ...
    Priority int `json:"-" db:"priority"` // 废弃字段，不暴露给 API
}
```

**推荐**：选项 B（彻底删除）

**3. "会破坏什么吗？"**

⚠️ **潜在风险**：
- 现有数据会丢失 `priority` 字段的值（但这个字段从未被使用，所以无影响）
- 数据库文件格式不变（只是表结构变化）
- **向后兼容性**：确保迁移脚本在所有环境中都能自动执行

✅ **零破坏性设计**：
- 自动检测 schema 版本
- 未迁移的数据库自动执行迁移
- 已迁移的数据库跳过迁移
- 迁移失败时回滚（使用事务）

---

## 📚 核心知识讲解

### 1. 数据库迁移（Migration）是什么？

**定义**：数据库 schema 的版本管理和升级过程。

**类比**：就像 Git 管理代码版本，Migration 管理数据库版本。

```
V1: CREATE TABLE todos (id, title, description)
    ↓ 迁移脚本
V2: ALTER TABLE todos ADD COLUMN status TEXT
    ↓ 迁移脚本
V3: ALTER TABLE todos DROP COLUMN priority
```

**为什么需要迁移？**
- 代码在演进，数据库结构也在演进
- 不能要求用户手动删除数据库（会丢失数据）
- 必须支持"从旧版本无缝升级到新版本"

---

### 2. SQLite 的 ALTER TABLE 限制

**PostgreSQL / MySQL 支持的操作**：
```sql
ALTER TABLE todos DROP COLUMN priority;        -- ✅ 支持
ALTER TABLE todos MODIFY COLUMN title TEXT;    -- ✅ 支持
ALTER TABLE todos ADD CONSTRAINT ...;          -- ✅ 支持
```

**SQLite 支持的操作**：
```sql
ALTER TABLE todos ADD COLUMN new_col TEXT;     -- ✅ 仅支持这个
ALTER TABLE todos RENAME TO todos_old;         -- ✅ 仅支持这个
ALTER TABLE todos RENAME COLUMN old TO new;    -- ✅ SQLite 3.25+ 支持

ALTER TABLE todos DROP COLUMN priority;        -- ❌ 不支持
ALTER TABLE todos MODIFY COLUMN ...;           -- ❌ 不支持
```

**为什么 SQLite 这么限制？**
- SQLite 设计目标：简单、轻量、嵌入式
- 表结构存储在单个二进制文件中，修改复杂
- **官方推荐**：使用"创建新表 + 迁移数据"的方式

---

### 3. 安全的字段删除步骤（SQLite 标准方法）

**完整流程**（12 步）：

```sql
-- 1. 开启事务（确保原子性）
BEGIN TRANSACTION;

-- 2. 创建新表（不含 priority 字段）
CREATE TABLE todos_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version INTEGER NOT NULL DEFAULT 1,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    due_date TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    completed_at DATETIME
);

-- 3. 迁移数据（仅复制需要的字段）
INSERT INTO todos_new (id, version, title, description, status, due_date, created_at, updated_at, completed_at)
SELECT id, version, title, description, status, due_date, created_at, updated_at, completed_at
FROM todos;

-- 4. 删除旧表
DROP TABLE todos;

-- 5. 重命名新表
ALTER TABLE todos_new RENAME TO todos;

-- 6. 重建索引
CREATE INDEX idx_status ON todos(status);
CREATE INDEX idx_created_at ON todos(created_at DESC);

-- 7. 提交事务
COMMIT;
```

**关键点**：
- **步骤 1-7 必须在一个事务中执行**（要么全成功，要么全失败）
- **步骤 3** 明确列出要复制的字段（排除 `priority`）
- **步骤 6** 重建所有索引（DROP TABLE 会删除索引）

---

### 4. 自动迁移的实现原理

**目标**：应用启动时，自动检测并执行需要的迁移。

**核心思路**：维护一个 schema 版本号。

```go
// 方案 A: 使用独立的 schema_version 表
CREATE TABLE schema_version (version INTEGER PRIMARY KEY);
INSERT INTO schema_version VALUES (1);

// 方案 B: 使用 PRAGMA user_version（SQLite 内置）
PRAGMA user_version = 1;
```

**自动迁移逻辑**：

```go
func (db *DB) migrate() error {
    currentVersion := db.getSchemaVersion()  // 读取当前版本

    if currentVersion < 1 {
        // 执行迁移 V0 → V1
        db.migrateToV1()
        db.setSchemaVersion(1)
    }

    if currentVersion < 2 {
        // 执行迁移 V1 → V2（删除 priority）
        db.migrateToV2()
        db.setSchemaVersion(2)
    }

    // 未来的迁移...
    return nil
}
```

**优点**：
- 增量迁移（只执行需要的步骤）
- 可回溯（知道当前数据库在哪个版本）
- 可测试（模拟各个版本的升级路径）

---

## 💻 完整代码示例

### 文件 1: `database/db.go` - 添加迁移逻辑

```go
package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"todo-list/model"
)

// Schema 版本常量
const (
	CurrentSchemaVersion = 2  // 当前最新版本
)

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	// 初始化 schema（首次创建表）
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// 执行自动迁移
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("Database initialized at %s (schema version: %d)", dbPath, CurrentSchemaVersion)
	return db, nil
}

// initSchema 初始化数据库表（仅在表不存在时执行）
func (db *DB) initSchema() error {
	// 检查 todos 表是否存在
	var tableName string
	err := db.conn.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='todos'`).Scan(&tableName)

	if err == sql.ErrNoRows {
		// 表不存在，创建新表（使用最新 schema，不含 priority）
		log.Println("Creating todos table with latest schema...")
		schema := `
			CREATE TABLE todos (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				version INTEGER NOT NULL DEFAULT 1,
				title TEXT NOT NULL,
				description TEXT,
				status TEXT NOT NULL DEFAULT 'pending',
				due_date TEXT,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL,
				completed_at DATETIME
			);

			CREATE INDEX idx_status ON todos(status);
			CREATE INDEX idx_created_at ON todos(created_at DESC);
		`
		if _, err := db.conn.Exec(schema); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}

		// 设置 schema 版本为最新
		if err := db.setSchemaVersion(CurrentSchemaVersion); err != nil {
			return fmt.Errorf("failed to set schema version: %w", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	// 表已存在，跳过初始化（由 migrate() 处理升级）
	return nil
}

// migrate 执行数据库迁移
func (db *DB) migrate() error {
	currentVersion := db.getSchemaVersion()
	log.Printf("Current schema version: %d", currentVersion)

	// 迁移 V1 → V2: 删除 priority 字段
	if currentVersion < 2 {
		log.Println("Migrating to schema version 2 (removing priority column)...")
		if err := db.migrateToV2(); err != nil {
			return fmt.Errorf("failed to migrate to V2: %w", err)
		}
		if err := db.setSchemaVersion(2); err != nil {
			return fmt.Errorf("failed to update schema version: %w", err)
		}
		log.Println("Migration to V2 completed successfully")
	}

	// 未来的迁移在这里添加
	// if currentVersion < 3 { ... }

	return nil
}

// migrateToV2 删除 priority 字段
func (db *DB) migrateToV2() error {
	// 检查 priority 字段是否存在
	hasPriority, err := db.hasColumn("todos", "priority")
	if err != nil {
		return err
	}

	if !hasPriority {
		log.Println("Priority column does not exist, skipping migration")
		return nil
	}

	// 使用事务确保原子性
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 确保事务结束时回滚（如果没有提交）
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 步骤 1: 创建新表（不含 priority）
	_, err = tx.Exec(`
		CREATE TABLE todos_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version INTEGER NOT NULL DEFAULT 1,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			due_date TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			completed_at DATETIME
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// 步骤 2: 迁移数据
	_, err = tx.Exec(`
		INSERT INTO todos_new (id, version, title, description, status, due_date, created_at, updated_at, completed_at)
		SELECT id, version, title, description, status, due_date, created_at, updated_at, completed_at
		FROM todos
	`)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// 步骤 3: 删除旧表
	_, err = tx.Exec(`DROP TABLE todos`)
	if err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// 步骤 4: 重命名新表
	_, err = tx.Exec(`ALTER TABLE todos_new RENAME TO todos`)
	if err != nil {
		return fmt.Errorf("failed to rename table: %w", err)
	}

	// 步骤 5: 重建索引
	_, err = tx.Exec(`
		CREATE INDEX idx_status ON todos(status);
		CREATE INDEX idx_created_at ON todos(created_at DESC);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getSchemaVersion 获取当前 schema 版本
func (db *DB) getSchemaVersion() int {
	var version int
	err := db.conn.QueryRow(`PRAGMA user_version`).Scan(&version)
	if err != nil {
		log.Printf("Failed to get schema version: %v, assuming version 0", err)
		return 0
	}
	return version
}

// setSchemaVersion 设置 schema 版本
func (db *DB) setSchemaVersion(version int) error {
	// 注意：PRAGMA user_version 不支持参数化查询
	query := fmt.Sprintf(`PRAGMA user_version = %d`, version)
	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}
	return nil
}

// hasColumn 检查表中是否存在某个列
func (db *DB) hasColumn(tableName, columnName string) (bool, error) {
	rows, err := db.conn.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false, fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

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
			return false, fmt.Errorf("failed to scan column info: %w", err)
		}
		if name == columnName {
			return true, nil
		}
	}

	return false, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// ... 其他函数（CreateTodo, ListTodos, UpdateTodo, DeleteTodo 等保持不变）
```

---

### 关键点解析

**1. Schema 版本管理**（第 14-16 行）
```go
const CurrentSchemaVersion = 2  // 定义当前最新版本
```
- 每次修改 schema，递增这个版本号
- 便于追踪数据库演进历史

**2. PRAGMA user_version**（第 114-123 行）
```go
// 读取版本
PRAGMA user_version

// 写入版本（注意：不支持参数化，需要字符串拼接）
PRAGMA user_version = 2
```
- SQLite 内置的用户自定义版本号
- 存储在数据库文件头部，不占用表空间
- 可选方案：创建 `schema_version` 表

**3. 事务保护**（第 87-95 行）
```go
tx, err := db.conn.Begin()
if err != nil {
    return fmt.Errorf("failed to begin transaction: %w", err)
}
defer func() {
    if err != nil {
        tx.Rollback()  // 发生错误时回滚
    }
}()

// ... 执行迁移操作（每步都要检查 err）...

if err = tx.Commit(); err != nil {
    return fmt.Errorf("failed to commit transaction: %w", err)
}
```
- 确保"要么全成功，要么全失败"
- **必须检查 Commit 的返回值**（磁盘满等错误）
- 避免迁移到一半失败导致数据损坏

**4. 幂等性检查**（第 78-83 行）
```go
hasPriority, err := db.hasColumn("todos", "priority")
if !hasPriority {
    log.Println("Priority column does not exist, skipping migration")
    return nil
}
```
- 如果字段已经删除，跳过迁移
- 允许多次执行迁移脚本（不会出错）

**5. PRAGMA table_info() 用法**（第 132-154 行）
```go
rows, err := db.conn.Query(`PRAGMA table_info(todos)`)
// 返回：cid, name, type, notnull, dflt_value, pk
```
- 检查表结构（列名、类型、约束）
- 实现运行时 schema 检测

---

## ⚠️ 潜在陷阱和最佳实践

### 1. **PRAGMA user_version 的陷阱**

**问题代码**：
```go
// ❌ 这会导致 SQL 注入风险！
func (db *DB) setSchemaVersion(version int) error {
    _, err := db.conn.Exec(`PRAGMA user_version = ?`, version)
    return err
}
```

**错误原因**：
- `PRAGMA` 语句不支持参数化查询
- 必须使用字符串拼接

**正确代码**：
```go
// ✅ 使用 fmt.Sprintf（version 是 int，不会有注入风险）
func (db *DB) setSchemaVersion(version int) error {
    query := fmt.Sprintf(`PRAGMA user_version = %d`, version)
    _, err := db.conn.Exec(query)
    return err
}
```

**安全性说明**：
- `version` 是 `int` 类型，不存在注入风险
- 如果是 `string` 类型，必须严格验证

---

### 2. **索引重建的陷阱**

**问题**：`DROP TABLE todos` 会删除所有索引！

```sql
-- 步骤 1-4: 创建新表、迁移数据、删除旧表、重命名
-- 此时 todos 表没有任何索引！

-- ❌ 如果忘记重建索引
-- 查询性能会严重下降（全表扫描）
```

**正确做法**：
```sql
-- ✅ 必须重建所有索引
CREATE INDEX idx_status ON todos(status);
CREATE INDEX idx_created_at ON todos(created_at DESC);
```

**如何避免遗漏？**
- 在迁移前，先查询现有索引：
  ```sql
  SELECT name, sql FROM sqlite_master WHERE type='index' AND tbl_name='todos';
  ```
- 迁移后，逐一重建

---

### 3. **外键约束的陷阱**

**场景**：如果 `todos` 表有外键关系

```sql
CREATE TABLE todos (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

**问题**：
- `DROP TABLE todos` 会打破外键约束
- 迁移过程中，外键约束可能失效

**解决方案**：
```sql
-- 步骤 0: 临时禁用外键检查
PRAGMA foreign_keys = OFF;

-- 步骤 1-5: 执行迁移...

-- 步骤 6: 重新启用外键检查
PRAGMA foreign_keys = ON;

-- 步骤 7: 验证外键完整性
PRAGMA foreign_key_check;
```

**注意**：
- 本项目没有外键，可以忽略
- 但未来如果添加外键，必须考虑这个问题

---

### 4. **迁移失败的回滚**

**问题场景**：
```go
// 迁移到一半失败了
tx.Exec(`CREATE TABLE todos_new (...)`)  // ✅ 成功
tx.Exec(`INSERT INTO todos_new ...`)     // ❌ 失败（如：磁盘满）
// 此时 todos_new 表已创建，但数据未复制
```

**正确处理**：
```go
tx, err := db.conn.Begin()
defer func() {
    if err != nil {
        log.Printf("Migration failed, rolling back: %v", err)
        tx.Rollback()  // 自动删除 todos_new 表
    }
}()

// ... 执行迁移 ...

if err = tx.Commit(); err != nil {
    return fmt.Errorf("failed to commit migration: %w", err)
}
```

**验证回滚是否成功**：
```bash
# 模拟磁盘满（测试环境）
# 1. 限制数据库文件大小
sqlite3 test.db "PRAGMA max_page_count = 100"

# 2. 插入大量数据触发错误
# 3. 检查 todos_new 表是否存在（应该不存在）
```

---

### 5. **生产环境的迁移策略**

**问题**：生产环境有 1000 万条数据，迁移需要 10 分钟，服务会挂起吗？

**答案**：取决于数据库锁机制。

**SQLite 的锁行为**：
- `BEGIN TRANSACTION` 期间，数据库被**独占锁定**
- 其他连接无法读写（会阻塞）
- 迁移时间越长，服务不可用时间越长

**缓解策略**：

**策略 A**：维护窗口迁移
```bash
# 1. 停止服务
# 2. 执行迁移
# 3. 重启服务
```

**策略 B**：蓝绿部署
```bash
# 1. 复制数据库文件
# 2. 在副本上执行迁移
# 3. 切换到新数据库
```

**策略 C**：分批迁移（复杂）
```sql
-- 1. 创建新表
-- 2. 分批复制数据（每次 10000 条）
INSERT INTO todos_new SELECT * FROM todos WHERE id BETWEEN 1 AND 10000;
-- 3. 应用层同时写入新旧表
-- 4. 完成后切换
```

**本项目推荐**：策略 A（学习项目，数据量小）

---

## 🛠️ 实现步骤建议

### 渐进式实现（分 4 步）

**第 1 步**：添加 schema 版本管理

```go
// database/db.go

const CurrentSchemaVersion = 2

func (db *DB) getSchemaVersion() int {
    var version int
    db.conn.QueryRow(`PRAGMA user_version`).Scan(&version)
    return version
}

func (db *DB) setSchemaVersion(version int) error {
    query := fmt.Sprintf(`PRAGMA user_version = %d`, version)
    _, err := db.conn.Exec(query)
    return err
}
```

**测试**：
```go
// 创建测试数据库
db, _ := New("test.db")
version := db.getSchemaVersion()
fmt.Println("Version:", version)  // 应该是 0（新数据库）

db.setSchemaVersion(2)
version = db.getSchemaVersion()
fmt.Println("Version:", version)  // 应该是 2
```

---

**第 2 步**：实现 `hasColumn` 辅助函数

```go
func (db *DB) hasColumn(tableName, columnName string) (bool, error) {
    rows, err := db.conn.Query(`PRAGMA table_info(` + tableName + `)`)
    if err != nil {
        return false, err
    }
    defer rows.Close()

    for rows.Next() {
        var cid int
        var name, dataType string
        var notNull, pk int
        var defaultVal sql.NullString

        rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk)
        if name == columnName {
            return true, nil
        }
    }

    return false, nil
}
```

**测试**：
```go
hasPriority, _ := db.hasColumn("todos", "priority")
fmt.Println("Has priority:", hasPriority)  // 应该是 true

hasInvalid, _ := db.hasColumn("todos", "invalid")
fmt.Println("Has invalid:", hasInvalid)    // 应该是 false
```

---

**第 3 步**：实现 `migrateToV2` 函数

```go
func (db *DB) migrateToV2() error {
    // 检查字段是否存在
    hasPriority, err := db.hasColumn("todos", "priority")
    if err != nil {
        return fmt.Errorf("failed to check column: %w", err)
    }
    if !hasPriority {
        log.Println("Priority column does not exist, skipping migration")
        return nil  // 跳过迁移
    }

    // 使用事务
    tx, err := db.conn.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer func() {
        if err != nil {
            tx.Rollback()
        }
    }()

    // 1. 创建新表
    _, err = tx.Exec(`CREATE TABLE todos_new (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        version INTEGER NOT NULL DEFAULT 1,
        title TEXT NOT NULL,
        description TEXT,
        status TEXT NOT NULL DEFAULT 'pending',
        due_date TEXT,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        completed_at DATETIME
    )`)
    if err != nil {
        return fmt.Errorf("failed to create new table: %w", err)
    }

    // 2. 迁移数据
    _, err = tx.Exec(`INSERT INTO todos_new (id, version, title, description, status, due_date, created_at, updated_at, completed_at)
        SELECT id, version, title, description, status, due_date, created_at, updated_at, completed_at
        FROM todos`)
    if err != nil {
        return fmt.Errorf("failed to copy data: %w", err)
    }

    // 3. 删除旧表
    _, err = tx.Exec(`DROP TABLE todos`)
    if err != nil {
        return fmt.Errorf("failed to drop old table: %w", err)
    }

    // 4. 重命名
    _, err = tx.Exec(`ALTER TABLE todos_new RENAME TO todos`)
    if err != nil {
        return fmt.Errorf("failed to rename table: %w", err)
    }

    // 5. 重建索引
    _, err = tx.Exec(`CREATE INDEX idx_status ON todos(status);
        CREATE INDEX idx_created_at ON todos(created_at DESC)`)
    if err != nil {
        return fmt.Errorf("failed to create indexes: %w", err)
    }

    // 提交
    if err = tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

**测试**：
```bash
# 1. 准备旧数据库（包含 priority 字段）
# 2. 执行迁移
db.migrateToV2()

# 3. 验证
hasPriority, _ := db.hasColumn("todos", "priority")
fmt.Println("Has priority after migration:", hasPriority)  // 应该是 false
```

---

**第 4 步**：集成到 `New()` 函数

```go
func New(dbPath string) (*DB, error) {
    conn, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    db := &DB{conn: conn}

    // 初始化 schema（首次创建）
    if err := db.initSchema(); err != nil {
        conn.Close()
        return nil, err
    }

    // 执行迁移
    if err := db.migrate(); err != nil {
        conn.Close()
        return nil, err
    }

    log.Printf("Database ready (schema version: %d)", CurrentSchemaVersion)
    return db, nil
}

func (db *DB) migrate() error {
    currentVersion := db.getSchemaVersion()

    if currentVersion < 2 {
        log.Println("Migrating to V2...")
        if err := db.migrateToV2(); err != nil {
            return err
        }
        db.setSchemaVersion(2)
        log.Println("Migration to V2 completed")
    }

    return nil
}
```

**测试**：
```bash
# 测试 1: 新数据库（应直接创建 V2 schema）
rm test.db
go run cmd/server/main.go
# 日志应显示：schema version: 2

# 测试 2: 旧数据库（应执行迁移）
# 1. 恢复旧数据库文件（包含 priority 字段）
# 2. 启动服务
go run cmd/server/main.go
# 日志应显示：Migrating to V2... Migration to V2 completed

# 测试 3: 已迁移的数据库（应跳过迁移）
go run cmd/server/main.go
# 日志应显示：schema version: 2（无迁移日志）
```

---

## ✅ 验证清单

实现完成后，请检查：

**功能正确性**：
- [ ] 新数据库创建时，直接使用 V2 schema（不含 priority）
- [ ] 旧数据库启动时，自动执行迁移
- [ ] 迁移后，`priority` 字段已删除
- [ ] 迁移后，索引已重建（idx_status, idx_created_at）
- [ ] 迁移后，schema version = 2
- [ ] 数据完整性（迁移前后 todo 数量一致）

**幂等性**：
- [ ] 多次执行迁移不会出错
- [ ] 已迁移的数据库跳过迁移

**事务性**：
- [ ] 迁移失败时自动回滚
- [ ] 不会出现"表已删除但数据未迁移"的情况

**向后兼容性**：
- [ ] 现有 API 不受影响
- [ ] model.Todo 结构体无需修改

**代码质量**：
- [ ] 有清晰的日志输出（迁移开始、完成、跳过）
- [ ] 错误处理完善
- [ ] 代码格式符合 gofmt

---

## 🧪 测试示例

### 手动测试脚本

```bash
#!/bin/bash

echo "=== 测试 1: 新数据库 ==="
rm -f test_new.db
DB_PATH=test_new.db go run cmd/server/main.go &
SERVER_PID=$!
sleep 2

# 检查 schema version
sqlite3 test_new.db "PRAGMA user_version;"  # 应该是 2

# 检查是否有 priority 字段
sqlite3 test_new.db "PRAGMA table_info(todos);" | grep priority  # 应该无输出

kill $SERVER_PID

echo -e "\n=== 测试 2: 旧数据库迁移 ==="
# 创建旧 schema（包含 priority）
sqlite3 test_old.db <<EOF
CREATE TABLE todos (
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

INSERT INTO todos (title, description, status, priority, created_at, updated_at)
VALUES ('测试任务', '测试描述', 'pending', 2, datetime('now'), datetime('now'));

CREATE INDEX idx_status ON todos(status);
CREATE INDEX idx_created_at ON todos(created_at DESC);

PRAGMA user_version = 1;
EOF

# 启动服务（应触发迁移）
DB_PATH=test_old.db go run cmd/server/main.go &
SERVER_PID=$!
sleep 2

# 检查 schema version
sqlite3 test_old.db "PRAGMA user_version;"  # 应该是 2

# 检查 priority 字段是否已删除
sqlite3 test_old.db "PRAGMA table_info(todos);" | grep priority  # 应该无输出

# 检查数据是否保留
sqlite3 test_old.db "SELECT title FROM todos;"  # 应该显示"测试任务"

# 检查索引是否重建
sqlite3 test_old.db "SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='todos';"
# 应该显示：idx_status, idx_created_at

kill $SERVER_PID

echo -e "\n=== 测试 3: 幂等性 ==="
# 再次启动（应跳过迁移）
DB_PATH=test_old.db go run cmd/server/main.go &
SERVER_PID=$!
sleep 2

# 日志应显示"schema version: 2"（无迁移日志）

kill $SERVER_PID

echo -e "\n=== 清理 ==="
rm -f test_new.db test_old.db
```

保存为 `scripts/test-migration.sh`，然后执行：
```bash
chmod +x scripts/test-migration.sh
./scripts/test-migration.sh
```

---

## 📖 扩展阅读

### SQLite 官方文档
- [ALTER TABLE 限制](https://www.sqlite.org/lang_altertable.html)
- [PRAGMA user_version](https://www.sqlite.org/pragma.html#pragma_user_version)
- [PRAGMA table_info](https://www.sqlite.org/pragma.html#pragma_table_info)
- [事务和锁](https://www.sqlite.org/lockingv3.html)

### 数据库迁移最佳实践
- [Evolutionary Database Design](https://martinfowler.com/articles/evodb.html)
- [Database Migrations](https://en.wikipedia.org/wiki/Schema_migration)

### Go 数据库迁移工具
- [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - 生产级迁移工具
- [pressly/goose](https://github.com/pressly/goose) - 轻量级迁移框架

---

## 🚀 开始实现吧！

现在请你按照以上步骤：

1. **修改 `database/db.go`**：
   - 添加 `CurrentSchemaVersion` 常量
   - 实现 `getSchemaVersion()` 和 `setSchemaVersion()`
   - 实现 `hasColumn()` 辅助函数
   - 实现 `migrateToV2()` 迁移逻辑
   - 在 `New()` 中调用 `migrate()`
   - 更新 `initSchema()`（新数据库直接创建 V2 schema）

2. **测试**：
   - 使用上面的测试脚本验证功能
   - 确保新数据库、旧数据库、迁移失败等场景都正确处理

3. **清理旧 schema**：
   - 从 `initSchema()` 中删除 `priority` 字段定义

**记住**：
- 迁移必须在事务中执行（原子性）
- 重建所有索引（不要遗漏）
- 实现幂等性（可以多次执行）
- 详细的日志输出（便于调试）

遇到问题随时问我！💪

---

## 💡 常见问题 FAQ

### Q1: 为什么不用 ALTER TABLE DROP COLUMN？

**答**：SQLite 不支持。这是 SQLite 的设计限制，必须使用"创建新表 + 迁移数据"的方法。

### Q2: 如果迁移到一半服务崩溃了怎么办？

**答**：
- 使用事务保护，崩溃时自动回滚
- 重启后，检测到 schema version 仍然是旧版本，会重新执行迁移
- 迁移是幂等的，多次执行不会出错

### Q3: PRAGMA user_version 存在哪里？

**答**：
- 存储在数据库文件头部（SQLite 文件格式的保留字段）
- 不占用表空间
- 使用 `PRAGMA user_version` 读取，`PRAGMA user_version = N` 设置

### Q4: 生产环境有 1000 万条数据，迁移会很慢吗？

**答**：
- 取决于数据大小和磁盘速度
- 估算：1000 万条 × 500 字节/条 = 5GB，复制可能需要 1-5 分钟
- 这段时间数据库被锁定，服务不可用
- 生产环境建议：维护窗口迁移或蓝绿部署

### Q5: 能否回滚到旧 schema？

**答**：
- 理论上可以（再次创建新表，添加 `priority` 字段）
- 但 **不推荐**：数据演进应该是单向的
- 如果真的需要回滚，提前备份数据库文件

### Q6: 如果未来要添加新字段怎么办？

**答**：
```go
// 在 migrate() 中添加新的迁移
if currentVersion < 3 {
    db.migrateToV3()  // 添加新字段
    db.setSchemaVersion(3)
}
```

SQLite 支持 `ALTER TABLE ADD COLUMN`，无需重建表：
```sql
ALTER TABLE todos ADD COLUMN new_field TEXT DEFAULT '';
```

---

**现在，开始实现数据库迁移逻辑吧！记住：事务、幂等性、日志。** 🚀
