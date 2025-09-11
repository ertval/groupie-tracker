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

## Current Architecture (Updated September 2025)

### Clean Architecture - Recently Refactored
```
cmd/server/main.go           # Entry point with graceful shutdown
internal/
  ├── api/client.go         # External API consumption
  ├── handlers/handlers.go  # HTTP handlers (590+ lines)
  ├── models/models.go      # Core data structures
  ├── storage/store.go      # Unified store with auto-refresh (330+ lines)
  └── service/service.go    # Business logic layer (200+ lines)
templates/                  # Self-contained HTML templates (NO inheritance)
static/css/                 # Page-specific stylesheets
tests/                     # Audit compliance & E2E tests
doc/                       # Architecture & refactoring documentation
```

**✅ Architecture Issues Resolved:**
- Single `storage/store.go` (no more wrapper complexity)
- Unified store pattern with auto-refresh capability
- Clean separation between storage, service, and handler layers
- Proper error handling with template compatibility

### Current Store Features (September 2025)
```go
// Auto-refresh with configurable intervals
store := storage.NewStoreWithCache(apiClient)
store.StartAutoRefresh()    // Default: 1-hour refresh

// Thread-safe operations
func (s *Store) GetArtist(id int) (models.Artist, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    artist, found := s.artists[id]
    return artist, found
}
```

## Critical Data Flow Patterns

### API Data Normalization (in `internal/api/client.go`)
- `/api/artists` → direct array
- `/api/locations`, `/api/dates`, `/api/relation` → `{"index": [...]}` format
- Must extract `.Index` field for locations/dates/relations

### Handler Error Template Pattern
```go
// Error handlers expect specific struct fields
data := struct {
    Title        string
    ErrorCode    int      // NOT "Code" 
    RequestedURL string
    ExtraCSS     string
}{
    ErrorCode: 404,
    ExtraCSS:  "errors.css",
}
```

### Template System (Self-Contained)
- Each `.tmpl` file is complete HTML document
- No template inheritance or `{{define "content"}}` blocks
- Direct execution: `h.templates.ExecuteTemplate(w, "artist_detail.tmpl", data)`
- Template functions: `add`, `sub`, `contains`, `safeLen`

### Auto-Refresh Architecture
```go
// Server lifecycle integration
store.StartAutoRefresh()    // After initial data load
defer store.StopAutoRefresh() // On graceful shutdown

// Background refresh with proper error handling
🔄 Auto-refreshing data from API...
✅ Auto-refresh completed successfully
```

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

## Current Status (September 2025)

**✅ Recently Completed:**
- Fixed 404 error page display (ErrorCode vs Code mismatch)
- Removed "+X more" truncation in location artist lists
- Implemented auto-refresh system (1-hour intervals)
- Unified storage layer (no more base_store.go wrapper)
- Comprehensive error handling with proper template compatibility
- All tests passing (46+ comprehensive tests)

**🔧 Current Architecture:**
- Clean single-store pattern with auto-refresh
- Thread-safe operations with proper mutex handling
- Graceful server shutdown with auto-refresh cleanup
- Self-contained template system working correctly
- SEO-friendly URL slugs (/artists/queen vs /artists/28)

## Development Workflow

1. **Always write tests first** (Zone01 requirement)
2. **Use the unified store pattern** (`internal/storage/store.go`)
3. **Follow self-contained template pattern** (no inheritance)
4. **Test with audit data** (Queen, Gorillaz, Travis Scott, Foo Fighters)
5. **Check error template compatibility** (ErrorCode, ExtraCSS fields)

**File Reading Priority:**
1. `internal/storage/store.go` (unified store with auto-refresh)
2. `internal/handlers/handlers.go` (error handling patterns)
3. `templates/*.tmpl` (self-contained template examples)
4. `doc/BUG_FIXES_AND_FEATURES_SUMMARY.md` (recent changes)

**Testing Strategy:**
- All tests use audit-compliant data (Queen, Gorillaz, etc.)
- Test auto-refresh functionality with mock API client
- Verify template error handling with proper field structure
- Ensure no regression in Zone01 audit requirements
