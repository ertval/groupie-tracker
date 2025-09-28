# Groupie Tracker - QWEN.md

## Project Overview

Groupie Tracker is a robust, modern web application built with Go that displays information about bands and artists by consuming data from the Groupie Trackers API. The application follows clean architecture patterns with idiomatic Go code and provides comprehensive server-side filtering and search functionality without JavaScript dependencies.

### Key Features
- **API Integration**: Fetches data from the Groupie Trackers API (https://groupietrackers.herokuapp.com/api)
- **Advanced Server-Side Filtering**: Complete filtering for both artists and locations without JavaScript
- **Comprehensive Search**: Search functionality across all data types (artist names, member names, locations, dates)
- **SEO-Friendly URLs**: Uses slugs for clean URLs (e.g., `/artists/queen` instead of `/artists/28`)
- **Responsive Design**: Template inheritance system with right sidebar filters
- **Robust Error Handling**: Proper HTTP status codes and graceful fallbacks
- **Panic Recovery**: Middleware for server stability
- **High Test Coverage**: 82.1% overall test coverage (data: 88.9%, handlers: 79.7%)

## Architecture

### Project Structure
```
├── cmd/                    # Application entry points
│   ├── cli/               # Main application server
│   └── testapi/           # Test API utilities
├── doc/                   # Documentation files
├── internal/              # Core application code
│   ├── config/            # Configuration variables
│   ├── data/              # Data layer (repository, models, filters, search)
│   └── server/            # HTTP server (handlers, middleware, routing)
├── static/                # Static assets (CSS, images)
├── templates/             # HTML templates
└── tests/                 # Test files
```

### Clean Architecture Layers

#### 1. Data Layer (`internal/data/`)
- **Repository Pattern**: Centralized data management with thread-safe read operations
- **Models**: Domain models with FilterParams and SearchParams structures
- **Filters**: Server-side filter logic for artists and locations
- **Search**: Comprehensive search functionality with suggestions

#### 2. Server Layer (`internal/server/`)
- **Server Initialization**: Package-level server with global variables
- **Routing**: HTTP routing and middleware setup
- **Handlers**: All HTTP endpoints with filter/search APIs
- **Middleware**: Panic recovery, logging, security headers

#### 3. Configuration (`internal/config/`)
- Centralized configuration variables for API URLs, timeouts, cache settings, etc.

## Data Flow

### 1. Application Startup
```
NewServer() → LoadData(ctx) → loadTemplates() → routes() → ListenAndServe()

Flow:
- Initialize package-level repository and templates variables
- Load all data from Groupie Trackers API
- Parse and cache HTML templates from templates/ directory
- Set up HTTP routes with package-level handler functions
- Start HTTP server with middleware chain
```

### 2. Data Loading Pipeline
```
API Fetch → Process & Normalize → Cache Images → Build Indexes → Store in Memory

Step 1: fetchAPIData(ctx)
  ├── GET /api/artists → []APIArtist
  ├── GET /api/locations → APILocation{Index: [...]}
  ├── GET /api/dates → APIDate{Index: [...]}
  └── GET /api/relation → APIRelation{Index: [...]}

Step 2: processArtists(apiArtists, apiRelations)
  ├── Merge artist data with concert dates/locations
  ├── Generate SEO slugs (e.g., "Queen" → "queen")
  └── Create Concert structs with venue and date info

Step 3: cacheImages(artists) [if config.WithCache = true]
  ├── Download artist images to static/img/artists/
  ├── Track cache status (CacheDisabled/CacheCold/CacheWarm)
  └── Return download/cache statistics

Step 4: createLocations(artists)
  ├── Extract unique venues from all concerts
  ├── Generate location slugs and artist counts
  └── Sort by artist count (most popular first)

Step 5: loadProcessedData()
  ├── Store in thread-safe maps: artistsByID, artistsBySlug, locationsBySlug
  ├── Pre-compute global statistics (total artists, members, locations)
  └── Ready for concurrent read access
```

### 3. Request Handling Flow
```
HTTP Request → Router → Handler → Repository → Template → Response

Example: GET /artists/queen
  ├── routes.go: mux.HandleFunc("/artists/", ArtistDetail)
  ├── handlers.go: ArtistDetail() extracts "queen" from URL path
  ├── repository.go: GetArtistBySlug("queen") returns cached Artist
  ├── handlers.go: Creates inline struct with Artist + metadata
  └── templates/artist_detail.tmpl: Renders using template inheritance
```

## Key Components

### Repository Pattern
The Repository provides centralized data management with:
- **Single data load**: All data loaded once at startup, no runtime API calls
- **Thread-safe operations**: Package-level variables with read-only access after initialization
- **Fast lookups**: O(1) access via pre-built maps (by ID and slug)
- **Caching**: Optional image caching to static filesystem

### Filtering System
Server-side filtering supports multiple criteria for both artists and locations:
- **Artist Filtering**: Creation year range, first album year range, member count, countries
- **Location Filtering**: Concert count range, artist count range, countries, concert years
- **Filter UI**: Right sidebar layout with range sliders and checkbox grids

### Search System
Comprehensive search functionality that works across multiple data types:
- **Search Areas**: Artists, members, locations, creation dates, album dates
- **Case-insensitive**: All searches work regardless of case
- **Suggestions**: Real-time JSON API with typed categorization
- **Combined Search**: Integration with filter criteria for advanced queries

### Configuration
All configuration is managed in `internal/config/config.go`:
- `WithCache`: Enable/disable image caching
- `APIBaseURL`: Base URL for the Groupie Tracker API
- `APIRequestTimeout`: Request timeout for API calls
- `DefaultPort`: Server port (default: 8080)
- `ReadTimeout`, `WriteTimeout`, `IdleTimeout`: HTTP server timeouts

## HTTP Endpoints

### Required API Endpoints
- `GET /` - Homepage with artist overview and quick search
- `GET /artists` - Complete artist listing with filter UI
- `POST /artists` - Server-side artist filtering
- `GET /artists/{slug}` - Individual artist pages (SEO-friendly URLs)
- `GET /locations` - Concert venue listing with filter UI
- `POST /locations` - Server-side location filtering
- `GET /locations/{slug}` - Location detail with artists who performed there
- `GET /search` - Dedicated search interface with advanced filters
- `POST /search` - Server-side search processing with filter integration
- `GET /api/suggestions?q=query` - JSON API for real-time search suggestions
- `GET /health` - JSON health check for monitoring

## Development Workflow

### Running the Application
```bash
# Run the server (starts on localhost:8080)
go run ./cmd/cli/

# Or build and run
go build -o groupie-tracker ./cmd/cli/
./groupie-tracker
```

### Testing
```bash
# Run internal tests
go test ./internal/... -v

# Run filter tests specifically
go test ./internal/data/filter_test.go -v

# Check test coverage
go test -cover ./internal/...
```

### Environment Variables
- `PORT` - Server port (default: 8080)

## Development Principles

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

## Thread Safety & Performance
- **Single data load**: All data loaded once at startup, no runtime API calls
- **Package-level variables**: Repository and templates stored as global variables following config pattern
- **Read-only operations**: Repository methods only read from pre-built maps
- **Concurrent safe**: Multiple requests can safely access repository data
- **Simplified architecture**: Removed App struct in favor of package-level functions
- **Pre-computed indexes**: SEO slugs, statistics, and mappings built at startup
- **Memory efficient**: Data stored in optimized Go maps and slices