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
go test -cover ./...         # Coverage report (75.8%)
go build -o groupie-tracker ./cmd/server
```

## Current Architecture (September 2025)

### Clean Repository Pattern
```
cmd/server/main.go           # Entry point with graceful shutdown
cmd/server/server.go         # Server configuration and middleware
internal/
  ├── repository/            # Core data management
  │   ├── repository.go      # Complete repository (324 lines, 80.4% coverage)
  │   └── repository_test.go # Comprehensive tests
  └── handlers/              # HTTP handlers
      ├── handlers.go        # All endpoints (366 lines, 71.2% coverage)
      └── handlers_test.go   # Handler tests
templates/                   # Self-contained HTML templates
static/css/                  # Page-specific stylesheets
tests/                      # End-to-end and audit tests
```

**🏗️ Current Architecture:**
- Repository pattern: `repository.Repository` manages all data operations
- Single initialization: Load data once at startup via `repo.LoadData(ctx)`
- Precomputed indexes: SEO slugs, location stats calculated at load time
- Thread-safe operations through repository methods

### Repository Pattern (September 2025)
```go
// Repository initialization in server startup
repo := repository.NewRepository(apiURL, timeout)
if err := repo.LoadData(ctx); err != nil {
    log.Fatalf("Failed to load data: %v", err)
}

// All data access through repository methods
artists := repo.GetArtists()
artist, found := repo.GetArtistBySlug("queen")
locationStats := repo.GetLocationStats()
stats := repo.GetStats()
```

## Critical Data Flow Patterns

### API Data Normalization (in `internal/repository/repository.go`)
- `/api/artists` → direct array of Artist structs
- `/api/locations`, `/api/dates`, `/api/relation` → `{"index": [...]}` format
- Must extract `.Index` field for locations/dates/relations

### Handler Error Template Pattern
```go
// Error handlers expect specific struct fields
data := ErrorData{
    Title:        "Page Not Found",
    ExtraCSS:     "errors.css",
    ErrorCode:    404,        // NOT "Code" 
    RequestedURL: r.URL.Path,
    Message:      "The page you're looking for doesn't exist.",
    Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
}
```

### Template System (Self-Contained)
- Each `.tmpl` file is complete HTML document
- No template inheritance or `{{define "content"}}` blocks
- Direct execution: `h.templates.ExecuteTemplate(w, "artist_detail.tmpl", data)`
- Template functions: `add`, `sub`, `join`, `generateLocationSlug`, `normalizeLocationName`

### Improved Error Handling (September 2025)
```go
func (h *Handler) render(w http.ResponseWriter, templateName string, data any, statusCode ...int) {
    // Handle nil templates gracefully (for tests)
    if h.templates == nil {
        w.Write([]byte("Internal server error - templates not loaded"))
        return
    }
    
    // If template fails and it's not error template, try to render error template
    if err := h.templates.ExecuteTemplate(w, templateName, data); err != nil {
        if templateName != "error.tmpl" {
            // Try error template, fallback to plain text if that fails too
        }
    }
}
```

## Zone01 Audit Requirements (Test Against These)

**Critical Data Points:**
- Queen: exactly 7 members
- Gorillaz: first album date "26-03-2001" 
- Travis Scott: 10+ concert locations
- Foo Fighters: exactly 6 members

**Required Endpoints:**
- `GET /` (home page)
- `GET /artists` (all artists)
- `GET /artists/{slug}` (artist detail via SEO slug)
- `GET /locations` (all locations)
- `GET /locations/{slug}` (location detail)
- `GET /health` (JSON health check)

## Current Status (September 2025)

**✅ Recently Completed:**
- Fixed all failing tests - repository tests now match current API structure
- Achieved 75.8% test coverage (exceeded 70% target)
- Fixed 500 error template handling - now properly renders error template when available
- Enhanced handlers test coverage (71.2%) with comprehensive test cases
- Improved repository test coverage (80.4%) with edge case testing
- Unified repository pattern with single data load (no wrapper complexity)
- Enhanced error handling with nil template protection for tests
- All tests passing (comprehensive test suite)

**🔧 Current Architecture:**
- Clean repository pattern with single data load at startup
- Thread-safe operations through repository methods
- Graceful server shutdown with proper resource cleanup
- Self-contained template system working correctly
- SEO-friendly URL slugs (/artists/queen vs /artists/28)
- Improved error template fallback system

## Development Workflow

1. **Always write tests first** (Zone01 requirement)
2. **Use the unified repository pattern** (`internal/repository/repository.go`)
3. **Follow self-contained template pattern** (no inheritance)
4. **Test with audit data** (Queen, Gorillaz, Travis Scott)
5. **Check error template compatibility** (ErrorCode, ExtraCSS fields)

**File Reading Priority:**
1. `internal/repository/repository.go` (repository with all business logic)
2. `internal/handlers/handlers.go` (error handling patterns)
3. `templates/*.tmpl` (self-contained template examples)
4. Test files for current API usage patterns

**Testing Strategy:**
- All tests use audit-compliant data (Queen, Gorillaz, etc.)
- Test repository methods with mock data
- Verify template error handling with proper field structure
- Ensure no regression in Zone01 audit requirements
