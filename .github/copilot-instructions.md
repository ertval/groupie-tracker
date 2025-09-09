# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
Zone01 educational project implementing a Go web application that consumes the Groupie Trackers API to display band/artist information with client-server interactions. The project follows strict TDD principles and Zone01 coding standards.

## Key Constraints & Commands

**Critical Constraints:**
- Standard-library-only Go project — NEVER add third-party modules
- Follow Test-Driven Development: write `*_test.go` before implementation
- Server must never crash — implement panic recovery in all handlers

**Quick Commands:**
```bash
go run ./cmd/server/          # Start server (PORT=8080)
go test ./...                 # Run all tests
go test -cover ./...         # Coverage report
go build -o groupie-tracker ./cmd/server
```

## Current Architecture (As of Sept 2025)

### Project Structure
```
cmd/server/main.go           # Entry point with graceful shutdown
internal/
  ├── api/client.go         # External API consumption
  ├── handlers/handlers.go  # HTTP handlers (650+ lines)
  ├── models/models.go      # Core data structures
  ├── storage/
  │   ├── store.go         # Unified store (wrapper)
  │   └── base_store.go    # BaseStore + automatic cache (400+ lines)
  └── service/service.go   # Business logic layer (200+ lines)
templates/                  # Self-contained HTML templates (NO inheritance)
static/css/                 # Page-specific stylesheets
tests/                     # Audit compliance & E2E tests
```

### **Critical Architecture Issue** 🚨
The current storage/service layer is **over-complicated** with multiple abstractions:
- `Store` wraps `BaseStore` and `Service`
- Complex interface hierarchies (`DataReader`, `APIClient`)
- Duplicated functionality between storage and service layers
- **NEEDS REFACTORING** to single store + single service pattern

### Template System (Recently Fixed - Sept 2025)
**Self-Contained Templates** (NO template inheritance):
- Each `.tmpl` file is a complete HTML document
- No `{{define "content"}}` blocks or `{{template "base.tmpl" .}}`
- Direct execution: `h.templates.ExecuteTemplate(w, "locations.tmpl", data)`
- Template functions: `add`, `sub`, `contains`, `safeLen`

## Critical Data Flow Patterns

### Storage Threading
```go
// Read operations
s.mu.RLock()
defer s.mu.RUnlock()
data := s.artists[id]

// Write operations  
s.mu.Lock()
defer s.mu.Unlock()
s.artists[id] = artist
```

### API Data Normalization (in `internal/api/client.go`)
- `/api/artists` → direct array
- `/api/locations`, `/api/dates`, `/api/relation` → `{"index": [...]}` format
- Must extract `.Index` field for locations/dates/relations

### Handler Panic Recovery Pattern
```go
func (h *Handlers) SomeHandler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Panic recovered: %v", err)
            h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
        }
    }()
    // handler logic
}
```

## Known Issues & Immediate Tasks

### 🐛 Active Bug: Popular Locations Not Sorted
**Issue:** `templates/locations.tmpl` shows "Most Popular Locations" in arbitrary order
**Location:** `internal/handlers/handlers.go:calculateLocationStats()`
**Fix Required:** Sort locations by `ConcertCount` descending before returning

### 🔄 Refactoring Required
**Current Problem:** Over-engineered storage/service layers
**Goal:** Simplify to single store struct + single service struct
**Files to Refactor:**
- `internal/storage/store.go` (remove wrapper)
- `internal/storage/base_store.go` (rename to `store.go`)
- `internal/service/service.go` (simplify interfaces)

## Zone01 Audit Requirements (Test Against These)

**Critical Data Points:**
- Queen: exactly 7 members
- Gorillaz: first album date "26-03-2001"
- Travis Scott: 10+ concert locations
- Foo Fighters: exactly 6 members

**Required Endpoints:**
- `GET /api/search?q=` (full search)
- `GET /api/suggest?q=` (autocomplete)
- `POST /api/refresh` (refresh cached data)

## Current Status (Sept 2025)

**✅ Completed:**
- Self-contained template system (no inheritance conflicts)
- SEO-friendly URL slugs (/artists/queen vs /artists/28)
- 46 comprehensive tests with 100% pass rate
- Thread-safe storage with automatic cache refresh
- Client-server interactive events

**🔧 Needs Immediate Attention:**
1. Fix popular locations sorting bug
2. Refactor storage/service complexity
3. Update tests for refactored code
4. Update documentation

## Development Workflow

1. **Always write tests first** (Zone01 requirement)
2. **Start with the bug fix** (popular locations)
3. **Then refactor** storage/service layers
4. **Update tests** to match new architecture
5. **Update documentation** (README.md, todo.md)

**File Reading Priority:**
1. `internal/handlers/handlers.go` (lines 480-632 for locations bug)
2. `internal/storage/store.go` (wrapper complexity)
3. `internal/service/service.go` (business logic patterns)

**Testing Strategy:**
- Preserve existing test data (Queen, Gorillaz, etc.)
- Test refactored code maintains same public APIs
- Ensure no regression in audit compliance
