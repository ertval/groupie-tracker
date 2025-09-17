# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with Go using clean architecture principles and comprehensive test coverage.

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- Implements responsive web design with self-contained HTML templates
- Provides robust error handling with proper HTTP status codes
- Features panic recovery middleware for server stability
- Achieves **75.8%** test coverage with comprehensive unit and integration tests

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

## 🏗️ Current Architecture (September 2025)

### Clean Repository Pattern
The application follows idiomatic Go patterns with clean architecture:

```
cmd/server/main.go           # Entry point with graceful shutdown
cmd/server/server.go         # Server configuration and middleware
internal/
  └── repository/            # Core data management
      ├── repository.go      # Complete repository with all functionality
      └── repository_test.go # Comprehensive test coverage (80.4%)
  └── handlers/              # HTTP request handlers  
      ├── handlers.go        # All HTTP endpoints and template rendering
      └── handlers_test.go   # Handler tests (71.2% coverage)
templates/                   # Self-contained HTML templates
static/css/                  # Page-specific stylesheets
tests/                      # End-to-end and audit tests
```

### Repository Design

The `repository` package provides all data management functionality:

```go
// Create repository with API connection
repo := repository.NewRepository("https://groupietrackers.herokuapp.com", 30*time.Second)

// Load data once from API endpoints
err := repo.LoadData(ctx)

// Access data through repository methods
artists := repo.GetArtists()                    // All artists sorted by name
artist, found := repo.GetArtistBySlug("queen")  // Artist by SEO slug
locationStats := repo.GetLocationStats()        // All locations with statistics
stats := repo.GetStats()                        // Global statistics
```

### Key Data Structures

#### Core Models
- **`Artist`**: Musical artist with concerts, SEO slug, and complete info
- **`LocationStats`**: Location with artists that performed there
- **`ComputedData`**: Internal processed data with indexes for efficiency

#### Repository Features
- **Single data load**: Efficient startup with one API call per endpoint
- **Precomputed indexes**: SEO slugs, location stats calculated at startup
- **Thread-safe access**: Repository methods support concurrent requests
- **Memory efficient**: No data duplication, single source of truth

## 📊 API Integration

Uses all 4 Groupie Trackers API endpoints efficiently:

1. **`/api/artists`** - Basic artist information (name, members, creation year, etc.)
2. **`/api/locations`** - Available concert locations  
3. **`/api/dates`** - Concert dates information
4. **`/api/relation`** - Artist-location-date relationships (primary data source)

### Data Processing Flow
1. Fetch from API endpoints in parallel
2. Process into domain models with SEO slugs
3. Compute location statistics with concert dates per artist
4. Generate global statistics for dashboard
5. Create efficient lookup indexes for fast access

## 🌐 Available Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page with featured artists |
| GET | `/artists` | All artists listing page |
| GET | `/artists/{slug}` | Artist detail page (SEO-friendly URLs) |
| GET | `/locations` | All locations with statistics |
| GET | `/locations/{slug}` | Location detail with concert dates |
| GET | `/health` | JSON health check endpoint |
| GET | `/static/*` | Static assets (CSS, images) |

### SEO-Friendly URLs
- `/artists/queen` instead of `/artists/28`
- `/locations/new-york-usa` instead of `/locations/1`
- Automatic slug generation from artist/location names

## 🧪 Testing & Quality

### Test Coverage: **75.8%**
- **Repository package**: 80.4% coverage (data management, business logic)
- **Handlers package**: 71.2% coverage (HTTP handlers, error cases)  
- **Overall internal packages**: 75.8% coverage
- **Audit tests**: Zone01 compliance verification

### Testing Strategy
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test suites
go test ./internal/repository  # Repository tests
go test ./internal/handlers    # Handler tests
go test ./tests               # Audit compliance tests
go test ./cmd/server          # Server integration tests

# Generate detailed coverage report
go test -coverprofile=coverage.out ./internal/... && go tool cover -html=coverage.out
```

### Audit Compliance
The application includes comprehensive audit tests that verify:
- **Queen**: Exactly 7 members
- **Gorillaz**: First album date "26-03-2001"
- **Travis Scott**: Multiple concert locations
- **All API endpoints**: Proper consumption of all 4 endpoints

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

## 🔧 Development

### Environment Variables
- `PORT`: Server port (default: 8080)
- API URL configured for official Groupie Trackers API

### Project Structure Best Practices
1. **Repository first**: All data logic in `internal/repository/repository.go`
2. **Handler simplicity**: Minimal business logic in handlers 
3. **Self-contained templates**: Each template is complete HTML
4. **Comprehensive testing**: Test-driven development approach
5. **Clean architecture**: Clear separation of concerns

## 📈 Performance & Reliability

- **Single data load**: Efficient startup with one API call per endpoint
- **Precomputed data**: Statistics and indexes calculated once
- **Thread-safe access**: Repository methods support concurrent requests  
- **Memory efficient**: No data duplication across structures
- **Panic recovery**: Server stability with graceful error handling
- **Graceful shutdown**: Clean resource management on termination

## 🛡️ Error Handling

### HTTP Status Codes
- **200 OK**: Successful requests with valid data
- **404 Not Found**: Artist/location not found, invalid paths  
- **405 Method Not Allowed**: Invalid HTTP methods
- **500 Internal Server Error**: Server errors with panic recovery

### Error Pages
- Custom error templates maintaining site design consistency
- User-friendly error messages with navigation options
- Proper HTTP status codes and headers

## 📝 Recent Updates (September 2025)

### Current Status (September 17, 2025)
- ✅ **All tests passing**: Fixed repository tests to match current API
- ✅ **75.8% test coverage**: Exceeded 70% coverage target
- ✅ **500 error template fix**: Proper error template rendering when templates fail
- ✅ **Clean repository pattern**: Single repository with all data management
- ✅ **Idiomatic Go**: Following Go best practices and patterns
- ✅ **Documentation updated**: Accurate reflection of current architecture
- ✅ **Template system**: Self-contained templates with proper error handling

### Architecture Improvements
- **Single repository**: All data logic consolidated in `internal/repository/repository.go`
- **API integration**: Proper consumption of all 4 Groupie Trackers endpoints
- **Error handling**: Improved 500 error handling with fallback to templates
- **Thread safety**: Repository designed for concurrent access
- **Template rendering**: Enhanced error handling in template execution

## 🎯 Zone01 Audit Compliance

- **Queen**: ✅ Exactly 7 members displayed
- **Gorillaz**: ✅ First album "26-03-2001" correctly shown  
- **Travis Scott**: ✅ 10+ concert locations verified
- **Foo Fighters**: ✅ Exactly 6 members confirmed
- **API endpoints**: ✅ All 4 endpoints properly consumed
- **SEO URLs**: ✅ Artist and location slugs implemented
- **Error handling**: ✅ Custom 404/500 pages with proper status codes

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