git clone <repository-url>
go run ./cmd/cli/
go build -o groupie-tracker ./cmd/cli/
go test ./internal/... -v
go test ./internal/data/filter_test.go -v
go test ./tests/... -v
go test -cover ./internal/...
# Groupie Tracker

Groupie Tracker is a Go web application that presents artist and concert data from the public [Groupie Trackers API](https://groupietrackers.herokuapp.com/api). The project embraces a small, composable design: each package owns a narrow responsibility, dependencies are injected explicitly, and the HTTP layer is stateless and testable.

## Highlights

- End-to-end Go implementation with no runtime JavaScript requirements
- Deterministic data loading pipeline that normalises API payloads into rich domain models
- Search service supporting query + filter composition and reusable suggestions cache
- HTTP server organised around a `Server` struct (no globals) with structured middleware and type-safe template rendering
- Comprehensive unit and integration tests covering data transforms, search behaviour, and key handlers

## Architecture Overview

```
cmd/cli/            # CLI entrypoint wiring configuration, data store, search service, server

internal/
  api/              # Remote API client and DTO definitions
  config/           # Environment-driven runtime configuration (timeouts, ports, limits)
  data/             # Domain models, transformation pipeline, in-memory store implementation
  search/           # Search + filter service with suggestion generation and HTTP form helpers
  server/           # HTTP server, handlers, middleware, template manager
  testsupport/      # Shared fixtures and stub loaders for unit/integration tests

templates/          # HTML templates with base layout & page-specific views
static/             # Stylesheets, images, and other static assets
tests/              # Audit and exploratory test suites (optional/manual)
```

Each package exposes a small constructor (`api.NewClient`, `data.Load`, `search.NewService`, `server.New`) so dependencies can be composed in `cmd/cli/main.go` or within tests.

## Getting Started

### Prerequisites

- Go 1.24 or newer
- Internet access for the Groupie Trackers API

### Run the server

```bash
git clone <repository-url>
cd groupie-tracker

# start the HTTP server (defaults to http://localhost:8082)

```

### Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP listen address (use `:PORT`) | `:8082` |
| `GROUPIE_API_BASE_URL` | Override API endpoint | official public API |
| `GROUPIE_HTTP_TIMEOUT` | HTTP client timeout (e.g. `15s`) | `10s` |
| `GROUPIE_READ_TIMEOUT` / `GROUPIE_WRITE_TIMEOUT` / `GROUPIE_IDLE_TIMEOUT` | Server timeouts | `10s`, `10s`, `60s` |
| `GROUPIE_MAX_BODY_MB` | Request body size limit | `32` |

## Testing

The refactor includes fast, deterministic tests for every layer. Run everything with:

```bash
go test ./...
```

Key suites:

- `internal/data` – verifies the transformation pipeline, indexes, and statistics
- `internal/search` – exercises query matching, filter composition, and suggestions
- `internal/server` – spins up the HTTP server against the real templates to validate routes and JSON endpoints

Exploratory audit tests remain in `tests/`; they rely on manual server startup and are skipped automatically when the server is offline.

## Runtime Flow

1. **Configuration** is loaded from the environment (`config.FromEnv`).
2. **API client** (`api.NewClient`) downloads artists and relations, enforcing timeouts.
3. **Data store** (`data.Load`) materialises domain models, locations, and statistics with rich indexes.
4. **Search service** (`search.NewService`) precomputes suggestions and exposes reusable filtering helpers.
5. **Server** (`server.New`) wires dependencies into HTTP handlers, loads templates relative to the project root, and applies logging/security middleware.

## HTTP Surface

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/` | Homepage with featured artists and stats |
| GET/POST | `/artists` | Full artist catalogue with server-side filters |
| GET    | `/artists/{slug}` | Artist detail page with navigation links |
| GET    | `/locations` | Aggregated concert locations |
| GET    | `/locations/{slug}` | Location detail page |
| GET/POST | `/search` | Search UI combining free text and filters |
| GET    | `/api/suggestions` | JSON suggestions endpoint (`q` query parameter) |
| GET    | `/health` | JSON health probe exposing dataset statistics |

All templates share a base layout and live reload safely because they are compiled once at server construction time.

## Documentation

Additional background and the original refactoring goals are captured in `docs/`, including `docs/refactoring_plan_kiss.md` which motivated this simplification.

---

Enjoy exploring the codebase! The modules are intentionally lightweight—follow the constructors, read the tests, and you will get the full story quickly.
### Key Development Principles
1. **Test-Driven Development** - Always write tests before implementation
2. **Zero Dependencies** - Use only Go standard library (no JavaScript for filtering)
3. **Centralized Configuration** - All settings in `internal/config` package
4. **Template Inheritance** - Use `{{define "base"}}` with `{{template "body" .}}`
5. **Inline Data Structures** - Handler data structs defined inline for type safety
6. **Thread-Safe Operations** - Repository is read-only after initial data load
7. **Server-Side Processing** - All filter logic handled server-side via POST requests
8. **Native HTML Controls** - Use details/summary and form elements without JavaScript
9. **Graceful Error Handling** - Proper HTTP status codes and error pages
10. **Idiomatic Go** - Follow Go best practices and conventions

### Configuration Management
- All configuration variables (API URLs, timeouts, cache settings) are in `internal/config/config.go`

### Error Handling Pattern
- Centralized error handling middleware
- Graceful fallback to error templates
- Consistent JSON error responses

## ⚡ Current Status (September 2025)

### 🟢 Project Health
- **Data package tests passing** with 69.8% coverage (`go test ./internal/data`)
- **Search functionality operational**: Comprehensive search across all data types implemented
- **Filter functionality operational**: Complete server-side filtering for artists and locations
- **Zero-dependency project** - uses only Go standard library, no JavaScript
- **Core application stable** with graceful shutdown and error recovery
- **Audit requirements compliant** with required endpoints and data validation
- **Go 1.24.3** with modern Go features and clean architecture patterns

### 🔧 Current Development Notes
- **Integration Test Issues**: Some server integration tests experiencing template resolution issues
- **Core Functionality**: All main application features working correctly (filtering, search, routing)
- **Template System**: Base template inheritance working properly in main application
- **API Integration**: Successfully consuming Groupie Trackers API with proper data normalization

### 📊 Technical Metrics
- **Comprehensive search functionality** across artists, members, locations, dates without JavaScript
- **Real-time suggestions API** with typed categorization (artist, member, location, etc.)
- **Server-side filtering** for both artists and locations without JavaScript dependencies
- **Combined search + filters** for advanced queries with simultaneous criteria
- **CSS Grid responsive layout** with right sidebar filters that convert to top section on mobile
- **Native HTML controls** using details/summary for collapsible functionality
- **Thread-safe** read operations after initial data load with comprehensive filter/search logic
- **SEO-friendly URLs** with slug-based routing (`/artists/queen`)
- **Template inheritance** with base/body pattern for consistent UI across all pages

---

### Summary
The application integrates with the Groupie Trackers API to provide comprehensive artist information with advanced server-side search and filtering capabilities. The current implementation successfully handles the API's inconsistent response formats by normalizing data structures and building efficient search indexes. The repository loads all data once at startup, provides optional artist image caching, and ensures thread-safe read operations for concurrent requests. 

Handlers extract URL parameters, perform sophisticated search/filter operations, fetch data from the repository, and render HTML templates using a base layout with inheritance. The search functionality provides comprehensive coverage across all data types (artists, members, locations, dates) with real-time suggestions and seamless integration with the existing filter system.

**Architecture Highlights:**
- **Clean Architecture**: Clear separation between data, server, and presentation layers
- **Zero Dependencies**: Uses only Go 1.24.3 standard library
- **Server-Side Processing**: All filtering and search handled server-side with HTML forms
- **Thread-Safe Design**: Single data load with concurrent read access
- **Comprehensive Testing**: Test-driven development with extensive unit and integration tests
- **SEO-Friendly**: Slug-based URLs and semantic HTML structure
- **Responsive Design**: Mobile-first CSS with native HTML controls

**Built with ❤️ using Go 1.24.3 | Zero Dependencies | No JavaScript | Server-Side Filtering | Test-Driven Development | Idiomatic Go | Claude Sonnet**
