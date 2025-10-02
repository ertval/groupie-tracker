# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
Go web application (Go 1.24.3) consuming the Groupie Trackers API. **Zero JavaScript dependencies** - all filtering, search, and interactivity handled server-side via HTML forms and POST requests. **Standard library only** - no external dependencies.

## Architecture: Layered Server-Side Stack with Concurrent Loading

### Layer 1: `internal/api` - External API Client
- **Single responsibility**: Fetch raw JSON from external API
- Uses `api.Client` with dependency injection
- Raw models: `api.Artist`, `api.Relation` (match API JSON exactly)
- Entry point: `cmd/server/main.go` creates client → calls `app.Initialize()` to wire the store and service before starting the web server

### Layer 2: `internal/data` + `internal/service` - Data & Business Logic
- **`internal/data.Store`**
  - Immutable in-memory dataset after `Load()` completes
  - Concurrent API fetching using goroutines/channels (artists and relations in parallel)
  - Worker pool (4 workers) for concurrent image downloads
  - Precomputes indexes (ID, slug, position), filter metadata, search suggestions, statistics
  - Thread-safe read-only access after initialization
  - Files: `store.go`, `loader.go`, `fixtures.go`

- **`internal/service.Service`**
  - Thin business façade over the store
  - Provides filtering, search, adjacency helpers, and a bounded search result cache (50-entry LRU)
  - Operates on precomputed store metadata—no mutating operations
  - Files: `service.go`, `filtering.go`, `search.go`

### Layer 3: `internal/web` - HTTP Layer  
- `Server` struct holds both `store *data.Store` and `svc *service.Service`
- **Package-level handlers** (methods on `Server`): `Home()`, `Artists()`, `ArtistDetail()`, etc.
- Middleware chain: `withRecovery` → `withLogging` → `withSecureHeaders`
- Templates pre-compiled at startup in `Server.templates` map

## Critical Patterns

### 1. Concurrent Data Loading (Oct 2025 Update)
```go
// Store.loadData() - parallel API fetching
artistsCh := make(chan result, 1)
relationsCh := make(chan result, 1)

go func() { artistsCh <- fetchArtists() }()
go func() { relationsCh <- fetchRelations() }()

// Worker pool for image caching (4 workers)
jobs := make(chan job, len(artists))
var wg sync.WaitGroup
for w := 0; w < 4; w++ {
    wg.Add(1)
    go worker(jobs, &wg)
}
```
**Standard library only** - uses goroutines, channels, sync.WaitGroup, sync.Mutex

### 2. Server-Side Form Processing (Zero JavaScript)
```go
// ALL filters/search use POST + form parsing
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        filters := parseArtistFilterParams(r) // helper in templates.go
    artists = s.svc.FilterArtists(filters)
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
- Slug generation: `createSlug(name)` in `loader.go` - lowercase, hyphenated

### 5. Dependency Injection at Startup
```go
// main.go pattern
apiClient := api.NewClient(config.APIBaseURL, timeout)
server, err := web.NewServer(apiClient, config.WithCache)
server.ListenAndServe()
```
 `app.Initialize` receives `*api.Client`, returns `(store *data.Store, svc *service.Service)` for the web layer.

### 5. Caching Strategy
- **Pre-computed at startup**: `Server.suggestions`, `Server.artistFilterOpts`, `Server.locationFilterOpts`
- **Optional image cache**: Controlled by `config.WithCache` flag
- **Search result cache**: LRU-style map (50 entries max) in `Server.searchCache`

## Testing Conventions

### E2E Tests Location
- **Primary**: `cmd/server/e2e_test.go` and `search_e2e_test.go`
- Create mock API with `httptest.NewServer()`, inject via `api.NewClient(mockAPI.URL)`
- Helper: `createTestServerWithAPI(t, mockURL)` returns `*httptest.Server`

### Unit Tests
- Service layer tests: `internal/service/*_test.go` (filter/search coverage)
- Web: `internal/web/server_test.go` - use `server.Handler` with `httptest` (no network listener)

### Running Tests
```bash
go test ./...                           # All tests
go test ./internal/service -v          # Service layer only
go test ./cmd/server -run TestE2E       # E2E tests
go test -cover ./internal/service       # With coverage (target: 70%+)
```

## Common Tasks

### Adding a New Filter Parameter
1. Add field to `domain.ArtistFilterParams` or `domain.LocationFilterParams` in `models.go`
2. Implement logic in `matchesArtistFilters()` in `filtering.go`
3. Update `parseArtistFilterParams()` in `web/templates.go` to parse form field
4. Add HTML form controls in `templates/artists.tmpl` or `templates/locations.tmpl`
5. Add test cases in `internal/service/filter_test.go`

### Adding a New Page/Handler
1. Create handler method on `Server`: `func (s *Server) MyPage(w http.ResponseWriter, r *http.Request)`
2. Register route in `createServeMux()` in `routes.go`: `router.HandleFunc("/mypage", s.restrictMethod(s.MyPage, "GET"))`
3. Create template in `templates/mypage.tmpl` with `{{define "base"}}` and `{{define "title"}}`
4. Add CSS file in `static/css/mypage.css`, reference in template data: `ExtraCSS: "mypage.css"`
5. Use `s.render(w, r, "mypage.tmpl", data)` in handler

### Modifying Search Functionality
- Core search: `SearchArtists()` in `domain/search.go`
- Suggestions: `GenerateAllSearchSuggestions()` cached in `Server.suggestions`
- Search handler: `Search()` in `web/handlers.go` - handles both GET (display form) and POST (execute search)

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
- **Phase 0**: Created `internal/api` package, renamed `data`→`domain`, `server`→`web`
- **Phase 1**: Introduced immutable store with concurrent loading and fixture builders
- **Phase 2**: Concurrent data loading - parallel API fetching with goroutines, worker pool for image downloads (4 workers)
- **Phase 3**: Extracted dedicated service package and replaced handler calls with service APIs
- **Result**: Improved performance with concurrent loading, cleaner separation of concerns, all tests passing, standard library only

## Key Files Reference
- Entry point: `cmd/server/main.go`
- Dependency wiring: `internal/app/app.go`
- Routing: `internal/web/routes.go`
- HTTP server: `internal/web/server.go`
- Data storage: `internal/data/store.go` (immutable after Load)
- Data loading helpers: `internal/data/loader.go`
- Business logic: `internal/service/service.go`, `filtering.go`, `search.go`
- Handlers: `internal/web/home.go`, `artists.go`, `locations.go`, `search.go`
- Template utilities: `internal/web/templates.go`
- Base template: `templates/base.tmpl` (global search bar in navbar)
