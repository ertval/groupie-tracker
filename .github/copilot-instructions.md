# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
An educational project implementing a Go web application that consumes the Groupie Trackers API to display band/artist information. The project follows strict TDD principles, uses only Go standard library, and maintains audit requirements. **Recently enhanced with comprehensive filter functionality including dual-range sliders and checkbox filters.**

## Key Constraints & Commands

**Critical Constraints:**
- **Standard-library-only Go project** — NEVER add third-party modules (`go.mod` has no dependencies)
- **Test-Driven Development** — Write `*_test.go` files before implementation  
- **No server crashes** — All handlers have panic recovery and proper error handling
- **Go 1.24+ required** — Uses modern Go features

**Quick Commands:**
```bash
go run ./cmd/cli/             # Start server (new entry point, default PORT=8080)
go test ./internal/...        # Run internal tests (clean, all passing)  
go test ./tests/...           # Run audit/e2e tests (package issues but functional)
go test -cover ./internal/... # Coverage: ~82% overall (data: 88.9%, handlers: 79.7%)
go build -o groupie-tracker ./cmd/cli
```

## Current Architecture (Updated January 2025)

### Clean Architecture with Filter Enhancement
```
cmd/cli/                     # New streamlined entry point
  ├── main.go                # Simple server startup
  └── e2e_test.go           # End-to-end integration tests
internal/
  ├── config/
  │   └── config.go          # Centralized global config (no constructor params)
  ├── data/                  # Core domain layer with filtering
  │   ├── repository.go      # Single data load with thread-safe access
  │   ├── models.go          # Domain models + FilterParams/FilterOptions
  │   ├── filters.go         # NEW: Filter logic with dual-range/checkbox support
  │   └── repository_test.go # Repository tests (88.9% coverage)
  └── server/                # HTTP layer (renamed from handlers/)
      ├── server.go          # App struct with server initialization
      ├── handlers.go        # All endpoints + JSON filter APIs (79.7% coverage)
      ├── routes.go          # HTTP routing and middleware setup
      ├── middleware.go      # Panic recovery, logging, security headers  
      ├── utils.go           # Template utilities and helper functions
      └── server_test.go     # Comprehensive handler tests
templates/                   # Template inheritance with filter UI components
static/
  ├── css/                  # Stylesheets including filter controls
  ├── js/filters.js         # NEW: Client-side filter interactions (361 lines)
  └── img/artists/          # Cached artist images
tests/                      # Audit tests (package issues but functional)
```

**🏗️ Current Architecture:**
- **Global config pattern**: `internal/config` package with module-level variables
- **Single data load**: Repository loads all data once at startup via `LoadData(ctx)`
- **Thread-safe reads**: All repository methods are read-only after initial load
- **Template inheritance**: Uses `{{define "base"}}` and `{{template "body" .}}` pattern
- **Filter system**: Dual-range sliders, checkbox grids, JSON API endpoints
- **In-memory indexes**: SEO slugs, location mappings, filter options precomputed

### Repository Pattern (January 2025)
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

// NEW: Filter functionality with dual-range sliders and checkboxes
filterOptions := repo.GetFilterOptions()  // Min/max bounds for sliders
filteredArtists := repo.FilterArtists(filterParams)  // Apply filter criteria
```

## Filter System Architecture (NEW - January 2025)

### Filter Data Structures (`internal/data/models.go`)
```go
// Filter parameters from client form submission
type FilterParams struct {
    CreationYearFrom    *int     `json:"creationYearFrom"`     // Range slider min
    CreationYearTo      *int     `json:"creationYearTo"`       // Range slider max
    FirstAlbumYearFrom  *int     `json:"firstAlbumYearFrom"`   // Range slider min
    FirstAlbumYearTo    *int     `json:"firstAlbumYearTo"`     // Range slider max  
    MemberCounts        []int    `json:"memberCounts"`         // Checkbox selections
    Countries           []string `json:"countries"`            // Checkbox selections
}

// Pre-computed filter bounds and options
type FilterOptions struct {
    CreationYearMin     int      `json:"creationYearMin"`      // Slider min bound
    CreationYearMax     int      `json:"creationYearMax"`      // Slider max bound  
    FirstAlbumYearMin   int      `json:"firstAlbumYearMin"`    // Slider min bound
    FirstAlbumYearMax   int      `json:"firstAlbumYearMax"`    // Slider max bound
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
    // Pre-computes min/max values for sliders, unique member counts, unique countries
}

// Helper: Extract country from location string "City, Country" format
func extractCountryFromLocation(location string) string

// Helper: Check if artist matches all active filter criteria
func matchesFilters(artist Artist, params FilterParams) bool
```

### Filter API Endpoints (`internal/server/handlers.go`)
```go
// POST /api/filter-artists - Apply filters and return matching artists as JSON
func (h *App) FilterArtists(w http.ResponseWriter, r *http.Request) {
    // Decodes FilterParams from JSON request body
    // Calls repo.FilterArtists() and returns filtered results as JSON
}

// GET /api/filter-options - Return filter bounds and available options as JSON  
func (h *App) FilterOptions(w http.ResponseWriter, r *http.Request) {
    // Returns FilterOptions struct with slider bounds and checkbox options
}
```

### Filter UI Components (`templates/artists.tmpl`)
```go
// Dual-range sliders for year filtering
<div class="range-slider">
    <input type="range" id="creation-year-from" name="creationYearFrom" 
           min="{{.FilterOptions.CreationYearMin}}" 
           max="{{.FilterOptions.CreationYearMax}}" 
           value="{{.FilterOptions.CreationYearMin}}" class="range-input range-from">
    <input type="range" id="creation-year-to" name="creationYearTo" 
           min="{{.FilterOptions.CreationYearMin}}" 
           max="{{.FilterOptions.CreationYearMax}}" 
           value="{{.FilterOptions.CreationYearMax}}" class="range-input range-to">
</div>

// Checkbox grids for discrete options
<div class="checkbox-grid" id="member-count-checkboxes">
    {{range .FilterOptions.MemberCounts}}
    <div class="checkbox-item">
        <input type="checkbox" id="members-{{.}}" name="memberCounts" value="{{.}}">
        <label for="members-{{.}}">{{.}} member{{if ne . 1}}s{{end}}</label>
    </div>
    {{end}}
</div>
```

### Filter JavaScript (`static/js/filters.js`)
- **361 lines** of client-side filter interaction logic
- **Dual-range slider synchronization** - prevents min > max values  
- **Real-time value updates** - displays current slider positions
- **AJAX form submission** - posts to `/api/filter-artists` endpoint
- **Dynamic DOM updates** - replaces artist grid with filtered results
- **Filter state management** - clear/reset functionality

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
- `GET /artists/{slug}` (artist detail via SEO slug)
- `GET /locations` (all locations)
- `GET /locations/{slug}` (location detail)
- `GET /health` (JSON health check)
- `POST /api/filter-artists` (JSON filter API)
- `GET /api/filter-options` (JSON filter bounds)

## Current Status (January 2025)

**✅ Recently Completed:**
- **Major Filter System Implementation** - dual-range sliders for year filtering, checkbox grids for member counts and countries
- **Restructured Architecture** - moved from cmd/server/ to cmd/cli/, renamed handlers/ to server/, added filters.go
- **Enhanced API Layer** - new JSON endpoints for filter functionality with proper error handling
- **Client-Side Interactions** - 361-line JavaScript implementation with AJAX form submission and dynamic DOM updates
- **Template Enhancement** - artists.tmpl updated with comprehensive filter UI components
- **Improved Test Coverage** - repository tests 88.9%, handlers 79.7%, overall ~82%
- **Thread-safe Operations** - all filter operations work with read-only repository after initial load

**🔧 Current Architecture:**
- Clean App struct pattern in server package with proper initialization
- Filter system as separate filters.go module with dedicated logic
- JSON API endpoints for client-server filter communication
- Dual-range slider synchronization preventing invalid min > max values
- Template inheritance system enhanced with filter components
- SEO-friendly URL slugs (/artists/queen vs /artists/28)
- Self-contained error handling with graceful template fallbacks

## Development Workflow

1. **Always write tests first** (audit requirement)  
2. **Use centralized config** (`internal/config` package for all settings)
3. **Follow template inheritance pattern** (base.tmpl with body blocks)
4. **Test with audit data** (Queen, Gorillaz, Travis Scott)
5. **Use inline struct patterns** for template data (type safety)**File Reading Priority:**
1. `internal/data/repository.go` (core data management)
2. `internal/data/filters.go` (NEW: filter logic and bounds calculation)
3. `internal/config/config.go` (centralized configuration)
4. `internal/server/handlers.go` (HTTP layer patterns + filter APIs)
5. `internal/server/server.go` (startup and App initialization)
6. `templates/artists.tmpl` (filter UI components)
7. `static/js/filters.js` (client-side filter interactions)
8. Test files for current usage patterns

**Testing Strategy:**
- Use `go test ./internal/...` for clean test runs
- All tests use audit-compliant data (Queen=7 members, Gorillaz="26-03-2001")
- Test repository methods with mock data where needed  
- Test filter functionality with various parameter combinations
- Override config variables in tests rather than passing parameters
- Ensure filter API endpoints return proper JSON responses
- Test dual-range slider edge cases (min=max, invalid ranges)
- Validate checkbox filter logic with multiple selections
- Ensure no regression in audit requirements
