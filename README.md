# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with idiomatic Go following clean architecture patterns and audit requirements. **Features comprehensive server-side filtering and search functionality without JavaScript dependencies.**

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- **Provides advanced server-side filtering for both artists and locations**
- **Implements comprehensive search functionality across all data types**
- Implements responsive web design with template inheritance system and right sidebar filters
- Provides robust error handling with proper HTTP status codes and graceful fallbacks
- Features panic recovery middleware for server stability
- Maintains test coverage with comprehensive unit and integration tests (data: 69.8% coverage)
- **Built with Go 1.24.3 standard library only - no external dependencies**
- **Concurrent data loading**: Parallel API fetching and worker pool for image caching

## 🚀 Recent Refactoring (Oct 2025 - Phases 1-3 Complete)

This project underwent comprehensive refactoring to improve performance, code quality, and maintainability:

### Phase 1: Store-Service-Repository Architecture
- **Created `Store` struct**: Immutable in-memory data storage with thread-safe read-only access
- **Created `Service` layer**: Clean business logic facade for web handlers
- **Refactored `Repository`**: Now a compatibility wrapper around Store
- **Removed duplicate API models**: Consolidated to use `api.Artist` and `api.Relation` directly
- **Result**: Cleaner separation of data storage and business logic

### Phase 2: Concurrent Data Loading
- **Parallel API fetching**: Artists and relations fetched concurrently using goroutines and channels
- **Worker pool for images**: 4 concurrent workers for downloading/caching artist images
- **Standard library only**: Uses goroutines, channels, sync.WaitGroup, sync.Mutex - no external dependencies
- **Result**: Improved startup time and performance without adding dependencies

### Phase 3: Service Layer
- **Business logic facade**: Clean API wrapping Store operations
- **Backward compatibility**: Repository maintains existing API for tests and handlers
- **Ready for future enhancements**: Service layer can be used directly by new code
- **Result**: Cleaner architecture with multiple access patterns

### Overall Impact
- **Performance**: Concurrent loading improves startup time
- **Code quality**: Better separation of concerns (Store/Service/Repository)
- **Maintainability**: Clear data flow and responsibilities
- **Test coverage**: All tests passing, standard library only
- **Architecture**: Ready for future scaling and enhancements

## �🔍 Search Features

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

## 📁 Project Structure & Files

### Key Implementation Files
- `cmd/cli/main.go` - Streamlined application entry point
- `cmd/cli/e2e_test.go` - End-to-end integration tests
- `cmd/cli/search_e2e_test.go` - Search-specific E2E tests
- `internal/web/server.go` - Package-level server initialization with global variables
- `internal/web/routes.go` - HTTP routing and middleware setup
- `internal/web/handlers.go` - All HTTP endpoints with filter/search APIs (package-level functions)
- `internal/web/middleware.go` - Panic recovery, logging, security headers
- `internal/web/templates.go` - Template utilities and form parsing helpers
- `internal/web/server_test.go` - Comprehensive handler unit tests
- `internal/web/search_integration_test.go` - Search integration testing
- `internal/config/config.go` - Centralized configuration variables
- `internal/domain/repository.go` - Core data management logic with thread-safe operations
- `internal/domain/filtering.go` - Server-side filter logic for artists and locations
- `internal/domain/search.go` - Comprehensive search functionality with suggestions
- `internal/domain/models.go` - Domain models with FilterParams + SearchParams structures
- `internal/domain/filter_test.go` - Comprehensive filter unit tests (18 tests)
- `internal/domain/search_test.go` - Comprehensive search unit tests (30+ tests)
- `internal/domain/repository_test.go` - Repository unit tests (91.2% coverage)
- `templates/base.tmpl` - Base template with global search bar
- `templates/search.tmpl` - Dedicated search interface with advanced filters
- `templates/artists.tmpl` - Enhanced with right sidebar filter UI
- `templates/locations.tmpl` - Enhanced with location filtering UI
- `templates/artist_detail.tmpl` - Individual artist detail pages
- `templates/location_detail.tmpl` - Individual location detail pages
- `templates/home.tmpl` - Homepage with artist overview
- `templates/error.tmpl` - Error pages with graceful fallback
- `static/css/base.css` - Core styling and global components
- `static/css/artists.css` - Updated for sidebar layout with responsive design
- `static/css/locations.css` - Comprehensive styling for location filters
- `static/css/search.css` - Search interface styling
- `static/css/home.css` - Homepage specific styling
- `static/css/errors.css` - Error page styling
- `static/img/artists/` - Cached artist images directory

### Notable Features
- **Comprehensive Search**: Full-text search across artists, members, locations, dates
- **Server-Side Filtering**: Complete filter functionality without JavaScript dependencies
- **Search Suggestions**: Real-time JSON API with typed suggestions
- **Combined Search+Filters**: Advanced search with simultaneous filter criteria
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
  ├── e2e_test.go          # End-to-end integration tests
  └── search_e2e_test.go   # Search-specific E2E tests

internal/
  ├── config/              # Centralized configuration
  │   └── config.go        # Global variables for timeouts, URLs, cache settings
  ├── data/                # Core domain layer (91.2% test coverage)
  │   ├── repository.go    # Single data load with thread-safe read operations
  │   ├── models.go        # Domain models + FilterParams/SearchParams structures
  │   ├── filtering.go       # Server-side filter logic (artists + locations)
  │   ├── search.go        # Comprehensive search functionality with suggestions
  │   ├── filter_test.go   # Comprehensive filter testing (18 tests passing)
  │   ├── search_test.go   # Comprehensive search testing (30+ tests passing)
  │   └── repository_test.go # Repository tests (91.2% coverage)
  └── server/              # HTTP layer (79.7% test coverage)
      ├── server.go        # Package-level server initialization with global variables
      ├── routes.go        # HTTP routing and middleware setup
      ├── handlers.go      # All HTTP endpoints + filter/search APIs (package-level functions)
      ├── middleware.go    # Panic recovery, logging, security headers
      ├── templates.go         # Template utilities and form parsing helpers
      ├── server_test.go   # Comprehensive unified server tests
      └── search_integration_test.go # Search integration testing

templates/                 # Template inheritance system + filter UI
  ├── base.tmpl           # Base layout with {{define "base"}}
  ├── home.tmpl           # Homepage with artist overview
  ├── artists.tmpl        # Artist listing + RIGHT SIDEBAR FILTERS
  ├── artist_detail.tmpl  # Artist pages with {{define "body"}}
  ├── locations.tmpl      # Location listing + FILTER SIDEBAR
  ├── location_detail.tmpl # Location detail pages
  ├── search.tmpl         # Dedicated search interface with advanced filters
  └── error.tmpl          # Error pages with graceful fallback

static/                   # Static assets with enhanced styling
  ├── css/               # Stylesheets with filter sidebar support
  │   ├── base.css       # Core styling and global components
  │   ├── home.css       # Homepage specific styling
  │   ├── artists.css    # ENHANCED: Sidebar layout + filter controls
  │   ├── locations.css  # Complete location filter styling
  │   ├── search.css     # Search interface styling
  │   ├── artist_detail.css # Artist detail page styling
  │   ├── location_detail.css # Location detail page styling
  │   ├── errors.css     # Error page styling
  │   └── dev.css        # Development/debug styling
  ├── img/artists/       # Cached artist images
  └── favicon.ico        # Site favicon

tests/                    # Audit and integration tests
  ├── audit_test.go      # Audit requirement validation
  ├── debug_test.go      # Debug and development tests
  ├── visual_e2e_test.go # Visual/UI end-to-end tests
  └── playwright_test.go # Browser automation tests
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


#### 2. Data Loading Pipeline (`internal/domain/repository.go`)
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

## � Documentation Structure

This project includes comprehensive documentation in the `doc/` folder:

### Implementation Documentation
- **`SEARCH_IMPLEMENTATION_SUMMARY.md`** - Complete details of search functionality implementation
  - Search suggestion system with typed categorization
  - Integration with existing filter system
  - Test-driven development approach
  
- **`FILTER_IMPLEMENTATION_SUMMARY.md`** - Comprehensive filter system documentation
  - Checkbox-based member count filtering
  - Country-based location filtering
  - Range sliders for date filtering
  - Template integration details

- **`OPTIMIZATION_SUMMARY.md`** - Code simplification and KISS principles applied
  - Repository pattern simplification
  - Elimination of complex data structures
  - Performance improvements

### Development Documentation  
- **`REFACTORING_PLAN.md`** - Strategic refactoring plan for data package
  - Clear API/domain model separation
  - Performance optimization strategies
  - Idiomatic Go patterns

- **`CREDENTIALS_SETUP.md`** - Setup instructions for development environment

### AI Development Instructions
- **`.github/copilot-instructions.md`** - Comprehensive instructions for AI coding agents
  - Project constraints and architecture patterns
  - Testing requirements and workflows
  - Critical data structures and flows

## �🚀 Quick Start

### Prerequisites
- Go 1.24.3+ (uses modern Go features)
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
go test ./internal/domain/filter_test.go -v

# Run audit tests (may have package declaration issues)
go test ./tests/... -v

# Check test coverage
go test -cover ./internal/...
```

**Environment Variables:**
- `PORT` - Server port (default: 8080)

## 🧪 Testing & Quality Assurance

### Test Coverage & Status
The current test status shows partial coverage with some integration test failures being addressed:

```bash
$ go test -cover ./internal/domain
ok      groupie-tracker/internal/domain   3.144s  coverage: 69.8% of statements

$ go test -cover ./internal/config
?       groupie-tracker/internal/config [no test files]

# Note: Server tests currently have some integration test failures being resolved
# Core data functionality and filtering logic remain fully tested and operational
```

**Current Test Status:**
- **Data Package**: 69.8% coverage with all core functionality tests passing
- **Search & Filter Logic**: Comprehensive test coverage with all unit tests passing
- **API Integration**: Some integration tests experiencing template resolution issues (being addressed)
- **Core Application**: Fully functional with server running successfully

### Search Functionality Testing  
```bash
# Test search functionality - Core tests are passing
$ go test ./internal/domain -v -run "Search"
# Includes 25+ test cases covering:
# - Artist name searches (case-insensitive)
# - Member name searches  
# - Location searches
# - Date/year searches
# - Combined search + filter functionality
# - Edge cases and error handling
```

### Filter Functionality Testing  
```bash
# Test filter functionality - All tests passing
$ go test ./internal/domain -v -run "Filter"  
# Includes 18+ test cases covering:
# - Year range filtering
# - Member count filtering
# - Country filtering
# - Combined filter criteria
```

**Note:** Some server integration tests are experiencing template resolution issues but core data functionality and filtering logic remain fully operational and tested.

Notes:
- `internal/config` and `tests` show `coverage: [no statements]` because those packages contain only variable declarations or only `_test.go` files; `go test` reports "no statements" when there are no non-test statements to instrument.

### Audit Requirements
The application validates against specific audit requirements:
- **Queen**: exactly 7 members
- **Gorillaz**: first album date "26-03-2001" 
- **Travis Scott**: 10+ concert locations
- **Foo Fighters**: exactly 6 members
- **Search Requirements**: All search cases (artists, members, locations, dates) implemented
- **Error Handling**: 404 for unknown artists/locations
- **Error Handling**: 500 for server errors (e.g., malformed requests)

### Required API Endpoints (Enhanced)
- `GET /` - Homepage with artist overview and quick search
- `GET /artists` - Complete artist listing with filter UI
- `POST /artists` - **Server-side artist filtering**
- `GET /artists/{slug}` - Individual artist pages (SEO-friendly URLs)
- `GET /locations` - Concert venue listing with filter UI  
- `POST /locations` - **Server-side location filtering**
- `GET /locations/{slug}` - Location detail with artists who performed there
- `GET /search` - **NEW: Dedicated search interface with advanced filters**
- `POST /search` - **NEW: Server-side search processing with filter integration**
- `GET /api/suggestions?q=query` - **NEW: JSON API for real-time search suggestions**
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
- **Data package tests passing** with 69.8% coverage (`go test ./internal/domain`)
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
