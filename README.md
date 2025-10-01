# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with idiomatic Go following clean architecture patterns and the KISS principle. **Features comprehensive server-side filtering and search functionality without JavaScript dependencies.**

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- **Provides advanced server-side filtering for both artists and locations**
- **Implements comprehensive search functionality across all data types**
- Implements responsive web design with template inheritance system
- Provides robust error handling with proper HTTP status codes and graceful fallbacks
- Features panic recovery middleware for server stability
- Maintains test coverage with comprehensive unit and integration tests
- **Built with Go 1.24.3 following simplified architecture principles and Zero JavaScript Dependencies**

## 🏗️ Simplified Architecture

### Core Design Principles
1. **KISS Principle**: Minimal abstractions, straightforward data flow
2. **Single Responsibility**: Each file has one clear purpose
3. **Idiomatic Go**: Standard patterns, value semantics, clear error handling
4. **Performance**: Pre-computed indexes, minimal allocations
5. **Testability**: Dependency injection, isolated components

### Project Structure
```
cmd/server/                   # Application entry point
  ├── main.go                 # Server bootstrap and lifecycle
  └── e2e_test.go             # End-to-end integration tests

internal/
  ├── api/                    # External API client
  │   ├── types.go            # Raw API response structures
  │   ├── client.go           # HTTP client with timeout
  │   └── client_test.go      # API client tests
  ├── data/                   # Core business logic
  │   ├── models.go           # Domain models (Artist, Location, etc.)
  │   ├── store.go            # Single data store with fast lookups
  │   └── store_test.go       # Store unit tests
  ├── web/                    # HTTP layer
  │   ├── server.go           # Server construction and lifecycle
  │   ├── handlers.go         # All HTTP endpoint handlers
  │   ├── routes.go           # Route configuration and middleware
  │   ├── render.go           # Template rendering utilities
  │   └── server_test.go      # Handler tests
  └── config/                 # Configuration
      └── config.go           # Global settings and timeouts

static/                      # CSS, images, JavaScript
templates/                   # HTML templates
tests/                       # Audit and browser tests
```

## 🔍 Search Features

### Comprehensive Search Functionality (Zero JavaScript)
- **Artist/Band Names**: Case-insensitive search across all artist names
- **Member Names**: Find artists by searching for any band member
- **Concert Locations**: Search by venue cities and countries  
- **Creation Dates**: Search by band formation years
- **First Album Dates**: Search by album release dates
- **Real-time Suggestions**: JSON API provides typed suggestions with categories
- **Combined Search + Filters**: Advanced search with simultaneous filter criteria
- **Server-Side Processing**: All search handled via HTML form submissions

### Search Interface Features
- **Quick Search Bar**: Global search in site header
- **Advanced Search Page**: Dedicated `/search` endpoint with full functionality
- **Suggestion Categories**: Results clearly labeled as "artist", "member", "location", etc.
- **Case-Insensitive**: "QUEEN", "queen", and "Queen" all return identical results
- **Partial Matching**: "Phil" finds both "Phil Collins" and "Philadelphia"
- **No Results Handling**: Helpful suggestions when searches return empty results

## ✨ Filter Features

### Artist Filtering (No JavaScript Required)
- **Creation Year Range**: Filter artists by formation year using number inputs
- **First Album Year Range**: Filter by first album release year  
- **Member Count**: Checkbox selection for band member counts (1-8 members)
- **Countries**: Checkbox grid for countries where artists have performed concerts
- **Right Sidebar Layout**: Collapsible filter panel using native HTML details/summary
- **Server-Side Processing**: All filtering handled via POST requests to maintain accessibility

### Location Filtering  
- **Concert Count Range**: Filter venues by total number of concerts hosted
- **Artist Count Range**: Filter by number of different artists who performed  
- **Countries**: Checkbox selection for venue countries
- **Responsive UI**: Matches artists page design with consistent styling and behavior

## 📁 Package Overview

### `internal/api`
Handles communication with the external Groupie Tracker API:
- `types.go`: Raw API response structures (APIArtist, APIRelation, etc.)
- `client.go`: HTTP client with timeout and error handling
- **Separation of Concerns**: Clean boundary between external data and domain models

### `internal/data`
Core business logic and data management:
- `models.go`: Domain models with computed fields (Artist, Location, Concert, etc.)
- `store.go`: Single data store with pre-computed indexes for fast lookups
  - Artists indexed by ID and slug
  - Locations indexed by slug
  - Pre-computed search suggestions and statistics
  - Integrated filtering and search methods
- **Performance**: All data loaded once at startup, fast in-memory access
- **Simplicity**: No repository pattern, no service layer - direct store access

### `internal/web`
HTTP layer and request handling:
- `server.go`: Server construction, template loading, and lifecycle management
- `handlers.go`: All HTTP endpoint handlers (Home, Artists, ArtistDetail, etc.)
- `routes.go`: Route configuration and middleware setup
- `render.go`: Template rendering utilities
- **Direct Access**: Handlers access store directly - no unnecessary abstractions

### `internal/config`
Centralized configuration management:
- `config.go`: Global settings (API URL, timeouts, ports, cache flags)

## 🚀 Quick Start

### Prerequisites
- Go 1.24.3 or later
- Internet connection (for API data)

### Running the Application
```bash
# Clone the repository
git clone <repository-url>
cd groupie-tracker

# Run the server
go run ./cmd/server/

# Or build and run
go build -o groupie-tracker ./cmd/server/
./groupie-tracker
```

The server will start on `http://localhost:8082`

### Testing
```bash
# Run all tests
go test ./... -v

# Run tests with coverage
go test -cover ./internal/...

# Run specific package tests
go test ./internal/data/... -v
go test ./internal/web/... -v
go test ./cmd/server/... -v
```

## 🎯 API Integration Details

### Groupie Trackers API Endpoints
```
https://groupietrackers.herokuapp.com/api/
├── /artists     → []Artist (direct array)
└── /relation    → {index: [...]} (wrapped in index field)
```

### Data Flow
```
1. Application Startup
   ├── NewStore() creates empty store
   ├── store.LoadData(ctx) fetches API data
   │   ├── client.FetchArtists() → []APIArtist
   │   └── client.FetchRelations() → APIRelation
   ├── Transform API data to domain models
   ├── Build indexes (by ID, by slug)
   ├── Generate locations from concert data
   ├── Pre-compute search suggestions
   └── Calculate statistics

2. HTTP Request
   ├── Middleware (logging, CORS)
   ├── Handler accesses store directly
   ├── Template rendering with data
   └── HTTP response
```

## ✨ Key Features

### Search & Filtering
- **Artist Search**: By name, members, creation year, first album
- **Location Search**: By venue names and countries
- **Filtering**: Year ranges, member counts, countries
- **Suggestions**: Pre-computed search suggestions for autocomplete
- **Server-Side**: All processing done server-side (no JavaScript required)

### Performance Optimizations
- **Pre-computed Indexes**: O(1) lookups by ID and slug
- **Cached Suggestions**: Generated once at startup
- **Minimal Allocations**: Reduced data transformations
- **Fast Filtering**: In-memory filtering with early termination
- **Template Caching**: Pre-compiled templates

### User Experience
- **SEO-Friendly URLs**: `/artist/queen` instead of `/artist/28`
- **Responsive Design**: Mobile-first CSS with flexbox layouts
- **Error Handling**: Proper HTTP status codes with custom error pages
- **Health Endpoint**: `/health` for monitoring
- **Static Assets**: Efficient serving of CSS, images, and favicon
## 🔧 Configuration

All configuration is managed through `internal/config/config.go`:

```go
var (
    WithCache         = false                                        // Image caching
    APIBaseURL        = "https://groupietrackers.herokuapp.com"     // API endpoint
    APIRequestTimeout = 30 * time.Second
    DefaultPort       = ":8082"                                      // Server port
    ReadTimeout       = 15 * time.Second
    WriteTimeout      = 15 * time.Second
    IdleTimeout       = 60 * time.Second
)
```

## 🧪 Testing Strategy

The application follows test-driven development principles:

1. **Unit Tests**: Each package has comprehensive unit tests
2. **Integration Tests**: End-to-end tests in `cmd/server/`
3. **API Tests**: External API integration tests
4. **Coverage**: Target >70% test coverage

### Running Tests
```bash
# Run all tests
go test ./... -v

# Run specific package tests
go test ./internal/data/... -v
go test ./internal/web/... -v
go test ./cmd/server/... -v

# Check coverage
go test -cover ./internal/...
```

### Test Results Summary
- ✅ Unit Tests: All passing (internal/data, internal/api)
- ✅ Integration Tests: All passing (internal/web)
- ✅ E2E Tests: Core functionality passing (cmd/server)
- ⚠️ Known Issues: Favicon 404 (non-critical), HTTP method validation (minor)

## 📚 Documentation

Additional documentation is available in the `doc/` directory:
- `STEP_BY_STEP_IMPLEMENTATION_GUIDE.md` - Detailed refactoring guide with implementation status
- `INTEGRATED_REFACTOR_PLAN.md` - Strategic refactoring plan
- `SEARCH_IMPLEMENTATION_SUMMARY.md` - Search functionality details
- `FILTER_IMPLEMENTATION_SUMMARY.md` - Filter system documentation
- `OPTIMIZATION_SUMMARY.md` - Performance optimization notes

## 🚀 Development Principles

1. **Idiomatic Go**: Follow Go best practices and conventions
2. **KISS Principle**: Keep implementations simple and maintainable
3. **Single Responsibility**: Each file and function has one clear purpose
4. **Explicit Dependencies**: No hidden globals or magic
5. **Error Handling**: Proper error propagation and logging
6. **Performance**: Pre-computed indexes, minimal allocations
7. **Testability**: Dependency injection, isolated components

## 📋 API Endpoints

### Public Routes
- `GET /` - Homepage
- `GET /artists` - List all artists
- `GET /artist/{slug}` - Artist detail page
- `GET /locations` - List all locations
- `GET /location/{slug}` - Location detail page
- `GET /search?q={query}` - Search interface
- `GET /health` - Health check endpoint
- `GET /static/*` - Static assets (CSS, images, etc.)

## 🎯 Key Features

### Search & Discovery
- Full-text search across artists, members, locations, and dates
- Search suggestions for autocomplete
- Case-insensitive search
- Partial matching support

### Filtering
- Artist filters: Creation year range, member counts, countries
- Location filters: Year range, artist counts, countries  
- Combined search + filters for advanced queries

### Performance
- All data loaded once at startup
- Pre-computed indexes for O(1) lookups
- Cached search suggestions
- Fast in-memory filtering
- Minimal memory allocations

### User Experience
- SEO-friendly URLs (`/artist/queen` not `/artist/28`)
- Responsive design
- Error pages with proper HTTP status codes
- Health monitoring endpoint

## � Project Status

**Current Version**: Refactored with simplified architecture  
**Last Updated**: October 1, 2025  
**Go Version**: 1.24.3  
**Status**: ✅ Production Ready

### Achievements
- ✅ Simplified architecture following KISS principles
- ✅ All core functionality implemented and tested
- ✅ Clean separation of concerns (API, Data, Web layers)
- ✅ Comprehensive test coverage
- ✅ No external dependencies (pure Go standard library)
- ✅ Fast performance with pre-computed indexes
- ✅ SEO-friendly URLs and responsive design

### Known Minor Issues
- Favicon returns 404 (file exists but routing issue)
- HTTP method validation not enforced (accepts all methods)

These are non-critical and don't affect core functionality.

## 🤝 Contributing

This is a learning project demonstrating:
- Clean architecture in Go
- KISS principle application
- Test-driven development
- Idiomatic Go patterns
- Performance optimization techniques

---

**Built with ❤️ using Go 1.24.3 and the KISS principle**
- **Server-Side Processing**: All filtering and search handled server-side with HTML forms
- **Thread-Safe Design**: Single data load with concurrent read access
- **Comprehensive Testing**: Test-driven development with extensive unit and integration tests
- **SEO-Friendly**: Slug-based URLs and semantic HTML structure
- **Responsive Design**: Mobile-first CSS with native HTML controls

**Built with ❤️ using Go 1.24.3 | Zero Dependencies | No JavaScript | Server-Side Filtering | Test-Driven Development | Idiomatic Go | Claude Sonnet**
