# Groupie Tracker

Groupie Tracker is a Go 1.24 web application that renders rich artist and concert data from the public Groupie Trackers API. The codebase follows a clean, server-side architecture with an immutable in-memory store, a small service façade, and HTTP handlers that render HTML templates—no JavaScript required.

## Highlights

- **Immutable data store**: `internal/data` loads API records concurrently, enriches them with derived metadata, and exposes read-only getters for artists, locations, statistics, and precomputed filter options.
- **Service layer**: `internal/service` owns filtering, search, adjacency helpers, and a small LRU-style search cache implemented with the Go standard library.
- **Web layer**: `internal/web` wires middleware, handlers, and templates. Every interaction (filters, search, pagination) posts HTML forms back to the server.
- **Concurrency built-in**: startup fetches artists and relations in parallel, derives indices concurrently, and optionally warms an image cache with a worker pool.
- **Standard library only**: no third-party dependencies for the backend or templates.
- **Tested end-to-end**: unit tests cover service behaviours, while integration tests exercise the HTTP server with httptest.

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

```bash
go test ./...
```

Targeted suites:

```bash
go test ./internal/service -v    # service-layer filtering & search
go test ./cmd/server -run TestE2E # end-to-end smoke
```

## Project Structure (abridged)

```
cmd/server/            # entry point, integration tests
doc/                   # design notes and refactor history
internal/
  api/                 # thin HTTP client for Groupie Trackers API
  app/                 # dependency wiring helper (store + service)
  config/              # runtime configuration (API URL, timeouts)
  data/                # immutable store, loaders, derived metadata
  service/             # business logic (filters, search, caches)
  web/                 # HTTP server, middleware, handlers, templates
static/                # CSS and cached artist images
templates/             # HTML templates (base + pages)
```

## Key Features

- Creation/member/location filtering for artists.
- Concert-count filtering for locations.
- Full-text search across artist names, members, cities, countries, creation years, and first album dates.
- Precomputed search suggestions consumed server-side.
- Robust error handling with dedicated templates and health checks.

## Next Steps

See `doc/GitHub-Copilot-Refactoring-Plan.md` for the historical refactor plan and `REFACTORING_SUMMARY_OCT2025.md` for a detailed changelog.
