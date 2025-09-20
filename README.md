# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with idiomatic Go following clean architecture patterns and audit requirements.

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- Implements responsive web design with template inheritance system
- Provides robust error handling with proper HTTP status codes and graceful fallbacks
- Features panic recovery middleware for server stability
- Achieves solid test coverage with comprehensive unit and integration tests (data: 71.9%, handlers: 51.9%)
- **Built with Go 1.24+ following Test-Driven Development principles and Idiomatic Go practices**

## 📁 Project Structure & Files

### Key Implementation Files
- `cmd/server/main.go` - Application entry point with graceful shutdown
- `cmd/server/server.go` - HTTP server, routing, middleware (162 lines)
- `internal/config/config.go` - Centralized configuration variables
- `internal/data/repository.go` - Core data management logic (396 lines)
- `internal/data/domain.go` - Domain models (Artist, Location, Concert)
- `internal/handlers/handlers.go` - All HTTP endpoints (453 lines)
- `templates/base.tmpl` - Base template with inheritance support
- `static/css/base.css` - Core styling with responsive design

### Notable Features
- **Image Caching System**: Optional artist image caching to `static/img/artists/`
- **SEO-Friendly URLs**: `/artists/queen` instead of `/artists/28`
- **Responsive Design**: Mobile-first CSS with flexbox layouts
- **Graceful Error Handling**: Proper HTTP status codes with custom error pages
- **Health Monitoring**: `/health` endpoint returns JSON status
- **Development Tools**: Debug endpoints for testing error conditions

## 🎯 API Integration Details

### Groupie Trackers API Endpoints
```
https://groupietrackers.herokuapp.com/api/
├── /artists     → []Artist (direct array)
├── /locations   → {index: [...]} (wrapped in index field)
├── /dates       → {index: [...]} (wrapped in index field)  
└── /relation    → {index: [...]} (wrapped in index field)
```

### Data Normalization Process
The application handles the API's inconsistent response formats:
- **Artists endpoint**: Returns direct array of artist objects
- **Other endpoints**: Return objects with `index` field containing the actual data
- **Relations processing**: Merges concert dates, locations, and artist data
- **Slug generation**: Creates URL-friendly identifiers from artist/location names

## 🏗️ Architecture & Data Flow

### Clean Architecture Structure
```
cmd/server/                 # Application entry point
  ├── main.go              # Graceful shutdown and server lifecycle
  ├── server.go            # HTTP server setup, routing, middleware
  └── server_test.go       # Server integration tests

internal/
  ├── config/              # Centralized configuration
  │   └── config.go        # Global variables for timeouts, URLs, cache settings
  ├── data/                # Core domain layer (71.9% test coverage)
  │   ├── repository.go    # Single data load with thread-safe read operations
  │   ├── domain.go        # Domain models (Artist, Location, Concert)
  │   ├── api.go           # API response structures  
  │   └── repository_test.go # Repository tests
  └── handlers/            # HTTP layer (51.9% test coverage)
      ├── handlers.go      # All HTTP endpoints (453 lines)
      └── handlers_test.go # Comprehensive handler tests

templates/                 # Template inheritance system
  ├── base.tmpl           # Base layout with {{define "base"}}
  ├── artist_detail.tmpl  # Artist pages with {{define "body"}}
  ├── artists.tmpl        # Artist listing
  ├── home.tmpl          # Homepage
  ├── locations.tmpl     # Location listing
  ├── location_detail.tmpl # Location detail pages
  └── error.tmpl         # Error pages with graceful fallback

static/                   # Static assets with proper MIME types
  ├── css/               # Stylesheets (base.css + page-specific)
  ├── img/artists/       # Cached artist images
  └── favicon.ico        # Site favicon
```

### 🔄 Detailed Data Flow

#### 1. Application Startup


#### 2. Data Loading Pipeline (`internal/data/repository.go`)
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

#### 3. Request Handling Flow
```
HTTP Request → Router → Handler → Repository → Template → Response

Example: GET /artists/queen
  ├── server.go: mux.HandleFunc("/artists/", h.ArtistDetail)
  ├── handlers.go: ArtistDetail() extracts "queen" from URL path
  ├── repository.go: GetArtistBySlug("queen") returns cached Artist
  ├── handlers.go: Creates inline struct with Artist + metadata
  └── templates/artist_detail.tmpl: Renders using template inheritance
```

#### 4. Template Inheritance System


### 🔒 Thread Safety & Performance
- **Single data load**: All data loaded once at startup, no runtime API calls
- **Read-only operations**: Repository methods only read from pre-built maps
- **Concurrent safe**: Multiple requests can safely access repository data
- **Pre-computed indexes**: SEO slugs, statistics, and mappings built at startup
- **Memory efficient**: Data stored in optimized Go maps and slices

## 🚀 Quick Start

### Prerequisites
- Go 1.24+ (uses modern Go features)
- Internet connection (for Groupie Trackers API)

### Installation & Running
```bash
# Clone the repository
git clone <repository-url>
cd groupie-tracker

# Run the server (starts on localhost:8080)
go run ./cmd/server/

# Or build and run
go build -o groupie-tracker ./cmd/server
./groupie-tracker

# Run internal tests (clean, all passing)
go test ./internal/... -v

# Run audit tests (may have package declaration issues)
go test ./tests/... -v

# Check test coverage
go test -cover ./internal/...
```

**Environment Variables:**
- `PORT` - Server port (default: 8080)

## 🧪 Testing & Quality Assurance

### Test Coverage & Status
The following is the exact output from running the full test suite with coverage in this workspace:

```bash
$ go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
ok      groupie-tracker/cmd/server      0.351s  coverage: 84.8% of statements
ok      groupie-tracker/cmd/testapi     (cached)        coverage: 53.3% of statements
ok      groupie-tracker/internal/config (cached)        coverage: [no statements]
ok      groupie-tracker/internal/data   (cached)        coverage: 88.9% of statements
ok      groupie-tracker/internal/handlers       0.322s  coverage: 79.7% of statements
ok      groupie-tracker/tests   (cached)        coverage: [no statements]
total:                                                  (statements)            82.1%
```

Notes:
- `internal/config` and `tests` show `coverage: [no statements]` because those packages contain only variable declarations or only `_test.go` files; `go test` reports "no statements" when there are no non-test statements to instrument.

### Audit Requirements
The application validates against specific audit requirements:
- **Queen**: exactly 7 members
- **Gorillaz**: first album date "26-03-2001" 
- **Travis Scott**: 10+ concert locations
- **Foo Fighters**: exactly 6 members
- **Error Handling**: 404 for unknown artists/locations
- **Error Handling**: 500 for server errors (e.g., malformed requests)

### Required API Endpoints
- `GET /` - Homepage with artist overview
- `GET /artists` - Complete artist listing  
- `GET /artists/{slug}` - Individual artist pages (SEO-friendly URLs)
- `GET /locations` - Concert venue listing
- `GET /locations/{slug}` - Location detail with artists who performed there
- `GET /health` - JSON health check for monitoring

## 🔧 Development Workflow

### Key Development Principles
1. **Test-Driven Development** - Always write tests before implementation
2. **Zero Dependencies** - Use only Go standard library
3. **Centralized Configuration** - All settings in `internal/config` package
4. **Template Inheritance** - Use `{{define "base"}}` with `{{template "body" .}}`
5. **Inline Data Structures** - Handler data structs defined inline for type safety
6. **Thread-Safe Operations** - Repository is read-only after initial data load
7. **Graceful Error Handling** - Proper HTTP status codes and error pages
8. **Idiomatic Go** - Follow Go best practices and conventions

### Configuration Management
- All configuration variables (API URLs, timeouts, cache settings) are in `internal/config/config.go`

### Error Handling Pattern
- Centralized error handling middleware
- Graceful fallback to error templates
- Consistent JSON error responses

## ⚡ Current Status (September 2025)

### 🟢 Project Health
- **All internal tests passing** (`go test ./internal/...`)
- **Test coverage**: Overall ~62% (data: 71.9%, handlers: 51.9%) 
- **Zero-dependency project** - uses only Go standard library
- **Production-ready** with graceful shutdown and error recovery
- **Audit requirements compliant** with required endpoints and data validation

### 📊 Technical Metrics
- **453 lines** of HTTP handler code in single file
- **396 lines** of repository logic with sequential data processing
- **Thread-safe** read operations after initial data load
- **SEO-friendly URLs** with slug-based routing (`/artists/queen`)
- **Template inheritance** with base/body pattern for consistent UI

---

### Summary
The application integrates with the Groupie Trackers API which provides artist, location, date, and relation data. It handles the API's inconsistent response formats by normalizing data structures and building efficient search indexes for fast lookups by ID and slug. The repository loads all data once at startup, caches artist images if enabled, and provides thread-safe read operations for concurrent requests. Handlers extract URL parameters, fetch data from the repository, and render HTML templates using a base layout with inheritance.

**Built with ❤️ using Go 1.24+ | Zero Dependencies | Test-Driven Development | Idiomatic GO | Claude 4 Sonnet**
