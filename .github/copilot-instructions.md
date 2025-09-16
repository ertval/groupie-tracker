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

## Current Architecture (Updated December 2024)

### Repository Pattern - Clean & Simple
```
cmd/server/main.go           # Entry point with graceful shutdown
internal/
  ├── api/client.go         # External API consumption
  ├── data/data.go          # Repository pattern with all business logic (570+ lines)
  └── handlers/handlers.go  # HTTP handlers with adapter pattern
templates/                  # Self-contained HTML templates (NO inheritance)
static/css/                 # Page-specific stylesheets
tests/                     # Audit compliance & E2E tests
doc/                       # Architecture & refactoring documentation
```

**🏗️ Current Architecture:**
- Repository pattern: `data.Repository` manages all data operations
- APIClient adapter pattern: `handlers.APIClientAdapter` bridges api↔data layers
- Single initialization: Load data once at startup via `repo.InitializeWithAPI()`
- Precomputed indexes: SEO slugs, location stats calculated at load time

### Repository Pattern (December 2024)
```go
// Repository initialization in server startup
repo := data.NewRepository()
apiClient := api.NewClient(url, timeout)
if err := repo.InitializeWithAPI(ctx, adapter); err != nil {
    return nil, err
}

// All data access through repository methods
artists := repo.GetAllArtistsSorted()
artist, found := repo.GetArtistBySlug("queen")
locationStats := repo.CalculateLocationStats()
```

## Critical Data Flow Patterns

### API Data Normalization (in `internal/api/client.go`)
- `/api/artists` → direct array
- `/api/locations`, `/api/dates`, `/api/relation` → `{"index": [...]}` format
- Must extract `.Index` field for locations/dates/relations

### Handler Error Template Pattern
```go
// Error handlers expect specific struct fields
data := ErrorData{
    PageData: PageData{
        Title:    "Page Not Found",
        ExtraCSS: "errors.css",
    },
    ErrorCode:    404,        // NOT "Code" 
    RequestedURL: r.URL.Path,
    Message:      "The page you're looking for doesn't exist.",
}
```

### Template System (Self-Contained)
- Each `.tmpl` file is complete HTML document
- No template inheritance or `{{define "content"}}` blocks
- Direct execution: `h.templates.ExecuteTemplate(w, "artist_detail.tmpl", data)`
- Template functions: `add`, `sub`, `join`, `generateLocationSlug`, `normalizeLocationName`

### Data Initialization Pattern
```go
// Server startup pattern in cmd/server/server.go
repo := data.NewRepository()
apiClient := api.NewClient(DefaultAPIURL, RequestTimeout)

// Use adapter to bridge api ↔ data layers
adapter := &handlers.APIClientAdapter{Client: apiClient}
if err := repo.InitializeWithAPI(ctx, adapter); err != nil {
    return nil, fmt.Errorf("failed to initialize data: %w", err)
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

## Current Status (December 2024)

**✅ Recently Completed:**
- Fixed 404 error page display (ErrorCode vs Code mismatch)
- Removed "+X more" truncation in location artist lists
- Unified repository pattern with single data load (no more wrapper complexity)
- Comprehensive error handling with proper template compatibility
- All tests passing (46+ comprehensive tests)

**🔧 Current Architecture:**
- Clean repository pattern with single data load at startup
- Thread-safe operations through repository methods
- Graceful server shutdown with proper resource cleanup
- Self-contained template system working correctly
- SEO-friendly URL slugs (/artists/queen vs /artists/28)

## Development Workflow

1. **Always write tests first** (Zone01 requirement)
2. **Use the unified repository pattern** (`internal/data/data.go`)
3. **Follow self-contained template pattern** (no inheritance)
4. **Test with audit data** (Queen, Gorillaz, Travis Scott, Foo Fighters)
5. **Check error template compatibility** (ErrorCode, ExtraCSS fields)

**File Reading Priority:**
1. `internal/data/data.go` (repository with all business logic)
2. `internal/handlers/handlers.go` (error handling patterns)
3. `templates/*.tmpl` (self-contained template examples)
4. `doc/BUG_FIXES_AND_FEATURES_SUMMARY.md` (recent changes)

**Testing Strategy:**
- All tests use audit-compliant data (Queen, Gorillaz, etc.)
- Test repository methods with mock API client
- Verify template error handling with proper field structure
- Ensure no regression in Zone01 audit requirements
