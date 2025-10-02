# Architecture

## Overview

Groupie Tracker is a clean, idiomatic Go web application that follows a server-side rendering architecture with immutable data structures, concurrent data loading, and zero external dependencies. The application fetches artist and concert data from an external API, normalizes it into domain models, and serves it through HTML templates.

## Design Principles

1. **Simplicity First**: No unnecessary abstractions or over-engineering
2. **Immutable Data**: All data becomes read-only after initial load
3. **Concurrent Loading**: Parallel API fetching and index building
4. **Type Safety**: Strong typing throughout the codebase
5. **Zero Dependencies**: Standard library only (except test dependencies)
6. **Server-Side Rendering**: No JavaScript required for core functionality

## Package Structure

The application is organized into four main internal packages with clear responsibilities:

### `internal/api` - External API Client

**Purpose**: HTTP client for fetching data from the Groupie Trackers API.

**Responsibilities**:
- Fetch artist and relation data from external API
- Handle timeouts and retries
- Parse JSON responses into API-specific structs

**Dependencies**: Only standard library (`net/http`, `encoding/json`, `context`)

**Key Types**:
- `Client`: HTTP client with configurable timeouts
- `Artist`: Raw artist data from API
- `Relation`: Raw concert date/location mappings

### `internal/data` - Core Domain and Data Access

**Purpose**: Central data repository holding all normalized, indexed, and enriched application data.

**Responsibilities**:
- Load and normalize API data into domain models
- Build indexes and precompute metadata
- Provide filtered/searched access to data
- Optional image caching with adaptive worker pool

**Dependencies**: `internal/api` + standard library

**Key Components**:

1. **Store** (`store.go`):
   - Orchestrates data loading via `Load()` (thread-safe with `sync.Once`)
   - Delegates core data access to `Catalog`
   - Provides filtering, search, and caching operations
   - Precomputes metadata (statistics, filter options, search suggestions)

2. **Catalog** (`catalog.go`):
   - Normalizes API data into domain models
   - Builds O(1) lookup indexes (byID, bySlug, position)
   - Creates location aggregates with artist counts
   - Thread-safe concurrent read access

3. **Filters** (`filters.go`):
   - `ArtistFilterParams`: Creation year, album year, member count, countries
   - `LocationFilterParams`: Concert count, year range, countries
   - `FilterArtists()` and `FilterLocations()` methods

4. **Search** (`searches.go`):
   - Full-text search across artists, members, locations
   - `SearchArtists()` with combined query + filter support
   - `SearchSuggestions()` with prioritized results
   - `AdjacentArtists()` for previous/next navigation

5. **Cache** (`cache.go`):
   - Optional image caching with HTTP downloads
   - Adaptive worker pool (scales with CPU cores)
   - Bounded search result cache (50 entries)

**Key Types**:
- `Artist`: Domain model with helper methods (Countries(), ConcertCount(), Slug(), etc.)
- `Location`: Venue with aggregated concert data
- `Concert`: Normalized concert event (artist, location, date)

### `internal/view` - View Models

**Purpose**: Transform domain models into template-ready structures.

**Responsibilities**:
- Build page-specific view models
- Format data for template consumption
- Provide UI-specific computed properties

**Dependencies**: `internal/data` + standard library

**Key Components**:
- `BasePage`: Common page data (title, stats, filter options)
- `HomePage`, `ArtistsPage`, `ArtistDetailPage`: Page-specific view models
- `LocationsPage`, `LocationDetailPage`, `SearchPage`: More view models
- Builder functions: `NewHomePage()`, `NewArtistsPage()`, etc.

### `internal/web` - HTTP Layer

**Purpose**: HTTP server, routing, middleware, and template rendering.

**Responsibilities**:
- Initialize HTTP server with templates and middleware
- Handle HTTP requests and responses
- Parse form data into filter/search parameters
- Render HTML templates with view models
- Error handling and static file serving

**Dependencies**: `internal/api`, `internal/conf`, `internal/data`, `internal/view` + standard library

**Key Components**:

1. **Server** (`server.go`):
   - Initializes with `Store` and compiles templates
   - Exposes typed handler methods
   - Provides rendering utilities

2. **Routes** (`routes.go`):
   - Configures all HTTP routes
   - Applies middleware stack
   - Maps paths to handler methods

3. **Handlers** (`handlers.go`):
   - `Home()`, `Artists()`, `ArtistDetail()`
   - `Locations()`, `LocationDetail()`
   - `Search()`, `SuggestionsAPI()`
   - `Health()`, `StaticFiles()`

4. **Middleware** (`middleware.go`):
   - Recovery from panics
   - Request logging
   - Security headers

5. **Templates** (`templates.go`):
   - Template compilation and caching
   - Custom template functions (add, sub, slugToName, etc.)
   - Buffered rendering for error safety

## Data Flow

The application follows a clear data flow from API to rendered HTML:

```
1. API Layer
   ├── Fetch raw data (artists, relations)
   └── Parse JSON into API structs

2. Data Layer - Normalization
   ├── Transform API structs to domain models
   ├── Build indexes (byID, bySlug, position)
   └── Create location aggregates

3. Data Layer - Enrichment
   ├── Precompute statistics
   ├── Build search suggestions
   └── Generate filter metadata

4. Data Layer - Access
   ├── Filter artists/locations
   ├── Search across all fields
   └── Adjacent navigation

5. View Layer
   ├── Build page-specific view models
   └── Format for template consumption

6. Web Layer - Rendering
   ├── Handle HTTP request
   ├── Parse form parameters
   ├── Call data layer methods
   ├── Build view model
   └── Render HTML template
```

## Concurrency Model

### Startup Concurrency

**Phase 1: Parallel API Fetching**
```go
// Artists and relations fetched concurrently
go fetchArtists() // Goroutine 1
go fetchRelations() // Goroutine 2
// Wait for both to complete
```

**Phase 2: Parallel Metadata Computation**
```go
// Statistics, filters, and suggestions computed concurrently
go computeStats() // Goroutine 1
go computeFilters() // Goroutine 2
go computeSuggestions() // Goroutine 3
// Wait for all to complete
```

**Phase 3: Optional Image Caching**
```go
// Adaptive worker pool (4-8 workers based on CPU cores)
for each worker:
  go downloadImages(queue)
```

### Runtime Concurrency

- **Immutable Data**: All collections are read-only after `Load()` completes
- **No Locking**: Safe concurrent reads without mutexes on core data
- **Thread-Safe Loading**: `sync.Once` ensures single initialization
- **Bounded Cache**: Mutex-protected search result cache (50 entries max)

## Performance Characteristics

### Time Complexity

- **Artist lookup by ID**: O(1) - map index
- **Artist lookup by slug**: O(1) - map index
- **Filter operations**: O(n) - linear scan with early termination
- **Search operations**: O(n) - linear scan with normalized matching
- **Adjacent artists**: O(1) - map lookup + array access

### Space Complexity

- **Base data**: O(n) - artists + locations + concerts
- **Indexes**: O(n) - byID, bySlug, position maps
- **Metadata**: O(1) - statistics (constant size)
- **Filter options**: O(m) - countries list (small constant)
- **Suggestions**: O(n) - precomputed list
- **Caches**: O(1) - bounded sizes (50 search results, local images)

### Startup Performance

- **Target**: < 1 second total startup time
- **API fetches**: 200-500ms (parallel)
- **Normalization**: 50-100ms
- **Index building**: 10-50ms
- **Metadata computation**: 10-50ms (parallel)
- **Template compilation**: 10-20ms
- **Image caching**: Optional, runs in background

## Error Handling

### Strategy

1. **Graceful Degradation**: Continue with partial data if possible
2. **User-Friendly Messages**: Never expose internal errors
3. **Dedicated Templates**: Custom error pages (404, 405, 500)
4. **Recovery Middleware**: Catch panics and render 500 page
5. **Health Check**: `/health` endpoint for monitoring

### Error Types

- **404 Not Found**: Artist/location doesn't exist, invalid routes
- **405 Method Not Allowed**: Wrong HTTP method for route
- **500 Internal Server Error**: Unexpected errors, panics
- **503 Service Unavailable**: API fetch failures during startup

## Testing Strategy

### Test Organization

- **Package-level consolidation**: One test file per package
- **Table-driven tests**: Reusable test patterns
- **Mock API**: In-memory test data for unit tests
- **E2E tests**: Full HTTP request/response cycle
- **Integration tests**: Real external API calls

### Coverage Goals

- **Data layer**: 60%+ (filters, search, cache)
- **Web layer**: 50%+ (handlers, middleware)
- **Critical paths**: 90%+ (core business logic)

## Security

### Measures

1. **Path Traversal Prevention**: Validated file paths for static serving
2. **Security Headers**: CSP, X-Frame-Options, X-Content-Type-Options
3. **Input Validation**: Form parsing with bounds checking
4. **Error Information Hiding**: No stack traces to users
5. **Method Validation**: Only allowed HTTP methods per route

## Future Considerations

### Potential Improvements

1. **Caching Layer**: Redis/Memcached for distributed caching
2. **Database**: Persistent storage for offline mode
3. **API Rate Limiting**: Respect external API limits
4. **Pagination**: For large result sets
5. **API Endpoints**: JSON API for programmatic access

### Scaling Considerations

- **Horizontal**: Multiple instances with shared cache
- **Vertical**: Increase worker pool sizes
- **CDN**: Offload static assets and cached images
- **Database**: Move from in-memory to persistent storage

## References

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Library Documentation](https://pkg.go.dev/std)
