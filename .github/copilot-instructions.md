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
- Use `base.html` with `{{template "content" .}}` pattern
- Load templates in `handlers.go` with fallback HTML for resilience
- All pages must work even if templates fail to load

## Development Workflow

### Running & Testing
```bash
# Start server
go run cmd/server/main.go

# Run all tests (must pass before any commit)
go test ./...

# Run with coverage
go test -cover ./...

# Build for production
go build -o groupie-tracker cmd/server/main.go
```

### Adding New Features
1. **Write test first** in appropriate `*_test.go` file
2. **Implement minimum code** to make test pass
3. **Update `todo.md`** with current status
4. **Test against audit requirements** before considering complete

### API Integration
The external API has specific response structures:
- `/api/artists` - direct array
- `/api/locations` - wrapped in `{"index": [...]}` 
- `/api/dates` - wrapped in `{"index": [...]}`
- `/api/relation` - wrapped in `{"index": [...]}`

Handle these inconsistencies in `internal/api/client.go`.

## Common Patterns & Conventions

### Storage Layer
```go
// Thread-safe operations required
s.mu.RLock()
defer s.mu.RUnlock()
// Read operations

s.mu.Lock()  
defer s.mu.Unlock()
// Write operations
```

### Handler Structure
```go
func (h *Handlers) ExampleHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGET {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // handler logic with proper error handling
}
```

### Frontend Integration
- JavaScript lives in `static/js/main.js` with ES6+ features
- CSS animations and responsive design in `static/css/main.css`
- No external frontend libraries - vanilla JavaScript only

## Zone01 Audit Requirements
The final implementation must pass audit tests that verify:
- Queen has exactly 7 members (including Freddie Mercury)
- Gorillaz first album date is "26-03-2001"  
- Travis Scott concert locations (10+ venues)
- Foo Fighters has 6 current members
- Client-server interaction works (live search, suggestions, refresh)
- Server stability (no crashes, proper error handling)

## Performance & Quality Standards
- All tests must complete in <5 seconds
- Server startup in <2 seconds with data loaded
- Thread-safe concurrent operations
- Memory-efficient data structures
- Proper HTTP status codes and Content-Type headers
- Graceful shutdown handling SIGTERM/SIGINT

When implementing features, prioritize Zone01 audit compliance and TDD practices over code elegance.
