# Groupie Tracker

Groupie Tracker is a Go 1.24 web application that renders rich artist and concert data from the public Groupie Trackers API. The codebase follows a clean, server-side architecture with an immutable in-memory store, a small service façade, and HTTP handlers that render HTML templates—no JavaScript required.

## Highlights

- **Immutable data store**: `internal/data` loads API records concurrently, enriches them with derived metadata, and exposes read-only getters for artists, locations, statistics, and precomputed filter options.
- **Unified service layer**: Business logic consolidated directly into `data.Store` methods for filtering, search, and caching—eliminating unnecessary abstractions.
- **Web layer**: `internal/web` wires middleware, handlers, and templates. Every interaction (filters, search, pagination) posts HTML forms back to the server.
- **Concurrency built-in**: startup fetches artists and relations in parallel, derives indices concurrently, and optionally warms an image cache with an adaptive worker pool (scales with CPU cores).
- **Standard library only**: no third-party dependencies for the backend or templates.
- **Comprehensive testing**: 60.5% data layer coverage, 48.3% web layer coverage, with unit, E2E, and integration tests consolidated into package-level test files.

## Architecture Overview

```
cmd/server/main.go
   └── internal/app.Initialize()        // dependency injection + data load
         ├── internal/data.Store        // immutable data, concurrent loaders
         └── internal/service.Service   // business rules, caching, filtering
               └── internal/web.Server  // HTTP handlers + templates
```

### Data Layer (`internal/data`)

- Fetches artists and relations concurrently via `api.Client`.
- Normalises locations, builds SEO-friendly slugs, and precomputes:
  - artist indexes (`byID`, `bySlug`, `position`) for O(1) lookups
  - location aggregates with artist counts and concert stats
  - filter metadata (year bounds, member counts, country lists)
  - search suggestions with cached lowercase tokens
- Optionally caches artist images with a four-worker pool.

### Service Layer (`internal/service`)

- Wraps the immutable store without duplicating state.
- Provides `FilterArtists`, `FilterLocations`, and `SearchArtists` using precomputed metadata.
- Maintains a bounded, mutex-protected search result cache (50 entries) for plain-text queries.
- Supplies adjacency helpers (`GetAdjacentArtists`) leveraging the store’s index map.

### Web Layer (`internal/web`)

- `Server` compiles templates at startup and exposes typed handlers (`Home`, `Artists`, `Locations`, `Search`, detail pages, health endpoints).
- Middleware stack (`withRecovery → withLogging → withSecureHeaders`) wraps every request.
- Forms drive all interactivity; handlers parse form data into strongly typed filter/search params before calling the service.
- Rendering pipeline buffers template output to guarantee consistent error handling.

## Development Workflow

### Prerequisites

- Go 1.24.3+
- Internet access for the Groupie Trackers API

### Run the Server

```bash
cd groupie-tracker
go run ./cmd/server
```

The server listens on `:8082` by default; set `PORT=<value>` to override.

### Execute Tests

**Consolidated Test Structure (October 2025)**:

```bash
# Run all tests
go test ./...

# Unit tests by package
go test ./internal/data -v    # data layer (filters, search, cache)
go test ./internal/web -v     # web layer (handlers, middleware)

# E2E and integration tests
go test ./tests -run "TestE2E" -v          # end-to-end HTTP tests
go test ./tests -run "TestAudit" -v        # integration with external API

# Generate coverage report
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**Test Organization**:
- `internal/data/data_test.go`: Consolidated unit tests for filtering, search, and caching (previously `filter_test.go` and `search_test.go`)
- `internal/web/web_test.go`: Web layer tests (renamed from `server_test.go` for consistency)
- `tests/e2e_test.go`: End-to-end HTTP tests with mock API (consolidated from `cmd/server/e2e_test.go` and `search_e2e_test.go`)
- `tests/integration_test.go`: Integration tests with external API (consolidated from `audit_test.go`)
- `tests/playwright_test.go`: Browser automation tests (requires Playwright)
- `tests/visual_e2e_test.go`: Visual regression tests (requires running server)

**Current Coverage**: 60.5% data layer, 48.3% web layer

## Project Structure (abridged)

```
cmd/server/            # entry point (main.go)
doc/                   # design notes and refactor history
internal/
  api/                 # thin HTTP client for Groupie Trackers API
  config/              # runtime configuration (API URL, timeouts)
  data/                # immutable store with business logic
    store.go           # core store, data loading, indexes (548 LOC)
    filters.go         # filtering logic (266 LOC)
    searches.go        # search and suggestions (322 LOC)
    cache.go           # image and search caching (201 LOC)
    models.go          # domain types
    fixtures.go        # test fixtures
  web/                 # HTTP server, middleware, handlers, templates
    server.go          # server struct and initialization
    routes.go          # route configuration
    handlers.go        # all HTTP handlers (475 LOC, consolidated)
    templates.go       # template rendering helpers
    middleware.go      # logging, recovery, security headers
    errors.go          # error handling
    static.go          # static file serving
static/                # CSS and cached artist images
templates/             # HTML templates (base + pages)
tests/                 # E2E and integration tests
  e2e_test.go          # HTTP end-to-end tests
  integration_test.go  # external API integration
  playwright_test.go   # browser automation
  visual_e2e_test.go   # visual regression
```

## Key Features

- Creation/member/location filtering for artists.
- Concert-count filtering for locations.
- Full-text search across artist names, members, cities, countries, creation years, and first album dates.
- Precomputed search suggestions consumed server-side.
- Robust error handling with dedicated templates and health checks.

## Next Steps

See `doc/GitHub-Copilot-Refactoring-Plan.md` for the historical refactor plan and `REFACTORING_SUMMARY_OCT2025.md` for a detailed changelog.
