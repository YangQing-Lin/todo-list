# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Full-stack Todo List application built as a Go learning project:
- **Backend**: Go 1.23.3 using stdlib `net/http` (no web framework dependencies)
- **Database**: SQLite with `mattn/go-sqlite3` driver
- **Frontend**: React 18 + TypeScript + Vite
- **Architecture**: RESTful API with frontend proxy

## Development Commands

### Backend (Go)
```bash
# Run server (port 7789)
go run cmd/server/main.go

# Run backend API tests
go run scripts/api-test/main.go

# Run integration tests
go run scripts/frontend-test/main.go

# Build binary
go build -o bin/todo-server cmd/server/main.go
```

### Frontend (React)
```bash
cd frontend

# Install dependencies
npm install

# Start dev server (port 3000, proxies /api to :7789)
npm run dev

# Type check + production build
npm run build

# Lint
npm run lint

# Preview production build
npm run preview
```

## Architecture & Design Patterns

### Backend Architecture (Go)

**Package Structure** (clean separation of concerns):
```
cmd/server/main.go    - Entry point: DB init, server lifecycle, graceful shutdown
api/routes.go         - Route registration + middleware chain
handler/handler.go    - HTTP handlers (request/response logic)
database/db.go        - SQLite operations (CRUD, schema management)
model/todo.go         - Domain model + business logic
```

**Key Design Decisions**:

1. **Middleware Chain** (`api/routes.go:40-44`):
   - CORS middleware applied first (enables cross-origin requests)
   - Recover middleware catches panics (prevents server crashes)
   - Chain pattern: middlewares wrap handlers from right to left

2. **Graceful Shutdown** (`cmd/server/main.go:40-62`):
   - Signal handling (SIGINT/SIGTERM) via channel
   - 30-second grace period for in-flight requests
   - Clean DB connection closure on exit

3. **Optimistic Locking** (concurrency control):
   - Every Todo has a `version` field
   - Update operations use `WHERE id = ? AND version = ?`
   - Version incremented on successful update
   - Returns `ErrVersionConflict` if stale (concurrent edit detected)
   - Handler returns 409 Conflict with user-friendly message

4. **Unified Response Format** (`handler/handler.go:16-28`):
   ```go
   type Response struct {
       Success bool        `json:"success"`
       Data    interface{} `json:"data,omitempty"`
       Error   *ErrorInfo  `json:"error,omitempty"`
       Message string      `json:"message,omitempty"`
   }
   ```
   - All API responses use this wrapper
   - Consistent error codes (e.g., `VERSION_CONFLICT`, `VALIDATION_ERROR`)

5. **Database Schema Migration** (`database/db.go:67-111`):
   - `ensureVersionColumn()` checks for `version` column existence
   - Auto-migrates existing tables (ALTER TABLE + backfill)
   - Enables backward compatibility for legacy data

### API Versioning & Backward Compatibility

**Current Strategy** (`api/routes.go:58-69`):
```go
// Both routes work identically (no breaking changes)
registerTodoRoutes("/api/v1/todos")  // Versioned
registerTodoRoutes("/api/todos")     // Legacy alias
```

**Why This Matters**:
- Existing clients using `/api/todos` continue working
- New clients can use `/api/v1/todos` for clarity
- Future v2 can coexist without breaking old integrations

**Future Breaking Changes**:
When introducing incompatible changes:
1. Keep old version alive: `/api/v1/...` (legacy behavior)
2. Add new version: `/api/v2/...` (new behavior)
3. Deprecation timeline communicated to users
4. Never delete old routes until migration complete

### Frontend Architecture

**Proxy Configuration** (`frontend/vite.config.ts:10-14`):
- `/api/*` requests proxied to `http://localhost:7789`
- Simplifies development (no CORS issues)
- Production: use reverse proxy (nginx) or API gateway

**Type Safety**:
- `frontend/src/types/index.ts` - Shared TypeScript interfaces
- Must match backend `model.Todo` structure exactly
- `version` field required for optimistic locking

## Critical Implementation Rules

### When Modifying the API

1. **Never Break Existing Endpoints**:
   - Existing fields cannot be removed or renamed
   - Response structure changes must be additive only
   - Use versioned routes (`/api/v2/...`) for breaking changes

2. **Optimistic Locking Protocol**:
   - All update operations MUST include `version` in request body
   - Handler MUST validate version presence (`handler/handler.go:195-198`)
   - Database MUST use version in WHERE clause (`database/db.go:234`)
   - Return 409 Conflict if version mismatch

3. **Error Handling**:
   - Always log errors before sending HTTP response
   - Use structured error codes (uppercase snake_case)
   - Chinese messages for user-facing errors (target audience)
   - Never expose stack traces or DB internals to clients

### Database Operations

1. **Migration Safety**:
   - All schema changes must be backward compatible
   - Use `ensureVersionColumn()` pattern for new columns
   - Provide default values for new NOT NULL columns
   - Test migration on copy of production DB first

2. **Index Strategy** (`database/db.go:56-57`):
   ```sql
   CREATE INDEX IF NOT EXISTS idx_status ON todos(status);
   CREATE INDEX IF NOT EXISTS idx_created_at ON todos(created_at DESC);
   ```
   - `idx_status`: fast filtering by status (future feature)
   - `idx_created_at DESC`: optimized for default sort order

3. **NULL Handling**:
   - Use `sql.NullString`, `sql.NullTime` for nullable DB columns
   - Go struct uses pointers (`*time.Time`) for optional fields
   - Scanner handles NULL → nil conversion automatically

### Testing Approach

**Current State**: Manual testing via scripts
- `scripts/api-test/main.go` - Backend API smoke tests
- `scripts/frontend-test/main.go` - Full integration test

**Future Improvement** (from README.md planned features):
- Add unit tests with Go's `testing` package
- Table-driven tests for handler validation logic
- Mock `database.DB` interface for handler tests
- React component tests with Vitest

## Common Gotchas

1. **Go 1.22+ Route Pattern Syntax** (`api/routes.go:59-64`):
   ```go
   mux.HandleFunc("GET /api/todos", handler)      // ✅ Method in pattern
   mux.HandleFunc("PUT /api/todos/{id}", handler) // ✅ Path params
   ```
   - Requires Go 1.22+ (this project uses 1.23.3)
   - Older tutorials using `mux.HandleFunc("/api/todos", ...)` won't work the same

2. **Middleware Execution Order**:
   ```go
   chain(handler, corsMiddleware, recoverMiddleware)
   // Executes: recover → cors → handler
   ```
   - Right-to-left application (functional composition)
   - Recover must be outermost to catch all panics

3. **JSON Decoding + MaxBytesReader** (`handler/handler.go:119`):
   ```go
   r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
   ```
   - Prevents DoS via giant payloads
   - Must be set BEFORE `json.NewDecoder(r.Body)`

4. **Database File Location**:
   - `./todos.db` created in CWD when server starts
   - CWD = directory where `go run cmd/server/main.go` executed
   - For production: use absolute path or config file

5. **Frontend Type Sync**:
   - No automatic type generation (yet)
   - Manually keep `frontend/src/types/index.ts` in sync with `model/todo.go`
   - Future: consider `tygo` or OpenAPI codegen

## Learning Goals & Progression

**Current Phase** (Week 1-2 Complete):
- ✅ Go HTTP server with stdlib
- ✅ SQLite integration with proper schema
- ✅ React frontend with TypeScript
- ✅ Full CRUD operations
- ✅ Optimistic locking for concurrency

**Next Milestones** (from README.md):
- Priority system implementation
- Search/filter functionality
- Tagging system
- Unit test coverage
- Docker containerization

**Code Philosophy** (per README.md footer):
> "Talk is cheap. Show me the code." - Linus Torvalds

Focus on working implementations over theoretical design.