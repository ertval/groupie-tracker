# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
Go web application (Go 1.24.3) consuming the Groupie Trackers API. **Zero JavaScript dependencies** - all filtering, search, and interactivity handled server-side via HTML forms and POST requests. **Standard library only** - no external dependencies.

## Architecture: 3-Layer Server-Side Stack (Simplified Oct 2025)

### Layer 1: `internal/api` - External API Client
- **Single responsibility**: Fetch raw JSON from external API
- Uses `api.Client` with dependency injection
- Raw models: `api.Artist`, `api.Relation` (match API JSON exactly)
- Entry point: `cmd/server/main.go` creates client → initializes Store directly

### Layer 2: `internal/data` - Unified Data & Business Logic Layer
- **`internal/data.Store`** - Single unified file (1,326 LOC)
  - Immutable in-memory dataset after `Load()` completes
  - Concurrent API fetching using goroutines/channels (artists and relations in parallel)
  - **Adaptive worker pool** for concurrent image downloads (scales with `runtime.NumCPU()`)
  - Precomputes indexes (ID, slug, position), filter metadata, search suggestions, statistics
  - Thread-safe read-only access after initialization
  - Contains ALL business logic: filtering, search with LRU cache (50 entries), data loading
  - Files: `store.go` (unified), `models.go`, `fixtures.go`

### Layer 3: `internal/web` - HTTP Layer  
- `Server` struct holds only `store *data.Store` (service layer eliminated)
- **Package-level handlers** (methods on `Server`): `Home()`, `Artists()`, `ArtistDetail()`, etc.
- Middleware chain: `withRecovery` → `withLogging` → `withSecureHeaders`
- Templates pre-compiled at startup in `Server.templates` map
- Files: `server.go`, `routes.go`, `pages.go`, `artists.go`, `locations.go`, `search.go`, `templates.go`, `middleware.go`, `errors.go`, `static.go`

## Critical Patterns

### 1. Adaptive Concurrent Loading (Oct 2025 Optimization)
```go
// Store.loadData() - parallel API fetching
artistsCh := make(chan result, 1)
relationsCh := make(chan result, 1)

go func() { artistsCh <- fetchArtists() }()
go func() { relationsCh <- fetchRelations() }()

// Adaptive worker pool for image caching (scales with CPU cores)
numWorkers := runtime.NumCPU()  // e.g., 12 workers on 12-core system
if numWorkers > len(artists) {
    numWorkers = len(artists)
}
for w := 0; w < numWorkers; w++ {
    wg.Add(1)
    go worker(jobs, &wg)
}
```
**Standard library only** - uses goroutines, channels, sync.WaitGroup, sync.Mutex, sync/atomic

### 2. Server-Side Form Processing (Zero JavaScript)
```go
// ALL filters/search use POST + form parsing
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        filters := parseArtistFilterParams(r) // helper in templates.go
        artists = s.store.FilterArtists(filters) // Direct store access
    }
    // render with results
}
```
**Never add JavaScript interactivity** - maintain HTML form submission pattern.

### 3. Template Rendering with Error Protection
```go
// Always use s.render() - never template.Execute() directly
s.render(w, r, "artists.tmpl", data) // executes to buffer first
```
Template helpers in `funcMap`: `add`, `sub`, `join`, `upper`, `title`, `contains`, `toSlice`

### 4. SEO-Friendly URL Slugs
- Artists: `/artists/queen` (not `/artists/28`)
- Locations: `/locations/london-uk` (not `/locations/42`)
- Slug generation: `createSlug(name)` in `store.go` - lowercase, hyphenated

### 5. Dependency Injection at Startup
```go
// main.go pattern - simplified (no app.Initialize)
apiClient := api.NewClient(config.APIBaseURL, timeout)
server, err := web.NewServer(apiClient, config.WithCache)
server.ListenAndServe()
```
`web.NewServer` receives `*api.Client`, creates `Store` directly, returns initialized server.

### 6. Caching Strategy
- **Pre-computed at startup**: `Store.suggestions`, `Store.artistFilterOpts`, `Store.locationFilterOpts`
- **Optional image cache**: Controlled by `config.WithCache` flag, adaptive worker pool
- **Search result cache**: LRU-style map (50 entries max) in `Store.searchCache`

## Testing Conventions

### E2E Tests Location
- **Primary**: `cmd/server/e2e_test.go` and `search_e2e_test.go`
- Create mock API with `httptest.NewServer()`, inject via `api.NewClient(mockAPI.URL)`
- Helper: `createTestServerWithAPI(t, mockURL)` returns `*httptest.Server`

### Unit Tests
- Data layer tests: `internal/data/*_test.go` (filter/search coverage)
- Web: `internal/web/server_test.go` - use `server.Handler` with `httptest` (no network listener)

### Running Tests
```bash
go test ./...                           # All tests
go test ./internal/data -v             # Data layer only
go test ./cmd/server -run TestE2E       # E2E tests
go test -cover ./internal/data          # With coverage (target: 70%+)
```

## Common Tasks

### Adding a New Filter Parameter
1. Add field to `ArtistFilterParams` or `LocationFilterParams` in `models.go`
2. Implement logic in `matchesArtistFilters()` in `store.go`
3. Update `parseArtistFilterParams()` in `web/templates.go` to parse form field
4. Add HTML form controls in `templates/artists.tmpl` or `templates/locations.tmpl`
5. Add test cases in `internal/data/filter_test.go`

### Adding a New Page/Handler
1. Create handler method on `Server`: `func (s *Server) MyPage(w http.ResponseWriter, r *http.Request)`
2. Register route in `createServeMux()` in `routes.go`: `router.HandleFunc("/mypage", s.restrictMethod(s.MyPage, "GET"))`
3. Create template in `templates/mypage.tmpl` with `{{define "base"}}` and `{{define "title"}}`
4. Add CSS file in `static/css/mypage.css`, reference in template data: `ExtraCSS: "mypage.css"`
5. Use `s.render(w, r, "mypage.tmpl", data)` in handler

### Modifying Search Functionality
- Core search: `SearchArtists()` in `store.go`
- Suggestions: `generateSearchSuggestions()` cached in `Store.suggestions`
- Search handler: `Search()` in `web/search.go` - handles both GET (display form) and POST (execute search)

## Configuration
All in `internal/config/config.go` as package-level vars:
- `config.WithCache` - Enable/disable image caching (default: false)
- `config.APIBaseURL` - External API endpoint
- `config.DefaultPort` - Server port (":8082")
- Timeouts: `ReadTimeout`, `WriteTimeout`, `IdleTimeout`

## Build & Run
```bash
go run ./cmd/server/              # Development server
go build -o groupie-tracker ./cmd/server/
./groupie-tracker                 # Production build
```

## Project Conventions
- **No empty interfaces**: Use concrete types (`Artist`, `Location`, not `interface{}`)
- **Package comments**: Keep minimal - code should be self-documenting
- **Error handling**: Use `fmt.Errorf("context: %w", err)` for wrapping
- **Sorting**: Default sort by `ConcertCount` (descending) for artists, `TotalConcerts` (descending) for locations
- **Template data**: Pass domain models directly - no wrapper structs (removed in Phase 2 refactoring)
- **Statistics**: Use type-safe `AppStats` struct, not `GetStats()` map (deprecated in Phase 3)

## Development Tools
- `/dev` - Development menu (only in dev builds)
- `/dev/panic` - Test panic recovery
- `/dev/404`, `/dev/500` - Test error pages
- `/health` - JSON health check endpoint

## Recent Refactoring Context (Oct 2025)
- **Phase 1 (Quick Wins)**: Deleted empty handlers.go, merged normalize.go into loader.go, consolidated small handler files into pages.go (-160 LOC)
- **Phase 2 (Service Elimination)**: Moved all filtering/search logic from service layer to Store, eliminated internal/service and internal/app packages entirely (-544 LOC)
- **Phase 3 (Data Consolidation)**: Merged loader.go into store.go to create single unified data file (1,326 LOC) with clear sections (+10 LOC)
- **Phase 4 (Concurrency Optimization)**: Changed worker pool from fixed 4 workers to adaptive `runtime.NumCPU()` (12 on typical systems), added 10-second HTTP timeout to prevent hanging (+6 LOC)
- **Result**: 3-layer architecture (down from 5), -688 total LOC (-23% reduction), improved concurrency (3x worker scaling), all tests passing, standard library only

## Key Files Reference
- Entry point: `cmd/server/main.go`
- Routing: `internal/web/routes.go`
- HTTP server: `internal/web/server.go`
- Data storage: `internal/data/store.go` (immutable after Load, unified 1,326 LOC file)
- Domain models: `internal/data/models.go`
- Handlers: `internal/web/pages.go`, `artists.go`, `locations.go`, `search.go`
- Template utilities: `internal/web/templates.go`
- Base template: `templates/base.tmpl` (global search bar in navbar)
