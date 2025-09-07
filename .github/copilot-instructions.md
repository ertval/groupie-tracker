# Groupie Tracker - AI Coding Agent Instructions

## Project Overview
This is a Zone01 educational project implementing a Go web application that consumes the Groupie Trackers API to display band/artist information with client-server interactions. The project follows strict TDD principles and Zone01 coding standards.

## Key Architecture Patterns

### Standard Library Only
- **CRITICAL**: Only Go standard library packages are allowed - no external dependencies
- Use `net/http` for server, `html/template` for templating, `encoding/json` for API calls
- All middleware, routing, and utilities are hand-crafted

### Project Structure Convention
```
cmd/server/main.go          # Entry point with graceful shutdown
internal/
  ├── api/client.go         # External API consumption
  ├── handlers/handlers.go  # HTTP request handlers  
  ├── models/models.go      # Core data structures
  ├── storage/store.go      # Thread-safe in-memory storage
  └── search/               # Search functionality (if implemented)
templates/                  # HTML templates with base.html pattern
static/                     # CSS/JS assets with specific naming
tests/audit_test.go         # Zone01 audit compliance tests
```

### Data Flow Architecture
1. **API Client** (`internal/api`) fetches from `https://groupietrackers.herokuapp.com/api`
2. **Store** (`internal/storage`) provides thread-safe in-memory cache with `sync.RWMutex`
3. **Handlers** (`internal/handlers`) serve both HTML pages and JSON APIs
4. **Templates** use Go's `html/template` with inheritance pattern (`base.html`)

## Critical Development Guidelines

### Test-Driven Development (Required)
- **Always write tests first** - this is a Zone01 requirement
- Test files must be `*_test.go` in same package as implementation
- Use specific test data: Queen (7 members), Gorillaz (first album: 26-03-2001), Travis Scott, Foo Fighters
- Integration tests in `tests/audit_test.go` validate against real API data

### Client-Server Events (Core Requirement)
The project MUST implement interactive events between client and server:
- **Search API**: `GET /api/search?q=query` for live search
- **Suggestions**: `GET /api/suggest?q=query` for autocomplete  
- **Data Refresh**: `POST /api/refresh` to reload from external API
- Frontend JavaScript with debouncing (300ms) and keyboard navigation

### Error Handling Patterns
```go
// Graceful degradation - server must never crash
func (h *Handlers) SomeHandler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Panic recovered: %v", err)
            h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
        }
    }()
    // handler logic
}
```

### Template System
## Copilot: quick onboarding for this repo

Purpose: give AI agents the minimal, actionable knowledge needed to be productive in this Go web app.

Key constraints
- Standard-library-only Go project — do not add third-party modules.
- Follow Test-Driven Development: write `*_test.go` in the same package before implementation.

Quick commands
- Run server: `go run ./cmd/server/` (defaults to PORT=8080)
- Run tests: `go test ./...`
- Coverage: `go test -cover ./...`
- Build: `go build -o groupie-tracker ./cmd/server`

High-level architecture (read these files first)
- Entry: `cmd/server/main.go`
- External API client: `internal/api/client.go` (handles inconsistent JSON shapes)
- Storage/cache: `internal/storage/store.go` (uses `sync.RWMutex`, must be thread-safe)
- HTTP handlers & templates: `internal/handlers/handlers.go`, `templates/` (base template + page blocks)
- Models: `internal/models/models.go`
- Tests & audits: `tests/` and `internal/*_test.go`

Important patterns and examples
- Data flow: API client -> in-memory store (lock/unlock) -> handlers -> templates/JSON handlers.
- Storage pattern: use `s.mu.RLock()`/`defer s.mu.RUnlock()` for reads and `s.mu.Lock()`/`defer s.mu.Unlock()` for writes.
- Template pattern: `templates/base.tmpl` selects page blocks by `.Title` (e.g. Title == "Artists" -> `artists-content`).
- Template funcs: `add`, `sub`, `contains` are used in templates; preserve their behavior when refactoring.

API quirks to handle (explicit)
- `/api/artists` returns a direct array.
- `/api/locations`, `/api/dates`, `/api/relation` return objects like `{"index": [...]}` — normalize in `internal/api/client.go`.

Core endpoints the agent may need to implement or test
- `GET /api/search?q=` (full search)
- `GET /api/suggest?q=` (autocomplete)
- `POST /api/refresh` (refresh cached data)

Zone01 audit checks (must be satisfied by tests)
- Queen must have 7 members.
- Gorillaz first album date == "26-03-2001".
- Travis Scott should show 10+ concert locations.
- Foo Fighters should show 6 members.

Developer notes for PRs
- Always include tests that reproduce the audit condition you are fixing.
- Preserve existing public function signatures where possible.
- Update `todo.md` and `doc/` when behavior or API shapes change.

If something is ambiguous, open `internal/api/client.go`, `internal/storage/store.go`, and `internal/handlers/handlers.go` — they contain the project-specific behaviors an agent must follow.

Please review this file and tell me which parts you'd like expanded (template loading, example tests, or common refactor-safe patterns).
