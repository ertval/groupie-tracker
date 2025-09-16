# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with Go using clean architecture principles and comprehensive test coverage.

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- Implements responsive web design with self-contained HTML templates
- Provides robust error handling with proper HTTP status codes
- Features panic recovery middleware for server stability
- Achieves 77% test coverage with comprehensive unit and integration tests

## 🚀 Quick Start

### Prerequisites
- Go 1.19 or later
- Internet connection (for API data)

### Installation & Running
```bash
# Clone the repository
git clone <repository-url>
cd groupie-tracker

# Run the server
go run ./cmd/server/

# Or build and run
go build -o groupie-tracker ./cmd/server
./groupie-tracker

# Run tests
go test ./...

# Check coverage
go test -cover ./...
```

The server starts on **localhost:8080** by default. Set the `PORT` environment variable to use a different port.

## 🏗️ Architecture (December 2024)

### Clean Repository Pattern
The application follows a **repository pattern** with clear separation of concerns:

```
cmd/server/main.go           # Entry point with graceful shutdown
internal/
  ├── api/client.go         # External API consumption (1:1 API mapping)
  ├── data/data.go          # Repository pattern with all business logic
  └── handlers/handlers.go  # HTTP handlers with adapter pattern
templates/                  # Self-contained HTML templates
static/css/                 # Page-specific stylesheets
tests/                     # Audit compliance & E2E tests
```

### Key Design Patterns

#### Repository Pattern
```go
// Single data repository manages all business logic
repo := data.NewRepository()
apiClient := api.NewClient(url, timeout)

// Load data once at startup
adapter := &handlers.APIClientAdapter{Client: apiClient}
err := repo.InitializeWithAPI(ctx, adapter)

// All data access through repository methods
artists := repo.GetAllArtistsSorted()
artist, found := repo.GetArtistBySlug("queen")
```

#### Template System
- **Self-contained templates**: Each `.tmpl` file is a complete HTML document
- **No template inheritance**: Direct execution without complex hierarchy
- **Template functions**: `add`, `sub`, `join`, `generateLocationSlug`, `normalizeLocationName`

#### Error Handling
- **Centralized panic recovery**: Single middleware handles all panics
- **Proper HTTP status codes**: 404, 500, etc. with custom error pages
- **Graceful degradation**: Server never crashes, always returns valid responses

## 📊 API Data Structure

The application consumes four main API endpoints:

1. **Artists** (`/api/artists`) - Band/artist information:
   - Name, image, creation year, first album date, members

2. **Locations** (`/api/locations`) - Concert venues by location

3. **Dates** (`/api/dates`) - Concert dates (past and upcoming)

4. **Relations** (`/api/relation`) - Links between artists, locations, and dates

### Data Normalization
- Artists: Direct array from `/api/artists`
- Locations/Dates/Relations: Extract `.Index` field from `{"index": [...]}` format
- SEO slugs: Auto-generated for artists and locations (`/artists/queen`, `/locations/new-york-usa`)

## 🌐 Available Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page with featured artists |
| GET | `/artists` | All artists page |
| GET | `/artists/{slug}` | Artist detail page (SEO-friendly URLs) |
| GET | `/artists/{id}` | Artist detail by ID (legacy support) |
| GET | `/locations` | All locations with statistics |
| GET | `/locations/{slug}` | Location detail page |
| GET | `/healthz` | JSON health check endpoint |
| GET | `/static/*` | Static assets (CSS, images) |

### SEO-Friendly URLs
- `/artists/queen` instead of `/artists/28`
- `/locations/new-york-usa` instead of `/locations/1`
- Automatic slug generation from artist/location names

## 🧪 Testing & Quality

### Test Coverage: **77.1%**
- **Server tests**: 67.2% coverage (middleware, routing, integration)
- **API tests**: 86.2% coverage (HTTP client, data fetching)
- **Data tests**: 92.8% coverage (repository, business logic)
- **Handlers tests**: 64.8% coverage (HTTP handlers, error cases)
- **Audit tests**: Zone01 compliance verification

### Testing Strategy
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test suites
go test ./internal/data     # Repository tests
go test ./tests             # Audit compliance tests
go test ./cmd/server        # Server integration tests
```

### Audit Compliance
The application includes comprehensive audit tests that verify:
- **Queen**: Exactly 7 members
- **Gorillaz**: First album date "26-03-2001"
- **Travis Scott**: 10+ concert locations
- **Foo Fighters**: Exactly 6 members

## 🔧 Development

### Project Commands
```bash
# Development
go run ./cmd/server/          # Start development server
go test ./...                 # Run all tests
go test -cover ./...         # Coverage report

# Production
go build -o groupie-tracker ./cmd/server  # Build binary
./groupie-tracker                         # Run production server

# Testing
go test -v ./tests           # Verbose audit tests
go test -race ./...          # Race condition detection
```

### Environment Variables
- `PORT`: Server port (default: 8080)
- API URL is hardcoded to official Groupie Trackers API

### Adding New Features
1. **API changes**: Update `internal/api/client.go`
2. **Business logic**: Add methods to `internal/data/data.go`
3. **Web handlers**: Extend `internal/handlers/handlers.go`
4. **Templates**: Add new `.tmpl` files (self-contained)
5. **Tests**: Always write tests first (TDD approach)

## 📈 Performance Features

- **Single data load**: All API data loaded once at startup
- **Precomputed indexes**: SEO slugs, location stats calculated at initialization
- **Thread-safe operations**: Repository methods are concurrent-safe
- **Graceful shutdown**: Proper resource cleanup on server termination
- **Panic recovery**: Server never crashes from handler panics

## 🛡️ Error Handling

### HTTP Status Codes
- **200 OK**: Successful requests
- **404 Not Found**: Artist/location not found, invalid paths
- **405 Method Not Allowed**: Invalid HTTP methods
- **500 Internal Server Error**: Server errors with panic recovery

### Error Pages
- Custom error templates with consistent styling
- User-friendly error messages
- Proper HTTP status codes in headers

## 📝 Recent Updates (December 2024)

- ✅ **Fixed duplicate panic recovery**: Eliminated duplicate log messages
- ✅ **Comprehensive test coverage**: Achieved 77.1% overall coverage
- ✅ **Updated all tests**: Compatible with current project structure
- ✅ **Repository pattern**: Unified data access through single repository
- ✅ **Self-contained templates**: No template inheritance complexity
- ✅ **SEO-friendly URLs**: Artist and location slugs for better SEO
- ✅ **Graceful shutdown**: Proper server lifecycle management

## 🏆 Zone01 Compliance

This project meets all Zone01 educational requirements:
- **Standard library only**: No third-party dependencies
- **Test-driven development**: Tests written before implementation
- **Server stability**: Comprehensive panic recovery
- **Clean architecture**: Repository pattern with clear separation
- **Audit compliance**: All required test cases pass
- **Error handling**: Proper HTTP status codes and error pages

## 📄 License

This project is part of the Zone01 educational curriculum.