# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
An educational Go web application consuming the Groupie Trackers API to display band/artist information. Built with strict TDD principles, Go standard library only, and audit requirements. **Features comprehensive server-side filtering with HTML forms (no JavaScript dependencies).**

## Key Constraints & Commands

**Critical Constraints:**
- **Standard-library-only Go project** — NEVER add third-party modules (`go.mod` has no dependencies)
- **Test-Driven Development** — Write `*_test.go` files before implementation  
- **No JavaScript dependencies** — All filtering via server-side HTML forms with POST requests
- **No server crashes** — All handlers have panic recovery and proper error handling
- **Go 1.24+ required** — Uses modern Go features

**Quick Commands:**
```bash
go run ./cmd/cli/             # Start server (streamlined entry point, default PORT=8080)
go test ./internal/...        # Run internal tests (clean, all passing)  
go test ./tests/...           # Run audit/e2e tests (functional but with package issues)
go test -cover ./internal/... # Coverage: ~82% overall (data: 88.9%, handlers: 79.7%)
go build -o groupie-tracker ./cmd/cli
```

## Current Architecture (Updated September 2025)

### Clean Architecture with Server-Side Filtering
```
cmd/cli/                     # Streamlined entry point
  ├── main.go                # Simple server startup
  └── e2e_test.go           # End-to-end integration tests
internal/
  ├── config/
  │   └── config.go          # Centralized global config (no constructor params)
  ├── data/                  # Core domain layer with filtering
  │   ├── repository.go      # Single data load with thread-safe access
  │   ├── models.go          # Domain models + FilterParams/FilterOptions  
  │   ├── filters.go         # Server-side filter logic (HTML form processing)
  │   ├── filter_test.go     # Filter unit tests (18 tests)
  │   └── repository_test.go # Repository tests (88.9% coverage)
  └── server/                # HTTP layer (renamed from handlers/)
      ├── server.go          # Package-level server initialization with global variables
      ├── handlers.go        # All endpoints + filter form processing (package-level functions)
      ├── routes.go          # HTTP routing and middleware setup
      ├── middleware.go      # Panic recovery, logging, security headers  
      ├── utils.go           # Template utilities and helper functions
      └── server_test.go     # Comprehensive handler tests
templates/                   # Template inheritance with filter UI components
static/
  ├── css/                  # Stylesheets including filter controls (no JavaScript)
  └── img/artists/          # Cached artist images
tests/                      # Audit tests (functional but with package issues)
```

**🏗️ Current Architecture:**
- **Global config pattern**: `internal/config` package with module-level variables
- **Package-level server functions**: Removed App struct, using package-level variables (repo, templates)
- **Single data load**: Repository loads all data once at startup via `LoadData(ctx)`
- **Thread-safe reads**: All repository methods are read-only after initial load
- **Template inheritance**: Uses `{{define "base"}}` and `{{template "body" .}}` pattern
- **Server-side filtering**: HTML forms with POST to `/artists` (no JavaScript/AJAX)
- **In-memory indexes**: SEO slugs, location mappings, filter options precomputed

### Repository Pattern (September 2025)
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

// Server-side filtering with HTML form data processing
filterOptions := repo.GetFilterOptions()  // Min/max bounds for form validation
filteredArtists := repo.FilterArtists(filterParams)  // Apply filter criteria from form
```

### Package-Level Server Pattern (Updated September 2025)
```go
// Package-level variables (follows global config pattern)
var (
    repo      *data.Repository
    templates map[string]*template.Template
)

// Server initialization assigns to package-level variables
func NewServer() (*http.Server, error) {
    repo = data.NewRepository()  // Assigns to package-level variable
    if err := repo.LoadData(ctx); err != nil {
        return nil, fmt.Errorf("failed to load data: %w", err)
    }
    loadTemplates()  // Assigns to package-level templates variable
    serveMux := withMiddleware(routes())  // Uses package-level functions
    return &http.Server{Addr: port, Handler: serveMux}, nil
}

// All handlers are package-level functions using global variables
func Home(w http.ResponseWriter, r *http.Request) {
    artists := repo.GetArtists()  // Uses package-level repo
    render(w, r, "home.tmpl", data)  // Uses package-level templates
}

func routes() *http.ServeMux {
    mux := http.NewServeMux()
    mux.HandleFunc("/", Home)  // References package-level function
    mux.HandleFunc("/artists", Artists)  // No receiver needed
    return mux
}
```

## Filter System Architecture (Server-Side HTML Forms)

### Filter Data Structures (`internal/data/models.go`)
```go
// Filter parameters from HTML form submission
type FilterParams struct {
    CreationYearFrom    *int     `json:"creationYearFrom"`     // Number input min
    CreationYearTo      *int     `json:"creationYearTo"`       // Number input max
    FirstAlbumYearFrom  *int     `json:"firstAlbumYearFrom"`   // Number input min
    FirstAlbumYearTo    *int     `json:"firstAlbumYearTo"`     // Number input max  
    MemberCounts        []int    `json:"memberCounts"`         // Checkbox selections
    Countries           []string `json:"countries"`            // Checkbox selections
}

// Pre-computed filter bounds and options
type FilterOptions struct {
    CreationYearMin     int      `json:"creationYearMin"`      // Input min bound
    CreationYearMax     int      `json:"creationYearMax"`      // Input max bound  
    FirstAlbumYearMin   int      `json:"firstAlbumYearMin"`    // Input min bound
    FirstAlbumYearMax   int      `json:"firstAlbumYearMax"`    // Input max bound
    MemberCounts        []int    `json:"memberCounts"`         // Available options
    Countries           []string `json:"countries"`            // Available options
}
```

### Filter Logic (`internal/data/filters.go`)
```go
// Main filter method - applies all filter criteria
func (r *Repository) FilterArtists(params FilterParams) []Artist {
    // Multi-criteria filtering with range checks and checkbox matching
    // Implements creation year range, first album year range, member count, countries
}

// Extract filter bounds from loaded data  
func (r *Repository) GetFilterOptions() FilterOptions {
    // Pre-computes min/max values for inputs, unique member counts, unique countries
}

// Helper: Extract country from location string "City, Country" format
func extractCountryFromLocation(location string) string

// Helper: Check if artist matches all active filter criteria
func matchesFilters(artist Artist, params FilterParams) bool
```

### Filter Form Processing (`internal/server/handlers.go`)
```go
// POST/GET /artists - Handle filter form submission and display
func Artists(w http.ResponseWriter, r *http.Request) {
    // Parses FilterParams from HTML form data (POST) or shows all artists (GET)
    // Calls repo.FilterArtists() and renders filtered results in same template
    // No JSON API endpoints - all server-side rendering
}
```

### Filter UI Components (`templates/artists.tmpl`)
```go
// HTML form with server-side submission
<form method="POST" action="/artists">
    <!-- Number inputs for year ranges -->
    <input type="number" name="creationYearFrom" 
           min="{{.FilterOptions.CreationYearMin}}" 
           max="{{.FilterOptions.CreationYearMax}}" 
           value="{{if .FilterParams.CreationYearFrom}}{{.FilterParams.CreationYearFrom}}{{end}}">
    
    <!-- Checkbox grids for discrete options -->
    <div class="checkbox-grid">
        {{range .FilterOptions.MemberCounts}}
        <input type="checkbox" name="memberCounts" value="{{.}}"
               {{if contains $.FilterParams.MemberCounts .}}checked{{end}}>
        {{end}}
    </div>
</form>
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

### New Entry Point Pattern (`cmd/cli/main.go`)
```go
// Simplified startup - server handles all initialization
func main() {
    log.Println("Starting Groupie Tracker server...")
    
    server, err := server.NewServer()  // Creates repo, loads data, sets up handlers
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }
    
    err = server.ListenAndServe()  // Blocking server start
    if err != nil && err != http.ErrServerClosed {
        log.Fatalf("Server failed: %v", err)
    }
}
```

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
func (h *App) render(w http.ResponseWriter, r *http.Request, templateName string, data any) {
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

## Audit Requirements (Test Against These)

**Critical Data Points:**
- Queen: exactly 7 members
- Gorillaz: first album date "26-03-2001" 
- Travis Scott: 10+ concert locations
- Foo Fighters: exactly 6 members

**Required Endpoints:**
- `GET /` (home page)
- `GET /artists` (all artists with filter UI)
- `POST /artists` (filter form submission - server-side processing)
- `GET /artists/{slug}` (artist detail via SEO slug)
- `GET /locations` (all locations)
- `GET /locations/{slug}` (location detail)
- `GET /health` (JSON health check)

## Current Status (September 2025)

**✅ Recently Completed:**
- **Major Filter System Implementation** - server-side form processing for year filtering, checkbox grids for member counts and countries
- **Restructured Architecture** - moved from cmd/server/ to cmd/cli/, renamed handlers/ to server/, added filters.go
- **Enhanced Server Layer** - form processing endpoints with proper error handling and validation
- **Template Enhancement** - artists.tmpl updated with comprehensive filter UI components using native HTML controls
- **Improved Test Coverage** - repository tests 88.9%, handlers 79.7%, overall ~82%
- **Thread-safe Operations** - all filter operations work with read-only repository after initial load

**🔧 Current Architecture:**
- Package-level server functions pattern instead of App struct 
- Repository and templates stored as global variables following config pattern
- Filter system as separate filters.go module with dedicated logic
- Server-side form processing with POST requests to `/artists`
- HTML form controls (number inputs, checkboxes) for filtering
- Template inheritance system enhanced with filter components
- SEO-friendly URL slugs (/artists/queen vs /artists/28)
- Self-contained error handling with graceful template fallbacks

## Development Workflow

1. **Always write tests first** (audit requirement)  
2. **Use centralized config** (`internal/config` package for all settings)
3. **Follow template inheritance pattern** (base.tmpl with body blocks)
4. **Test with audit data** (Queen, Gorillaz, Travis Scott)
5. **Use inline struct patterns** for template data (type safety)

**File Reading Priority:**
1. `internal/data/repository.go` (core data management)
2. `internal/data/filters.go` (filter logic and bounds calculation)
3. `internal/config/config.go` (centralized configuration)
4. `internal/server/handlers.go` (HTTP layer patterns + filter form processing)
5. `internal/server/server.go` (startup and package-level initialization)
6. `templates/artists.tmpl` (filter UI components)
7. Test files for current usage patterns

**Testing Strategy:**
- Use `go test ./internal/...` for clean test runs
- All tests use audit-compliant data (Queen=7 members, Gorillaz="26-03-2001")
- Test repository methods with mock data where needed  
- Test filter functionality with various parameter combinations
- Override config variables in tests rather than passing parameters
- Ensure server-side form processing works correctly
- Test filter UI with different parameter combinations
- Validate checkbox filter logic with multiple selections
- Ensure no regression in audit requirements
