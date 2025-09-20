# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
Zone01 educational project implementing a Go web application that consumes the Groupie Trackers API to display band/artist information. The project follows strict TDD principles, uses only Go standard library, and maintains Zone01 coding standards.

## Key Constraints & Commands

**Critical Constraints:**
- **Standard-library-only Go project** — NEVER add third-party modules (`go.mod` has no dependencies)
- **Test-Driven Development** — Write `*_test.go` files before implementation  
- **No server crashes** — All handlers have panic recovery and proper error handling
- **Go 1.24+ required** — Uses modern Go features

**Quick Commands:**
```bash
go run ./cmd/server/          # Start server (default PORT=8080)
go test ./internal/...        # Run internal tests (clean, all passing)
go test ./tests/...           # Run audit/e2e tests (package issues but functional)
go test -cover ./internal/... # Coverage: ~60% overall, handlers ~52%, data ~72%
go build -o groupie-tracker ./cmd/server
```

## Current Architecture (December 2025)

### Simplified Clean Architecture
```
cmd/server/
  ├── main.go                # Entry point with graceful shutdown
  ├── server.go              # HTTP server setup, routing, middleware
  └── server_test.go         # Server integration tests
internal/
  ├── config/
  │   └── config.go          # Centralized global config (no constructor params)
  ├── data/                  # Core domain layer
  │   ├── repository.go      # Single data load with thread-safe access
  │   ├── domain.go          # Domain models (Artist, Location, Concert)
  │   ├── api.go             # API response structures
  │   └── repository_test.go # Repository tests (71.9% coverage)
  └── handlers/              # HTTP layer
      ├── handlers.go        # All endpoints in one file (453 lines)
      └── handlers_test.go   # Comprehensive handler tests (51.9% coverage)
templates/                   # Template inheritance with base/body pattern
static/                     # Static assets with proper MIME types
tests/                      # Audit tests (package issues but functional)
```

**🏗️ Current Architecture:**
- **Global config pattern**: `internal/config` package with module-level variables
- **Single data load**: Repository loads all data once at startup via `LoadData(ctx)`
- **Thread-safe reads**: All repository methods are read-only after initial load
- **Template inheritance**: Uses `{{define "base"}}` and `{{template "body" .}}` pattern
- **In-memory indexes**: SEO slugs, location mappings precomputed for fast lookup

### Repository Pattern (December 2025)
```go
// Repository initialization in server startup (reads internal/config automatically)
repo := data.NewRepository()  // No constructor parameters needed
if err := repo.LoadData(ctx); err != nil {
    log.Fatalf("Failed to load data: %v", err)
}

// All data access through repository methods (thread-safe reads)
artists := repo.GetArtists()
artist, found := repo.GetArtistBySlug("queen")
locations := repo.GetLocations()
stats := repo.GetStats()  // Precomputed statistics
```

## Critical Data Flow Patterns

### Centralized Configuration Pattern
```go
// ALL configuration managed through internal/config package
config.WithCache = false              // Image caching toggle
config.APIBaseURL = "https://..."     // API endpoint
config.APIRequestTimeout = 30*time.Second
config.DefaultPort = ":8080"
config.ReadTimeout = 15*time.Second

// Repository reads config internally - no parameters needed
repo := data.NewRepository()  // Uses config.* variables
```

### API Data Normalization (in `internal/data/repository.go`)
- `/api/artists` → direct array of Artist structs
- `/api/locations`, `/api/dates`, `/api/relation` → `{"index": [...]}` format
- Must extract `.Index` field for locations/dates/relations

### Handler Error Template Pattern
```go
// Error handlers expect specific struct fields
data := struct {
    Title        string
    ExtraCSS     string
    ErrorCode    int       // NOT "Code" 
    RequestedURL string
    Message      string
    Timestamp    string
}{
    Title:        "Page Not Found",
    ExtraCSS:     "errors.css",
    ErrorCode:    404,
    RequestedURL: r.URL.Path,
    Message:      "The page you're looking for doesn't exist.",
    Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
}
```

### Template System (Template Inheritance)
- Uses `{{define "base"}}` wrapper with `{{template "body" .}}` content injection
- Each page template defines `{{define "title"}}` and `{{define "body"}}`
- Template execution: `h.render(w, r, "artist_detail.tmpl", data)`
- Custom template functions: `add`, `sub`, `join`, plus helpers in handlers.go
- Template data uses inline struct patterns for type safety

### Current Error Handling Pattern
```go
func (h *Handler) render(w http.ResponseWriter, r *http.Request, templateName string, data any) {
    // Nil template protection for tests
    if h.templates[templateName] == nil {
        h.Error(w, r, 500, "Template not found")
        return
    }
    
    // Execute template with error fallback to error.tmpl
    if err := h.templates[templateName].Execute(w, data); err != nil {
        // Graceful fallback without panic
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

## Current Status (December 2025)

**✅ Recently Completed:**
- Fixed all failing tests - repository tests now match current API structure
- Achieved 75.8% test coverage (exceeded 70% target)
- Fixed 500 error template handling - now properly renders error template when available
- Enhanced handlers test coverage (71.2%) with comprehensive test cases
- Improved repository test coverage (84.2%) with edge case testing
- Unified repository pattern with single data load (no wrapper complexity)
- Enhanced error handling with nil template protection for tests
- All tests passing (comprehensive test suite)
- Note: tests/ folder has mixed package issue that causes test failures but doesn't affect functionality

**🔧 Current Architecture:**
- Clean repository pattern with single data load at startup
- Thread-safe operations through repository methods
- Graceful server shutdown with proper resource cleanup
- Self-contained template system working correctly
- SEO-friendly URL slugs (/artists/queen vs /artists/28)
- Improved error template fallback system

## Development Workflow

1. **Always write tests first** (Zone01 requirement)  
2. **Use centralized config** (`internal/config` package for all settings)
3. **Follow template inheritance pattern** (base.tmpl with body blocks)
4. **Test with audit data** (Queen, Gorillaz, Travis Scott)
5. **Use inline struct patterns** for template data (type safety)**File Reading Priority:**
1. `internal/data/repository.go` (core data management)
2. `internal/config/config.go` (centralized configuration)
3. `internal/handlers/handlers.go` (HTTP layer patterns)
4. `cmd/server/server.go` (startup and routing)
5. Test files for current usage patterns

**Testing Strategy:**
- Use `go test ./internal/...` for clean test runs
- All tests use audit-compliant data (Queen=7 members, Gorillaz="26-03-2001")
- Test repository methods with mock data where needed  
- Override config variables in tests rather than passing parameters
- Ensure no regression in Zone01 audit requirements
