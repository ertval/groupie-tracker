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

## 🏗️ Architecture (September 2025)

### Idiomatic Go Repository Pattern
The application follows idiomatic Go patterns with clean architecture:

```
cmd/server/main.go           # Entry point with graceful shutdown
internal/
  └── repository/            # Core data management
      ├── repository.go      # Single repository with all functionality
      └── repository_test.go # Comprehensive test coverage
  └── handlers/              # HTTP request handlers
      ├── handlers.go        # All HTTP endpoints
      └── handlers_test.go   # Handler tests
templates/                   # Self-contained HTML templates
static/css/                  # Page-specific stylesheets
tests/                      # End-to-end and audit tests
doc/                        # Current documentation
```

### Repository Design

#### API Response Structs
Direct mappings from the 4 API endpoints:
- `ArtistAPIResponse` - from `/api/artists` 
- `LocationAPIResponse` - from `/api/locations`
- `DateAPIResponse` - from `/api/dates`
- `RelationAPIResponse` - from `/api/relation`

#### Domain Models
Processed business logic structures:
- `Artist` - Musical artist with SEO slug
- `Concert` - Concert information with location-date mappings
- `LocationStats` - Location statistics with concert dates per artist
- `ComputedData` - Internal processed data structure

#### Single Repository
- One exported `Repository` struct
- Thread-safe data access
- Single initialization: `NewRepository(baseURL, timeout)`
- Single data load: `LoadData(ctx)` fetches from all 4 endpoints
- Precomputed indexes and statistics
  ├── api/client.go         # External API consumption (1:1 API mapping)
  ├── data/data.go          # Repository pattern with all business logic
  └── handlers/handlers.go  # HTTP handlers with adapter pattern
templates/                  # Self-contained HTML templates
static/css/                 # Page-specific stylesheets
tests/                     # Audit compliance & E2E tests
```

### Key Features

#### Repository Pattern
```go
// Initialize repository with API connection
repo := repository.NewRepository("https://groupietrackers.herokuapp.com", 30*time.Second)

// Load data once from all 4 API endpoints  
err := repo.LoadData(ctx)

// Access data through repository methods
artists := repo.GetArtists()                    // All artists sorted by name
artist, found := repo.GetArtistBySlug("queen")  // Artist by SEO slug
locationStats := repo.GetLocationStats()        // All locations with statistics
```

#### Enhanced Location Details
- **Concert dates per artist**: Location detail pages now show specific concert dates for each artist
- **Rich statistics**: Artist count, total shows, concert count per location
- **SEO-friendly URLs**: Clean slugs for all artists and locations

#### Template System
- **Self-contained templates**: Each `.tmpl` file is a complete HTML document  
- **No template inheritance**: Direct execution with clear data structures
- **Concert dates display**: Shows dates under artist member count in location details

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

### Idiomatic Go Refactoring (September 16, 2025)
- ✅ **Clean repository architecture**: Single repository struct with proper separation of concerns
- ✅ **API response structs**: Direct mappings from all 4 API endpoints without duplication
- ✅ **Enhanced location details**: Concert dates now displayed under artist member count  
- ✅ **Domain model clarity**: Clear separation between API responses and domain models
- ✅ **Comprehensive testing**: All tests updated and passing with new structure
- ✅ **Documentation cleanup**: Removed outdated docs, created current architecture summary
- ✅ **Performance optimization**: Single data load with precomputed statistics
- ✅ **Thread safety**: Repository methods designed for concurrent access

### Architecture Improvements
- **Four API endpoint usage**: Properly utilizes `/api/artists`, `/api/locations`, `/api/dates`, `/api/relation`
- **No data duplication**: Single source of truth for all application data
- **Computed data structure**: Efficient internal data organization for template needs
- **Location concert dates**: Enhanced location pages show when each artist performed there

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