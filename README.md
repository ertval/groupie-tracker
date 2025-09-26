# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with idiomatic Go following clean architecture patterns and audit requirements. **Features comprehensive server-side filtering without JavaScript dependencies.**

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- **Provides advanced server-side filtering for both artists and locations**
- Implements responsive web design with template inheritance system and right sidebar filters
- Provides robust error handling with proper HTTP status codes and graceful fallbacks
- Features panic recovery middleware for server stability
- Achieves solid test coverage with comprehensive unit and integration tests (data: 88.9%, handlers: 79.7%, overall: 82.1%)
- **Built with Go 1.24+ following Test-Driven Development principles and Zero JavaScript Dependencies**

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

## 📁 Project Structure & Files

### Key Implementation Files
- `cmd/cli/main.go` - Streamlined application entry point
- `internal/server/server.go` - Package-level server initialization with global variables
- `internal/server/routes.go` - HTTP routing and middleware setup
- `internal/server/handlers.go` - All HTTP endpoints with filter APIs (package-level functions)
- `internal/server/middleware.go` - Panic recovery, logging, security headers
- `internal/config/config.go` - Centralized configuration variables
- `internal/data/repository.go` - Core data management logic (396 lines)
- `internal/data/filters.go` - **NEW: Server-side filter logic for artists and locations**
- `internal/data/models.go` - Domain models with FilterParams structures  
- `templates/base.tmpl` - Base template with inheritance support
- `templates/artists.tmpl` - **Enhanced with right sidebar filter UI**
- `templates/locations.tmpl` - **Enhanced with location filtering UI**
- `static/css/artists.css` - **Updated for sidebar layout with responsive design**
- `static/css/locations.css` - **NEW: Comprehensive styling for location filters**

### Notable Features
- **Server-Side Filtering**: Complete filter functionality without JavaScript dependencies
- **Native HTML Controls**: Uses details/summary elements for collapsible filters  
- **Dual-Range Filters**: Number inputs for year and count ranges with bounds validation
- **Checkbox Grids**: Multi-select filtering for discrete values (member counts, countries)
- **Right Sidebar Layout**: CSS Grid-based responsive design with sticky positioning
- **Image Caching System**: Optional artist image caching to `static/img/artists/`
- **SEO-Friendly URLs**: `/artists/queen` instead of `/artists/28`
- **Responsive Design**: Mobile-first CSS with flexbox layouts and sidebar conversion
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
cmd/cli/                    # Application entry point  
  ├── main.go              # Streamlined server startup
  └── e2e_test.go          # End-to-end integration tests

internal/
  ├── config/              # Centralized configuration
  │   └── config.go        # Global variables for timeouts, URLs, cache settings
  ├── data/                # Core domain layer (88.9% test coverage)
  │   ├── repository.go    # Single data load with thread-safe read operations
  │   ├── models.go        # Domain models + FilterParams structures
  │   ├── filters.go       # NEW: Server-side filter logic (artists + locations)
  │   ├── filter_test.go   # NEW: Comprehensive filter testing (18 tests passing)
  │   └── repository_test.go # Repository tests
  └── server/              # HTTP layer (79.7% test coverage)
      ├── server.go        # Package-level server initialization with global variables
      ├── routes.go        # HTTP routing and middleware setup
      ├── handlers.go      # All HTTP endpoints + filter APIs (package-level functions)
      ├── middleware.go    # Panic recovery, logging, security headers
      └── server_test.go   # Comprehensive unified server tests

templates/                 # Template inheritance system + filter UI
  ├── base.tmpl           # Base layout with {{define "base"}}
  ├── artist_detail.tmpl  # Artist pages with {{define "body"}}
  ├── artists.tmpl        # Artist listing + RIGHT SIDEBAR FILTERS
  ├── home.tmpl          # Homepage
  ├── locations.tmpl     # Location listing + FILTER SIDEBAR
  ├── location_detail.tmpl # Location detail pages
  └── error.tmpl         # Error pages with graceful fallback

static/                   # Static assets with enhanced styling
  ├── css/               # Stylesheets with filter sidebar support
  │   ├── base.css       # Core styling 
  │   ├── artists.css    # ENHANCED: Sidebar layout + filter controls
  │   └── locations.css  # NEW: Complete location filter styling
  ├── img/artists/       # Cached artist images
  └── favicon.ico        # Site favicon
```

### 🔄 Detailed Data Flow

#### 1. Application Startup
```
NewServer() → LoadData(ctx) → loadTemplates() → routes() → ListenAndServe()

Flow:
- Initialize package-level repository and templates variables
- Load all data from Groupie Trackers API 
- Parse and cache HTML templates from templates/ directory
- Set up HTTP routes with package-level handler functions
- Start HTTP server with middleware chain
```


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
  ├── routes.go: mux.HandleFunc("/artists/", ArtistDetail)
  ├── handlers.go: ArtistDetail() extracts "queen" from URL path
  ├── repository.go: GetArtistBySlug("queen") returns cached Artist
  ├── handlers.go: Creates inline struct with Artist + metadata
  └── templates/artist_detail.tmpl: Renders using template inheritance
```

#### 4. Template Inheritance System


### 🔒 Thread Safety & Performance
- **Single data load**: All data loaded once at startup, no runtime API calls
- **Package-level variables**: Repository and templates stored as global variables following config pattern
- **Read-only operations**: Repository methods only read from pre-built maps
- **Concurrent safe**: Multiple requests can safely access repository data
- **Simplified architecture**: Removed App struct in favor of package-level functions
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
go run ./cmd/cli/

# Or build and run
go build -o groupie-tracker ./cmd/cli/
./groupie-tracker

# Run internal tests (clean, all passing)
go test ./internal/... -v

# Run filter tests specifically  
go test ./internal/data/filter_test.go -v

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
ok      groupie-tracker/cmd/cli        0.351s  coverage: 84.8% of statements
ok      groupie-tracker/cmd/testapi    (cached)        coverage: 53.3% of statements
ok      groupie-tracker/internal/config (cached)        coverage: [no statements]
ok      groupie-tracker/internal/data   (cached)        coverage: 88.9% of statements
ok      groupie-tracker/internal/server        0.322s  coverage: 79.7% of statements
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

### Required API Endpoints (Enhanced)
- `GET /` - Homepage with artist overview
- `GET /artists` - Complete artist listing with filter UI
- `POST /artists` - **NEW: Server-side artist filtering**
- `GET /artists/{slug}` - Individual artist pages (SEO-friendly URLs)
- `GET /locations` - Concert venue listing with filter UI  
- `POST /locations` - **NEW: Server-side location filtering**
- `GET /locations/{slug}` - Location detail with artists who performed there
- `GET /health` - JSON health check for monitoring

## 🔧 Development Workflow

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
- **All internal tests passing** (`go test ./internal/...`)
- **Filter tests passing**: 18/18 filter-specific tests pass
- **Test coverage**: Overall 82.1% (data: 88.9%, server: 79.7%) 
- **Zero-dependency project** - uses only Go standard library, no JavaScript
- **Production-ready** with graceful shutdown and error recovery
- **Audit requirements compliant** with required endpoints and data validation
- **Accessibility compliant** - server-side rendering with native HTML controls

### 📊 Technical Metrics
- **Server-side filtering** for both artists and locations without JavaScript dependencies
- **CSS Grid responsive layout** with right sidebar filters that convert to top section on mobile
- **Native HTML controls** using details/summary for collapsible functionality
- **Thread-safe** read operations after initial data load with comprehensive filter logic
- **SEO-friendly URLs** with slug-based routing (`/artists/queen`)
- **Template inheritance** with base/body pattern for consistent UI across all pages

---

### Summary
The application integrates with the Groupie Trackers API which provides artist, location, date, and relation data. It handles the API's inconsistent response formats by normalizing data structures and building efficient search indexes for fast lookups by ID and slug. The repository loads all data once at startup, caches artist images if enabled, and provides thread-safe read operations for concurrent requests. Handlers extract URL parameters, fetch data from the repository, and render HTML templates using a base layout with inheritance.

**Built with ❤️ using Go 1.24+ | Zero Dependencies | No JavaScript | Server-Side Filtering | Test-Driven Development | Idiomatic Go | Claude Sonnet**
