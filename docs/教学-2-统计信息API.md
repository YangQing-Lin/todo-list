# 教学 Part 2: 统计信息 API

## 🎯 功能分析 - Linus 式思考

### Linus 的三个问题

**1. "这是个真问题还是臆想出来的？"**

✅ **真实问题**：
- 用户打开 Todo 应用时，第一眼想看到的是："我有多少待办？多少完成了？有几个逾期了？"
- 当前需要加载全部数据，在前端遍历计算统计信息（低效且浪费带宽）
- Dashboard 是应用的"控制面板"，统计数据是核心价值

**2. "有更简单的方法吗？"**

💡 **核心思路**：数据库聚合查询
- ❌ 不要在前端遍历计算（N 条数据 = N 次比较）
- ✅ 数据库一次查询搞定（使用 COUNT + WHERE）
- ✅ 利用 SQL 的聚合能力（GROUP BY, CASE WHEN）

**3. "会破坏什么吗？"**

✅ **零破坏性设计**：
- 新增独立端点 `GET /api/v1/todos/stats`
- 不修改现有 API
- 老客户端完全不受影响

---

## 📚 核心知识讲解

### 1. SQL 聚合函数的力量

**问题**：如何一次查询获取多个统计维度？

**笨方法**（多次查询）：
```go
// ❌ 低效：6 次数据库查询
total := db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&count)
pending := db.QueryRow("SELECT COUNT(*) FROM todos WHERE status = 'pending'").Scan(&count)
completed := db.QueryRow("SELECT COUNT(*) FROM todos WHERE status = 'completed'").Scan(&count)
// ... 还有 overdue, today, this_week
```

**聪明方法**（一次查询）：
```go
// ✅ 高效：1 次查询，使用 CASE WHEN
SELECT
    COUNT(*) as total,
    SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
    SUM(CASE WHEN status = 'pending' AND due_date < datetime('now') THEN 1 ELSE 0 END) as overdue,
    SUM(CASE WHEN status = 'pending' AND date(due_date) = date('now') THEN 1 ELSE 0 END) as today,
    SUM(CASE WHEN status = 'pending' AND date(due_date) BETWEEN date('now') AND date('now', '+7 days') THEN 1 ELSE 0 END) as this_week
FROM todos
```

**为什么这样更快？**
- 只需要扫描表一次
- 所有统计在一次遍历中完成
- 减少数据库往返次数（网络延迟）

---

### 2. SQLite 日期函数

SQLite 提供强大的日期处理能力：

```sql
-- 当前时间（ISO 8601 格式）
datetime('now')              -- "2024-01-15 14:30:00"
date('now')                  -- "2024-01-15"

-- 日期比较（逾期检测）
due_date < datetime('now')   -- 已经过了截止时间

-- 日期匹配（今天到期）
date(due_date) = date('now') -- 截止日期是今天

-- 日期范围（本周到期）
date(due_date) BETWEEN date('now') AND date('now', '+7 days')
```

**注意事项**：
- `due_date` 存储为 ISO 8601 字符串（如 "2024-01-20T15:00:00Z"）
- `date()` 函数提取日期部分，忽略时间
- `datetime('now')` 返回 UTC 时间

**时区问题**：
- SQLite 的 `datetime('now')` 返回 UTC 时间
- 如果 `due_date` 存储的是本地时间，需要调整比较逻辑
- 推荐：统一使用 UTC 时间存储，前端显示时转换为本地时间

---

### 3. CASE WHEN 表达式

`CASE WHEN` 是 SQL 中的 if-else：

```sql
-- 基础语法
CASE
    WHEN condition1 THEN result1
    WHEN condition2 THEN result2
    ELSE result3
END

-- 计数示例
SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END)
-- 解释：遍历每一行，如果 status = 'pending' 就贡献 1，否则贡献 0，最后求和
```

**常见模式**：

```sql
-- 条件计数
SUM(CASE WHEN condition THEN 1 ELSE 0 END)

-- 条件求和
SUM(CASE WHEN condition THEN amount ELSE 0 END)

-- 分类统计
SELECT
    status,
    COUNT(*) as count
FROM todos
GROUP BY status
```

---

## 💻 完整代码示例

### 文件 1: `database/db.go` - 新增 `GetStats()` 函数

```go
package database

import (
	"database/sql"
	"fmt"
)

// TodoStats 统计信息
type TodoStats struct {
	Total     int `json:"total"`      // 总数量
	Pending   int `json:"pending"`    // 未完成
	Completed int `json:"completed"`  // 已完成
	Overdue   int `json:"overdue"`    // 已逾期
	Today     int `json:"today"`      // 今天到期
	ThisWeek  int `json:"this_week"`  // 本周到期
}

// GetStats 获取待办事项统计信息
func (db *DB) GetStats() (*TodoStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND due_date < datetime('now') THEN 1 ELSE 0 END) as overdue,
			SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND date(due_date) = date('now') THEN 1 ELSE 0 END) as today,
			SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND date(due_date) BETWEEN date('now') AND date('now', '+7 days') THEN 1 ELSE 0 END) as this_week
		FROM todos
	`

	var stats TodoStats
	var pending, completed, overdue, today, thisWeek sql.NullInt64

	err := db.conn.QueryRow(query).Scan(
		&stats.Total,
		&pending,
		&completed,
		&overdue,
		&today,
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
	if today.Valid {
		stats.Today = int(today.Int64)
	}
	if thisWeek.Valid {
		stats.ThisWeek = int(thisWeek.Int64)
	}

	return &stats, nil
}
```

**关键点解析**：

1. **NULL 值处理**（第 26-30 行）
   - 空表时，`SUM()` 返回 NULL 而不是 0
   - 使用 `sql.NullInt64` 接收可能为 NULL 的值
   - 检查 `.Valid` 字段，避免误用 NULL

2. **due_date 判空**（第 15-19 行）
   - `due_date IS NOT NULL` 过滤掉没有设置截止日期的任务
   - 避免日期比较时出错

3. **日期范围**（第 19 行）
   - `BETWEEN date('now') AND date('now', '+7 days')` 表示"从今天到 7 天后"
   - 包含起始和结束日期（闭区间）

---

### 文件 2: `handler/handler.go` - 新增 `GetStats` handler

```go
// GetStats 获取统计信息
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetStats()
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "GET_STATS_ERROR", "获取统计信息失败")
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

**关键点**：
- 遵循项目统一的响应格式
- 错误处理使用 `sendError`（包含 error code）
- 成功响应手动构造 `Response` 结构体

---

### 文件 3: `api/routes.go` - 注册新路由

```go
func SetupRoutes(h *handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	withMiddlewares := func(f http.HandlerFunc) http.HandlerFunc {
		return chain(f, corsMiddleware, recoverMiddleware)
	}

	optionsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	registerTodoRoutes := func(base string) {
		mux.HandleFunc("GET "+base, withMiddlewares(h.ListTodos))
		mux.HandleFunc("POST "+base, withMiddlewares(h.CreateTodo))
		mux.HandleFunc("OPTIONS "+base, withMiddlewares(optionsHandler))

		// 新增：统计信息端点（注意：必须在 {id} 路由之前注册）
		mux.HandleFunc("GET "+base+"/stats", withMiddlewares(h.GetStats))

		mux.HandleFunc("PUT "+base+"/{id}", withMiddlewares(h.UpdateTodo))
		mux.HandleFunc("DELETE "+base+"/{id}", withMiddlewares(h.DeleteTodo))
		mux.HandleFunc("OPTIONS "+base+"/{id}", withMiddlewares(optionsHandler))
	}

	// Versioned routes with legacy aliases for backward compatibility
	registerTodoRoutes("/api/v1/todos")
	registerTodoRoutes("/api/todos")

	mux.HandleFunc("/health", h.HealthCheck)

	return mux
}
```

**⚠️ 重要：路由注册顺序**

```go
// ✅ 正确顺序
mux.HandleFunc("GET /api/v1/todos/stats", handler)  // 先注册具体路径
mux.HandleFunc("GET /api/v1/todos/{id}", handler)   // 后注册通配路径

// ❌ 错误顺序
mux.HandleFunc("GET /api/v1/todos/{id}", handler)   // {id} 会匹配 "stats"
mux.HandleFunc("GET /api/v1/todos/stats", handler)  // 永远不会被调用！
```

**为什么？**
- Go 1.22+ 路由器使用"最长前缀匹配"
- `/todos/stats` 更具体，优先级更高
- 但如果先注册 `{id}`，它会捕获所有 `/todos/*` 请求

---

## ⚠️ 潜在陷阱和最佳实践

### 1. **NULL 值的陷阱**

**问题代码**：
```go
// ❌ 这会导致 panic 或错误数据
var stats TodoStats
db.QueryRow(query).Scan(
    &stats.Total,
    &stats.Pending,   // int 类型无法接收 NULL！
    &stats.Completed,
    // ...
)
```

**错误原因**：
- 空表时，`SUM()` 返回 NULL
- Go 的 `int` 类型无法表示 NULL
- `Scan()` 会返回错误或设置为默认值 0（行为不确定）

**正确代码**：
```go
// ✅ 使用 sql.NullInt64 处理 NULL
var pending sql.NullInt64
db.QueryRow(query).Scan(&pending)

if pending.Valid {
    stats.Pending = int(pending.Int64)
} else {
    stats.Pending = 0  // 显式设置默认值
}
```

---

### 2. **时区问题**

**场景**：用户在东八区（UTC+8），服务器在 UTC

```go
// ❌ 可能出错的代码
// 服务器时间：2024-01-15 16:00:00 UTC
// 用户本地：2024-01-16 00:00:00 UTC+8

// due_date 存储为 "2024-01-16T00:00:00+08:00"
// datetime('now') 返回 "2024-01-15 16:00:00"（UTC）
// 比较结果：due_date > now（误判为"未到期"）
```

**解决方案**：

**方案 A**：统一使用 UTC 时间
```go
// 后端存储和比较都用 UTC
todo.DueDate = time.Now().UTC()

// 前端显示时转换
const localTime = new Date(todo.due_date).toLocaleString()
```

**方案 B**：SQLite 时区转换
```sql
-- 将 due_date 转换为 UTC 后比较
datetime(due_date, 'utc') < datetime('now')
```

**推荐**：方案 A（后端用 UTC，前端负责展示）

---

### 3. **性能考虑**

**问题**：统计查询会扫描全表吗？

**答案**：取决于是否有索引

```sql
-- 没有索引：全表扫描（10000 条 = 10000 次比较）
SELECT COUNT(*) FROM todos WHERE status = 'pending'

-- 有索引：索引扫描（10000 条 pending = 可能只需几十次比较）
CREATE INDEX idx_status ON todos(status);
SELECT COUNT(*) FROM todos WHERE status = 'pending'
```

**优化建议**：

```sql
-- 为常用查询字段添加索引
CREATE INDEX idx_status ON todos(status);
CREATE INDEX idx_due_date ON todos(due_date);

-- 复合索引（同时筛选 status 和 due_date）
CREATE INDEX idx_status_due_date ON todos(status, due_date);
```

**何时需要优化？**
- 数据量 < 1000 条：全表扫描可以接受（毫秒级）
- 数据量 > 10000 条：考虑添加索引
- 统计查询频繁（每秒 > 10 次）：必须优化

---

### 4. **空表边界情况**

**测试场景**：
```bash
# 数据库为空时
curl http://localhost:7789/api/v1/todos/stats

# 预期返回
{
  "success": true,
  "data": {
    "total": 0,
    "pending": 0,
    "completed": 0,
    "overdue": 0,
    "today": 0,
    "this_week": 0
  },
  "message": "获取统计信息成功"
}
```

**验证代码**：
```go
// 单元测试
func TestGetStats_EmptyDB(t *testing.T) {
    db := setupTestDB(t)
    stats, err := db.GetStats()

    assert.NoError(t, err)
    assert.Equal(t, 0, stats.Total)
    assert.Equal(t, 0, stats.Pending)
    // ...
}
```

---

## 🛠️ 实现步骤建议

### 渐进式实现（分3步）

**第 1 步**：实现基础统计（total, pending, completed）

```go
func (db *DB) GetStats() (*TodoStats, error) {
    query := `
        SELECT
            COUNT(*) as total,
            SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
            SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed
        FROM todos
    `

    var stats TodoStats
    var pending, completed sql.NullInt64

    err := db.conn.QueryRow(query).Scan(&stats.Total, &pending, &completed)
    // ... NULL 值处理

    return &stats, nil
}
```

**测试**：
```bash
# 创建几条测试数据
curl -X POST http://localhost:7789/api/v1/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "测试待办1"}'

curl -X POST http://localhost:7789/api/v1/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "测试待办2"}'

# 完成一条
curl -X PUT http://localhost:7789/api/v1/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"status": "completed"}'

# 查看统计
curl http://localhost:7789/api/v1/todos/stats
# 预期：{"total": 2, "pending": 1, "completed": 1}
```

---

**第 2 步**：添加日期相关统计（overdue, today, this_week）

```go
query := `
    SELECT
        COUNT(*) as total,
        SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
        SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
        SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND due_date < datetime('now') THEN 1 ELSE 0 END) as overdue,
        SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND date(due_date) = date('now') THEN 1 ELSE 0 END) as today,
        SUM(CASE WHEN status = 'pending' AND due_date IS NOT NULL AND date(due_date) BETWEEN date('now') AND date('now', '+7 days') THEN 1 ELSE 0 END) as this_week
    FROM todos
`
```

**测试**：
```bash
# 创建今天到期的任务
TODAY=$(date -u +"%Y-%m-%dT23:59:59Z")
curl -X POST http://localhost:7789/api/v1/todos \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"今天到期\", \"due_date\": \"$TODAY\"}"

# 创建已逾期的任务
YESTERDAY=$(date -u -d "yesterday" +"%Y-%m-%dT23:59:59Z")
curl -X POST http://localhost:7789/api/v1/todos \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"已逾期\", \"due_date\": \"$YESTERDAY\"}"

# 查看统计
curl http://localhost:7789/api/v1/todos/stats
# 预期：today=1, overdue=1
```

---

**第 3 步**：注册路由和添加 handler

```go
// handler/handler.go
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
    stats, err := h.db.GetStats()
    if err != nil {
        h.sendError(w, http.StatusInternalServerError, "GET_STATS_ERROR", "获取统计信息失败")
        return
    }

    response := Response{
        Success: true,
        Data:    stats,
        Message: "获取统计信息成功",
    }
    h.sendJSON(w, http.StatusOK, response)
}

// api/routes.go
registerTodoRoutes := func(base string) {
    // ... 现有路由
    mux.HandleFunc("GET "+base+"/stats", withMiddlewares(h.GetStats))
}
```

**测试**：
```bash
curl http://localhost:7789/api/v1/todos/stats
curl http://localhost:7789/api/todos/stats  # 向后兼容别名
```

---

## ✅ 验证清单

实现完成后，请检查：

**功能正确性**：
- [ ] 空表时返回全 0（不是 NULL 或错误）
- [ ] total 字段正确（等于 pending + completed）
- [ ] overdue 统计正确（只统计 pending 且已过期的）
- [ ] today 统计正确（截止日期是今天的）
- [ ] this_week 统计正确（7 天内到期的）

**API 设计**：
- [ ] 路由注册在 `/stats` 之前没有 `{id}` 通配符
- [ ] 同时支持 `/api/v1/todos/stats` 和 `/api/todos/stats`
- [ ] 响应格式符合项目规范（Response 结构体）

**代码质量**：
- [ ] NULL 值正确处理（使用 sql.NullInt64）
- [ ] 错误处理完善（返回有意义的错误信息）
- [ ] 代码格式符合 gofmt
- [ ] 有清晰的注释

**性能**：
- [ ] 单次查询获取所有统计（不是多次查询）
- [ ] 测试 10000 条数据时的响应时间（应 < 100ms）

**向后兼容性**：
- [ ] 新增端点，不影响现有 API
- [ ] 老客户端无需改动

---

## 🧪 测试示例

### 手动测试脚本

```bash
#!/bin/bash

BASE_URL="http://localhost:7789/api/v1/todos"

echo "=== 清理测试数据 ==="
# 假设有 DELETE ALL 端点或手动删除数据库

echo -e "\n=== 测试 1: 空表统计 ==="
curl -s "$BASE_URL/stats" | jq

echo -e "\n=== 创建测试数据 ==="
# 创建 5 个 pending 任务
for i in {1..5}; do
  curl -s -X POST "$BASE_URL" \
    -H "Content-Type: application/json" \
    -d "{\"title\": \"待办任务 $i\"}" > /dev/null
done

# 创建 3 个 completed 任务
for i in {1..3}; do
  RESPONSE=$(curl -s -X POST "$BASE_URL" \
    -H "Content-Type: application/json" \
    -d "{\"title\": \"已完成任务 $i\"}")
  ID=$(echo $RESPONSE | jq -r '.data.id')
  curl -s -X PUT "$BASE_URL/$ID" \
    -H "Content-Type: application/json" \
    -d '{"status": "completed"}' > /dev/null
done

# 创建今天到期的任务
TODAY=$(date -u +"%Y-%m-%dT23:59:59Z")
curl -s -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"今天到期\", \"due_date\": \"$TODAY\"}" > /dev/null

# 创建已逾期的任务
YESTERDAY=$(date -u -v-1d +"%Y-%m-%dT23:59:59Z")  # macOS
# YESTERDAY=$(date -u -d "yesterday" +"%Y-%m-%dT23:59:59Z")  # Linux
curl -s -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"已逾期\", \"due_date\": \"$YESTERDAY\"}" > /dev/null

echo -e "\n=== 测试 2: 统计信息 ==="
curl -s "$BASE_URL/stats" | jq

echo -e "\n=== 测试 3: 向后兼容（/api/todos/stats）==="
curl -s "http://localhost:7789/api/todos/stats" | jq

echo -e "\n=== 预期结果 ==="
echo "total: 10 (5 pending + 3 completed + 1 today + 1 overdue)"
echo "pending: 7 (5 + today + overdue)"
echo "completed: 3"
echo "overdue: 1"
echo "today: 1"
echo "this_week: 2 (today + overdue，如果 overdue 在本周内)"
```

保存为 `scripts/test-stats.sh`，然后执行：
```bash
chmod +x scripts/test-stats.sh
./scripts/test-stats.sh
```

---

## 📖 扩展阅读

### Go 标准库文档
- [`sql.NullInt64` - 处理 NULL 值](https://pkg.go.dev/database/sql#NullInt64)
- [`database/sql` - 查询模式](https://pkg.go.dev/database/sql)

### SQLite 文档
- [聚合函数](https://www.sqlite.org/lang_aggfunc.html)
- [CASE 表达式](https://www.sqlite.org/lang_expr.html#case)
- [日期和时间函数](https://www.sqlite.org/lang_datefunc.html)

### 最佳实践
- [NULL 值处理最佳实践](https://go.dev/doc/database/querying)
- [时区处理指南](https://www.sqlite.org/lang_datefunc.html#tmshf)

---

## 🚀 开始实现吧！

现在请你按照以上步骤：

1. **修改 `database/db.go`**：
   - 添加 `TodoStats` 结构体
   - 实现 `GetStats()` 函数
   - 正确处理 NULL 值

2. **修改 `handler/handler.go`**：
   - 添加 `GetStats` handler
   - 使用统一的响应格式

3. **修改 `api/routes.go`**：
   - 注册 `/stats` 路由
   - 注意路由顺序（在 `{id}` 之前）

4. **测试**：
   - 使用上面的测试脚本验证功能
   - 确保空表、正常数据、边界情况都正确处理

**记住**：
- 一次查询搞定所有统计（不要多次查询）
- NULL 值必须处理（空表时 SUM 返回 NULL）
- 路由注册顺序很重要（`/stats` 在 `/{id}` 前）

遇到问题随时问我！💪

---

## 💡 常见问题 FAQ

### Q1: 为什么不用 GROUP BY？

**答**：
- `GROUP BY` 适合"按某个维度分组"的场景（如：每个状态的数量）
- 这里需要"多个独立条件的计数"（pending、overdue、today 等）
- `CASE WHEN` + `SUM` 更直观，一次查询搞定

### Q2: sql.NullInt64 太麻烦了，有更简单的方法吗？

**答**：
```go
// 方案 A: 使用 COALESCE（SQL 层面处理 NULL）
SELECT
    COUNT(*) as total,
    COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending
FROM todos

// 这样可以直接用 int 接收
var stats TodoStats
db.QueryRow(query).Scan(&stats.Total, &stats.Pending)
```

**推荐**：两种方法都可以，`COALESCE` 更简洁，但 `sql.NullInt64` 更明确表达"这个值可能是 NULL"。

### Q3: 统计查询会不会很慢？

**答**：
- 小数据量（< 10000 条）：全表扫描可以接受（< 10ms）
- 大数据量：添加索引后依然很快（< 50ms）
- 统计查询通常不是性能瓶颈，除非每秒调用上百次

### Q4: this_week 包含今天吗？

**答**：
看你的业务需求：
```sql
-- 包含今天（0-7 天）
BETWEEN date('now') AND date('now', '+7 days')

-- 不包含今天（1-7 天）
BETWEEN date('now', '+1 day') AND date('now', '+7 days')
```

推荐：包含今天（更符合用户直觉）。

### Q5: 如果需要更多统计维度怎么办？

**答**：
- 继续添加 `CASE WHEN` 表达式
- 或者创建独立的统计端点（如：`/api/v1/todos/stats/advanced`）
- 记住 YAGNI 原则：先证明你需要，再添加

---

**现在，开始写代码吧！记住：一次查询，正确处理 NULL，注意路由顺序。** 🚀
