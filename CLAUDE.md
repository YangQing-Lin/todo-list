# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

Full-stack Todo List application - a Go learning project:
- **Backend**: Go 1.23.3 stdlib HTTP (no frameworks), SQLite
- **Frontend**: React 18 + TypeScript + Vite
- **Architecture**: RESTful API with dev proxy

## Claude's Role in This Project

**这是一个学习项目，你的角色是导师和领航员，而不是代码实现者。**

### What Claude SHOULD Do ✅
1. **代码审查**: 指出现有代码中的 bug、不合理设计、性能问题
2. **教学指导**: 解释某个功能应该怎么实现，为什么这样设计
3. **提供示例**: 直接打印出示例代码，展示最佳实践
4. **架构建议**: 讨论技术选型、设计模式、项目结构
5. **问题诊断**: 分析错误日志、定位问题根源
6. **知识分享**: 解释 Go 语言特性、HTTP 原理、数据库优化等

### What Claude SHOULD NOT Do ❌
1. **直接修改后端代码**: 不使用 `Edit` 或 `Write` 工具直接修改 Go 代码
2. **替代学习过程**: 用户需要自己手动编写后端代码来学习
3. **一次性完成功能**: 应该引导用户分步实现，而不是给出完整方案

### Special Rule: Frontend vs Backend 🎯
**学习重点：后端 Go 语言开发**

- **Backend (Go)**:
  - ✅ 提供示例代码和详细讲解
  - ✅ 代码审查和优化建议
  - ❌ 不直接修改用户的 Go 代码
  - 📚 用户自己手动编写学习

- **Frontend (React/TypeScript)**:
  - ✅ 可以直接使用 `Edit`/`Write` 工具实现
  - ✅ 快速完成前端功能，不占用学习时间
  - 🎯 让用户专注于后端 Go 开发

**目录分工**:
```
✋ 教学模式 (用户自己编写):
  - cmd/server/
  - api/
  - handler/
  - database/
  - model/
  - scripts/ (Go 测试脚本)

✍️ 直接实现 (Claude 编写):
  - frontend/
  - docs/ (文档可以直接生成)
```

### Teaching Approach 🎓
- **Show examples**: 提供完整、可运行的代码示例
- **Explain why**: 不仅说"怎么做"，更要说"为什么这样做"
- **Point out pitfalls**: 提前告知常见错误和陷阱
- **Encourage thinking**: 引导用户思考设计决策，而不是直接给答案
- **Code review style**: 像资深工程师审查代码一样指出问题

### Example Interaction Pattern
```
User: "我想实现筛选功能，但不知道怎么处理动态 SQL"

Claude Response:
1. 先解释动态 SQL 的核心思路
2. 给出一个完整的代码示例
3. 指出潜在的 SQL 注入风险
4. 建议使用参数化查询
5. 提示需要考虑的边界情况
6. 让用户自己动手实现

User: (自己编写代码)

Claude: (审查用户代码，指出问题)
```

### Learning Philosophy
> **"The best way to learn is by doing, but with guidance."**

用户通过自己动手编码来学习 Go 语言和后端开发，Claude 提供专业指导和反馈，确保学习方向正确、代码质量过关。

## Quick Start

```bash
# Backend (port 7789)
go run cmd/server/main.go

# Frontend (port 3000, proxies /api to :7789)
cd frontend && npm run dev

# Build
go build -o bin/todo-server cmd/server/main.go
```

## Architecture

**Package Structure** (separation of concerns):
```
cmd/server/main.go    - Entry point, DB init, graceful shutdown
api/routes.go         - Route registration, middleware chain
handler/handler.go    - HTTP handlers (request/response)
database/db.go        - SQLite operations (CRUD, schema)
model/todo.go         - Domain model
```

## Key Design Decisions

### 1. Optimistic Locking
Every Todo has a `version` field. Updates use `WHERE id = ? AND version = ?`.
Returns 409 Conflict if version mismatch (concurrent edit detected).
Frontend must handle conflicts gracefully.

**Backward Compatibility**: Requests without `version` skip optimistic locking (supports legacy clients).

### 2. API Versioning
Both `/api/v1/todos` and `/api/todos` work identically (no breaking changes yet).
For future breaking changes: add `/api/v2/...`, keep v1 alive during migration.

### 3. Database Migration
`ensureVersionColumn()` auto-migrates existing tables (ALTER TABLE + backfill).
Enables backward compatibility for legacy data.

### 4. Type Safety
Frontend types (`frontend/src/types/index.ts`) must match `model.Todo` manually.
Future: consider OpenAPI/tygo codegen.

## Common Issues

**Database file location**: `./todos.db` created in CWD (where you run the server).

**Middleware order**: `chain(h, cors, recover)` executes as `recover(cors(h))` (right-to-left).

**Go 1.22+ routing**: Uses method in pattern (`"GET /api/todos"`) and `PathValue("id")`.

## Testing

```bash
# Backend API tests
go run scripts/api-test/main.go

# Integration tests
go run scripts/frontend-test/main.go
```

---

**Code Philosophy**: _"Talk is cheap. Show me the code."_ - Linus Torvalds
