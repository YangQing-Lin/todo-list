# 教学 Part 1: 筛选、搜索、分页 API

## 🎯 功能分析 - Linus 式思考

### Linus 的三个问题

**1. "这是个真问题还是臆想出来的？"**

✅ **真实问题**：
- 当前 `ListTodos()` 返回全部数据（database/db.go:49）
- 用户有 1000 条待办事项时，每次请求都加载全部数据
- 无法快速找到"今天到期的待办"或"包含'重要'关键词"的事项

**2. "有更简单的方法吗？"**

💡 **核心思路**：动态 SQL 构建
- ❌ 不要为每个查询组合写单独的函数（会有几十个函数）
- ✅ 一个函数 + 参数化查询 = 解决所有问题
- ✅ 保持向后兼容：所有参数可选

**3. "会破坏什么吗？"**

✅ **零破坏性设计**：
- 所有查询参数为可选（不传 = 使用默认值）
- 返回格式保持不变（仅增加 `total/limit/offset` 字段）
- 老客户端可以忽略新字段继续工作

---

## 📚 核心知识讲解

### 1. 动态 SQL 的"好品味"写法

**坏代码示例**（SQL 注入风险）:

```go
// ❌ 永远不要这样写！
query := "SELECT * FROM todos WHERE title LIKE '%" + search + "%'"
```

**为什么这是灾难？**
- 用户输入 `search = "'; DROP TABLE todos; --"` 会直接执行删除表的命令
- 任何特殊字符（单引号、分号）都可能破坏 SQL 结构

**好代码示例**（参数化查询）:

```go
// ✅ 正确写法
query := "SELECT * FROM todos WHERE title LIKE ?"
args := []interface{}{"%" + search + "%"}
rows, err := db.Query(query, args...)
```

**关键原则**：
- 用户输入永远不能直接拼接到 SQL 字符串
- 使用 `?` 占位符 + `args` 数组
- SQLite 驱动会自动转义特殊字符，防止 SQL 注入

---

### 2. 动态条件构建的模式

**核心技巧**：使用 `WHERE 1=1` 简化逻辑

```go
// 为什么用 WHERE 1=1？
query := "SELECT * FROM todos WHERE 1=1"  // 永远为真的条件

// 这样添加条件就不需要判断"是否是第一个 AND"
if status != "" {
    query += " AND status = ?"
    args = append(args, status)
}

if search != "" {
    query += " AND title LIKE ?"
    args = append(args, "%"+search+"%")
}
```

**对比：不用 WHERE 1=1 的复杂写法**

```go
// ❌ 复杂写法：需要判断是否第一个条件
query := "SELECT * FROM todos"
hasWhere := false

if status != "" {
    if !hasWhere {
        query += " WHERE"
        hasWhere = true
    } else {
        query += " AND"
    }
    query += " status = ?"
    args = append(args, status)
}

if search != "" {
    if !hasWhere {
        query += " WHERE"
        hasWhere = true
    } else {
        query += " AND"
    }
    query += " title LIKE ?"
    args = append(args, "%"+search+"%")
}
```

**Linus 说**："这就是'好品味'——让特殊情况消失。不用 WHERE 1=1 的话，你需要判断是否第一个条件，代码会更复杂。"

---

### 3. 分页的正确姿势

**两步走**：
1. **先查总数**（不带 LIMIT/OFFSET）
2. **再查数据**（带分页参数）

```go
// 步骤 1: 查询总数
countQuery := "SELECT COUNT(*) FROM todos WHERE ..."
var total int
db.QueryRow(countQuery, args...).Scan(&total)

// 步骤 2: 查询分页数据
query += " LIMIT ? OFFSET ?"
args = append(args, limit, offset)
rows, err := db.Query(query, args...)
```

**为什么要分两步？**
- 前端需要知道总数（显示"第 1-20 条，共 100 条"）
- 如果只查一次，无法知道总数
- `COUNT(*)` 查询很快（有索引的话几乎瞬间完成）

**分页参数说明**：
```
LIMIT 20 OFFSET 40

解释：
- LIMIT 20: 每页 20 条数据
- OFFSET 40: 跳过前 40 条（即第 3 页，因为 0-19 是第 1 页，20-39 是第 2 页）

公式：
- offset = (page - 1) * limit
- 第 1 页：offset = 0
- 第 2 页：offset = 20
- 第 3 页：offset = 40
```

---

## 💻 完整代码示例

### 文件 1: `database/db.go` - 新增 `TodoFilter` 和修改 `ListTodos()`

```go
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"todo-list/model"
)

// TodoFilter 查询过滤器
type TodoFilter struct {
	Status string // pending/completed/all
	Search string // 搜索标题和描述
	Sort   string // created_at/due_date/status
	Order  string // asc/desc
	Limit  int    // 每页数量
	Offset int    // 偏移量
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
		filter.Order = strings.ToUpper(filter.Order) // ASC/DESC 必须大写
	}
	if filter.Limit <= 0 {
		filter.Limit = 50 // 默认每页 50 条
	}
	if filter.Status == "" {
		filter.Status = "all"
	}

	// 构建基础查询
	baseQuery := "SELECT id, version, title, description, status, due_date, created_at, updated_at, completed_at FROM todos WHERE 1=1"
	args := []interface{}{}

	// 动态添加条件
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

	// 添加排序和分页（注意：sort 和 order 需要验证，防止 SQL 注入）
	// 白名单验证
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
			return nil, 0, fmt.Errorf("扫描行失败: %w", err)
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
```

**关键点解析**：

1. **默认值处理**（第 19-30 行）
   - 避免空值导致的错误
   - 保持向后兼容

2. **WHERE 1=1 技巧**（第 33 行）
   - 简化条件拼接逻辑
   - 消除"是否第一个条件"的判断

3. **参数化查询**（第 38-47 行）
   - 所有用户输入通过 `args` 数组传递
   - 防止 SQL 注入

4. **总数查询**（第 50-66 行）
   - 与数据查询使用相同的筛选条件
   - 不包含 LIMIT/OFFSET

5. **白名单验证**（第 69-84 行）
   - 列名和排序方向无法用 `?` 占位符
   - 必须通过白名单验证后才能拼接

---

### 文件 2: `handler/handler.go` - 修改 `ListTodos` handler

```go
package handler

import (
	"net/http"
	"strconv"

	"todo-list/database"
)

func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	// 解析分页参数
	limit := 50 // 默认值
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			// 限制最大值，防止恶意请求
			if limit > 200 {
				limit = 200
			}
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
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

	// 调用数据库层
	todos, total, err := h.db.ListTodos(filter)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "获取待办事项失败")
		return
	}

	// 返回结果（包含分页信息）
	sendSuccessResponse(w, map[string]interface{}{
		"todos":  todos,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, "获取待办事项成功")
}
```

**关键点解析**：

1. **参数解析**（第 12-15 行）
   - 字符串参数直接读取（空字符串会被数据库层处理为默认值）

2. **数字参数验证**（第 21-34 行）
   - 使用 `strconv.Atoi` 转换
   - 验证范围（limit > 0, offset >= 0）
   - 限制 limit 最大值（防止恶意请求）

3. **返回格式**（第 55-60 行）
   - 包含分页元信息（total, limit, offset）
   - 前端可以根据这些信息渲染分页组件

---

## ⚠️ 潜在陷阱和最佳实践

### 1. **SQL 注入防护**

**危险代码**：

```go
// ❌ 这是灾难性的写法
query := fmt.Sprintf("SELECT * FROM todos ORDER BY %s %s", sort, order)
```

**攻击示例**：
```
GET /api/todos?sort=id;DROP TABLE todos;--&order=ASC

生成的 SQL：
SELECT * FROM todos ORDER BY id;DROP TABLE todos;-- ASC

结果：todos 表被删除！
```

**安全代码**：

```go
// ✅ 使用白名单验证
allowedSortFields := map[string]bool{
    "created_at": true,
    "due_date": true,
    "status": true,
}

if !allowedSortFields[sort] {
    sort = "created_at"  // 使用默认值
}
```

**核心原则**：
- 参数化查询（`?` 占位符）只能防护**值**，不能防护**列名/表名**
- 列名、表名、排序方向必须用**白名单验证**
- 永远不要相信用户输入

---

### 2. **性能考虑**

**优化建议**：

```sql
-- 为常用查询字段添加索引
CREATE INDEX idx_status ON todos(status);
CREATE INDEX idx_created_at ON todos(created_at DESC);
CREATE INDEX idx_due_date ON todos(due_date);
```

**LIKE 查询的限制**：

```go
// ✅ 可以使用索引（前缀匹配）
search = "关键词%"

// ❌ 无法使用索引（中间匹配）
search = "%关键词%"
```

**为什么中间匹配无法使用索引？**
- 索引像字典一样按字母顺序排列
- 前缀匹配：可以直接定位到"关键词"开头的所有记录
- 中间匹配：无法定位，需要全表扫描

**解决方案**：
- 对于小数据量（<10000 条），全表扫描可以接受
- 对于大数据量，使用 SQLite FTS5（全文搜索扩展）
- 这是进阶话题，暂时不需要

---

### 3. **边界情况测试**

需要测试的场景：

```go
// 1. 空数据库
// 预期：返回 { todos: [], total: 0 }

// 2. 非法参数
filter := TodoFilter{
    Sort: "invalid_field",  // 应该回退到 "created_at"
    Order: "INVALID",       // 应该回退到 "DESC"
    Limit: -1,              // 应该使用默认值 50
    Offset: -100,           // 应该使用 0
}

// 3. SQL 注入尝试
filter.Search = "'; DROP TABLE todos; --"
// 预期：被参数化查询安全处理，作为普通字符串搜索

filter.Sort = "id; DROP TABLE todos; --"
// 预期：白名单验证失败，使用默认值 "created_at"

// 4. 超大分页
filter.Limit = 99999
// 预期：被限制为 200（在 handler 层限制）

// 5. 特殊字符搜索
filter.Search = "100% 完成"
// 预期：% 被转义为 %%，正常搜索

// 6. Unicode 字符
filter.Search = "🚀 重要任务"
// 预期：正常搜索（Go 和 SQLite 都支持 UTF-8）
```

---

## 🛠️ 实现步骤建议

### 渐进式实现（分4步）

**第 1 步**：先实现基础筛选（不带搜索和分页）

```go
func (db *DB) ListTodos(status string) ([]model.Todo, error) {
    query := "SELECT * FROM todos WHERE 1=1"
    args := []interface{}{}

    if status != "" && status != "all" {
        query += " AND status = ?"
        args = append(args, status)
    }

    rows, err := db.Query(query, args...)
    // ...
}
```

**测试**：
```bash
curl "http://localhost:7789/api/todos?status=pending"
curl "http://localhost:7789/api/todos?status=completed"
curl "http://localhost:7789/api/todos?status=all"
```

---

**第 2 步**：添加搜索功能

```go
if search != "" {
    query += " AND (title LIKE ? OR description LIKE ?)"
    searchPattern := "%" + search + "%"
    args = append(args, searchPattern, searchPattern)
}
```

**测试**：
```bash
curl "http://localhost:7789/api/todos?search=重要"
curl "http://localhost:7789/api/todos?status=pending&search=任务"
```

---

**第 3 步**：添加排序和分页

```go
// 白名单验证
allowedSortFields := map[string]bool{
    "created_at": true,
    "due_date": true,
    "status": true,
}

if !allowedSortFields[filter.Sort] {
    filter.Sort = "created_at"
}

// 注意：sort 和 order 已验证，可以安全拼接
query += fmt.Sprintf(" ORDER BY %s %s LIMIT ? OFFSET ?", filter.Sort, filter.Order)
args = append(args, filter.Limit, filter.Offset)
```

**常见错误**：
```go
// ❌ 错误：? 占位符不能用于列名
query += " ORDER BY ? ? LIMIT ? OFFSET ?"
args = append(args, filter.Sort, filter.Order, filter.Limit, filter.Offset)

// ✅ 正确：列名必须拼接（但要先验证！）
query += fmt.Sprintf(" ORDER BY %s %s", validatedSort, validatedOrder)
```

**测试**：
```bash
curl "http://localhost:7789/api/todos?sort=due_date&order=asc"
curl "http://localhost:7789/api/todos?limit=10&offset=0"
curl "http://localhost:7789/api/todos?limit=10&offset=10"
```

---

**第 4 步**：添加总数查询

```go
// 步骤 1: 查询总数（使用与数据查询相同的筛选条件）
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
err := db.conn.QueryRow(countQuery, countArgs...).Scan(&total)
if err != nil {
    return nil, 0, fmt.Errorf("查询总数失败: %w", err)
}

// 步骤 2: 查询分页数据
// ...

return todos, total, nil
```

**测试**：
```bash
# 查看返回的 total 字段是否正确
curl "http://localhost:7789/api/todos?limit=5"
# 预期：{ "todos": [...], "total": 100, "limit": 5, "offset": 0 }
```

---

## ✅ 验证清单

实现完成后，请检查：

**功能正确性**：
- [ ] 不传任何参数时，行为与原 API 一致
- [ ] 筛选功能正常（status=pending/completed/all）
- [ ] 搜索功能正常（匹配标题和描述）
- [ ] 排序功能正常（created_at/due_date/status, asc/desc）
- [ ] 分页功能正常（limit/offset 正确生效）
- [ ] 返回的 total 字段正确（反映筛选后的总数）

**安全性**：
- [ ] 所有用户输入都经过验证
- [ ] sort/order 使用白名单验证
- [ ] 使用参数化查询（args 数组）
- [ ] 测试 SQL 注入攻击（应该被安全处理）

**性能**：
- [ ] 为常用字段添加索引（status, created_at, due_date）
- [ ] 分页查询效率可接受（<100ms）
- [ ] 测试大数据量（10000 条）时的性能

**向后兼容性**：
- [ ] 不传参数时行为不变
- [ ] 返回格式保持兼容（仅增加字段）
- [ ] 老客户端可以正常工作

**代码质量**：
- [ ] 代码格式符合 gofmt
- [ ] 有清晰的注释
- [ ] 错误处理完善
- [ ] 变量命名清晰

---

## 🧪 测试示例

### 手动测试脚本

```bash
#!/bin/bash

BASE_URL="http://localhost:7789/api/v1/todos"

echo "=== 测试 1: 不带参数（默认行为） ==="
curl -s "$BASE_URL" | jq

echo -e "\n=== 测试 2: 筛选（只看待办） ==="
curl -s "$BASE_URL?status=pending" | jq

echo -e "\n=== 测试 3: 搜索 ==="
curl -s "$BASE_URL?search=重要" | jq

echo -e "\n=== 测试 4: 排序（按截止日期升序） ==="
curl -s "$BASE_URL?sort=due_date&order=asc" | jq

echo -e "\n=== 测试 5: 分页（第 1 页，每页 5 条） ==="
curl -s "$BASE_URL?limit=5&offset=0" | jq

echo -e "\n=== 测试 6: 组合查询 ==="
curl -s "$BASE_URL?status=pending&search=任务&sort=due_date&order=asc&limit=10&offset=0" | jq

echo -e "\n=== 测试 7: SQL 注入尝试（应该安全） ==="
curl -s "$BASE_URL?search=%27;%20DROP%20TABLE%20todos;%20--" | jq
curl -s "$BASE_URL?sort=id;DROP%20TABLE%20todos;--" | jq

echo -e "\n=== 测试 8: 非法参数（应该使用默认值） ==="
curl -s "$BASE_URL?sort=invalid&order=invalid&limit=-1&offset=-100" | jq

echo -e "\n=== 测试 9: 超大 limit（应该被限制） ==="
curl -s "$BASE_URL?limit=99999" | jq

echo -e "\n=== 测试 10: 验证 total 字段 ==="
curl -s "$BASE_URL?limit=1" | jq '.data | { total, returned: (.todos | length) }'
```

保存为 `scripts/test-filter.sh`，然后执行：
```bash
chmod +x scripts/test-filter.sh
./scripts/test-filter.sh
```

---

## 📖 扩展阅读

### Go 标准库文档
- [`database/sql` 包](https://pkg.go.dev/database/sql)
- [`sql.NullString` 处理 NULL 值](https://pkg.go.dev/database/sql#NullString)

### SQLite 文档
- [LIKE 操作符](https://www.sqlite.org/lang_expr.html#like)
- [索引优化](https://www.sqlite.org/queryplanner.html)
- [参数化查询](https://www.sqlite.org/lang_expr.html#varparam)

### 安全相关
- [OWASP SQL 注入](https://owasp.org/www-community/attacks/SQL_Injection)
- [参数化查询最佳实践](https://cheatsheetseries.owasp.org/cheatsheets/Query_Parameterization_Cheat_Sheet.html)

---

## 🚀 开始实现吧！

现在请你按照以上步骤：

1. **修改 `database/db.go`**：
   - 添加 `TodoFilter` 结构体
   - 修改 `ListTodos()` 函数签名和实现
   - 实现动态 SQL 构建逻辑

2. **修改 `handler/handler.go`**：
   - 解析查询参数
   - 构建 `TodoFilter`
   - 返回包含分页信息的响应

3. **测试**：
   - 使用上面的测试脚本验证功能
   - 确保所有边界情况都正常处理

**记住 Linus 的话**：

> "Talk is cheap. Show me the code."

遇到问题随时问我，我会审查你的代码并指出改进方向！💪

---

## 💡 常见问题 FAQ

### Q1: 为什么不直接用 ORM（如 GORM）？

**Linus 的回答**：
- 这是学习项目，目标是理解 SQL 和数据库交互的本质
- ORM 隐藏了太多细节，你不会真正理解发生了什么
- 标准库 `database/sql` 已经足够强大
- 过早抽象是万恶之源

### Q2: 动态 SQL 会不会影响性能？

**答**：
- 不会。SQL 字符串拼接在 Go 中是零成本的
- 真正的性能瓶颈在数据库查询，而不是字符串操作
- SQLite 会缓存查询计划（即使 SQL 略有不同）

### Q3: 为什么 total 要单独查询，不能从数据查询中获取？

**答**：
- SQL 的 `LIMIT` 会截断结果，无法获取总数
- `SELECT COUNT(*) OVER()` 可以在一次查询中获取，但 SQLite 不推荐（性能更差）
- 分两次查询更清晰，且 `COUNT(*)` 查询很快

### Q4: 如果用户频繁改变筛选条件，会不会导致大量数据库查询？

**答**：
- 是的，但这是正常的。数据库就是用来查询的
- 如果真的成为性能瓶颈，可以考虑：
  - 前端防抖（用户停止输入 500ms 后再发请求）
  - 缓存查询结果（简单场景用内存缓存，复杂场景用 Redis）
- 过早优化是万恶之源——先证明它是问题，再优化

### Q5: 白名单验证是不是太麻烦了？有没有自动化的方式？

**Linus 的回答**：
- 麻烦是好事！安全永远不嫌麻烦
- 如果你觉得麻烦，说明你写的代码太复杂了
- 三个字段的白名单（created_at, due_date, status）很简单
- 自动化工具（如 ORM）会让你失去对 SQL 的控制

---

**现在，开始写代码吧！记住：分步实现，逐步测试，保持简洁。** 🚀
