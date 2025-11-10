# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

Full-stack Todo List application - a Go learning project:
- **Backend**: Go 1.23.3 stdlib HTTP (no frameworks), SQLite
- **Frontend**: React 18 + TypeScript + Vite
- **Architecture**: RESTful API with dev proxy

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
