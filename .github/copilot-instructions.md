# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
Go web application consuming the Groupie Trackers API to display band/artist information. Built with strict TDD principles, Go standard library only, and audit requirements. **Features comprehensive server-side filtering and search functionality with HTML forms (no JavaScript dependencies).**

## Key Constraints & Commands

**Critical Constraints:**
- **Standard-library-only Go project** — NEVER add third-party modules (`go.mod` has no dependencies)
- **Test-Driven Development** — Write `*_test.go` files before implementation  
- **No JavaScript dependencies** — All filtering via server-side HTML forms with POST requests
- **No server crashes** — All handlers have panic recovery and proper error handling
- **Go 1.24.3 required** — Uses modern Go features

**Quick Commands:**
```bash
go run ./cmd/cli/             # Start server (streamlined entry point, default PORT=8080)
go test ./internal/data       # Run data tests (current coverage: 69.8%)
go test ./internal/...        # Run internal tests (some integration test issues being resolved)
go test ./tests/...           # Run audit/e2e tests (functional but with package issues)
go test -cover ./internal/data # Get detailed data package coverage report
go build -o groupie-tracker ./cmd/cli
```

## Current Architecture

### Clean Architecture with Server-Side Processing
```
cmd/cli/                     # Streamlined entry point with simple main.go
internal/
  ├── config/
  │   └── config.go          # Global config variables (no constructors needed)
  ├── data/                  # Core domain layer with filtering and search
  │   ├── repository.go      # Single data load with thread-safe read access
  │   ├── models.go          # Domain models + FilterParams/SearchParams  
  │   ├── filters.go         # Server-side filter logic (HTML form processing)
  │   ├── search.go          # Multi-type search with typed suggestions
  │   └── *_test.go          # Comprehensive test coverage
  └── server/                # HTTP layer
      ├── server.go          # Package-level initialization with global variables
      ├── handlers.go        # All endpoints as package-level functions
      ├── routes.go          # HTTP routing and middleware
      ├── middleware.go      # Panic recovery, logging, security
      └── utils.go           # Template utilities and helpers
templates/                   # Template inheritance with {{define "base"}} pattern
static/css/                  # Stylesheets (no JavaScript dependencies)
tests/                       # Audit compliance tests
```

### Critical Architecture Patterns

**Global State Pattern**: Repository and templates stored as package-level variables in server package
```go
var (
    repo      *data.Repository              // Global data access
    templates map[string]*template.Template // Pre-compiled templates
)
```

**Repository Pattern**: Load once, read many times (thread-safe after LoadData)
```go
repo := data.NewRepository()  // No constructor parameters - reads config internally
if err := repo.LoadData(ctx); err != nil { /* handle error */ }
artists := repo.GetArtists()             // Thread-safe read operations
filteredArtists := repo.FilterArtists(filterParams)  // Server-side filtering
```

**Configuration Pattern**: All config managed through `internal/config` package variables
```go
config.WithCache = false              // Image caching toggle
config.APIBaseURL = "https://..."     // Override in tests as needed
config.DefaultPort = ":8080"
```

## Server-Side Filter & Search System

### Filter Data Structures
```go
// HTML form parameters from POST /artists
type FilterParams struct {
    CreationYearFrom   *int     // Number input min value
    CreationYearTo     *int     // Number input max value
    FirstAlbumYearFrom *int     // Album year range filtering
    FirstAlbumYearTo   *int
    MemberCounts       []int    // Checkbox selections [1,2,3,4,5,6,7,8]
    Countries          []string // Checkbox selections from concert locations
}

// Pre-computed bounds for form validation
type FilterOptions struct {
    CreationYearMin   int      // Determines input min attribute
    CreationYearMax   int      // Determines input max attribute
    MemberCounts      []int    // Available checkbox options
    Countries         []string // Available country checkboxes
}
```

### Search Data Structures
```go
type SearchSuggestionType string
const (
    SuggestionTypeArtist     = "artist"      // Band/artist names
    SuggestionTypeMember     = "member"      // Band member names
    SuggestionTypeLocation   = "location"    // Concert locations
    SuggestionTypeFirstAlbum = "first-album" // Album dates
    SuggestionTypeCreation   = "creation"    // Formation years
)

type SearchSuggestion struct {
    Text        string               // Display text for UI
    Type        SearchSuggestionType // Category for grouping
    Description string               // Context hint
    URL         string               // Direct navigation link
}
```

### Critical Form Processing Pattern
```go
// POST /artists and POST /search - Server-side form processing
func Artists(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        // Parse FilterParams from HTML form data
        filterParams := parseFilterParams(r)  // Uses r.FormValue() internally
        filteredArtists := repo.FilterArtists(filterParams)
        // Render same template with filtered results
    }
    // GET request shows all artists with empty filter form
}
```

## Critical Data Flow Patterns

### API Data Normalization
```go
// API endpoint responses require different parsing:
// /api/artists → direct array: []Artist
// /api/locations → wrapped: {"index": [...]} - extract .Index field
// /api/dates → wrapped: {"index": [...]} - extract .Index field  
// /api/relation → wrapped: {"index": [...]} - extract .Index field
```

### Entry Point Pattern (`cmd/cli/main.go`)
```go
func main() {
    server, err := server.NewServer()  // Creates repo, loads data, sets up handlers
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }
    err = server.ListenAndServe()  // Blocking server start
}
```

### Error Template Pattern
```go
// Error handlers expect these specific field names:
data := struct {
    Title        string    // Page title
    ExtraCSS     string    // Additional CSS file
    ErrorCode    int       // HTTP status (NOT "Code")
    RequestedURL string    // Original request path  
    Message      string    // Error description
    Timestamp    string    // Formatted timestamp
}
```

### Template System
- Uses `{{define "base"}}` wrapper with `{{template "body" .}}` content blocks
- Template data uses inline struct patterns for type safety
- Custom functions: `hasField`, `contains`, `add`, `sub`, `join`
- Templates precompiled at startup, accessed via global `templates` variable

## Audit Requirements (Test Against These)

**Critical Data Points:**
- Queen: exactly 7 members
- Gorillaz: first album date "26-03-2001" 
- Travis Scott: 10+ concert locations
- Foo Fighters: exactly 6 members

**Required Endpoints:**
- `GET /` (home page)
- `GET /artists` (all artists with filter UI)
- `POST /artists` (filter form submission)
- `GET /artists/{slug}` (artist detail via SEO slug)
- `GET /locations` (all locations)
- `GET /locations/{slug}` (location detail)
- `GET /search` (search page with form)
- `POST /search` (search form submission)
- `GET /api/suggestions` (JSON search suggestions API)
- `GET /health` (JSON health check)

## Development Workflow

1. **Always write tests first** (audit requirement)  
2. **Use centralized config** (`internal/config` package for all settings)
3. **Follow template inheritance pattern** (base.tmpl with body blocks)
4. **Test with audit data** (Queen, Gorillaz, Travis Scott)
5. **Use inline struct patterns** for template data (type safety)

**File Reading Priority:**
1. `internal/data/repository.go` (core data management)
2. `internal/data/filters.go` (filter logic and bounds calculation)
3. `internal/data/search.go` (search functionality and suggestions)
4. `internal/config/config.go` (centralized configuration)
5. `internal/server/handlers.go` (HTTP layer patterns + form processing)
6. `internal/server/server.go` (startup and package-level initialization)
7. `templates/artists.tmpl` (filter UI components)
8. `templates/search.tmpl` (search interface and results)
9. Test files for current usage patterns

**Testing Strategy:**
- Use `go test ./internal/...` for clean test runs
- All tests use audit-compliant data (Queen=7 members, Gorillaz="26-03-2001")
- Test repository methods with mock data where needed  
- Test filter functionality with various parameter combinations
- Test search functionality across all data types (artists, members, locations, dates)
- Override config variables in tests rather than passing parameters
- Ensure server-side form processing works correctly
