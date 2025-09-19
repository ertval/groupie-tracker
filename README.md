# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with idiomatic Go following KISS principles and clean architecture patterns.

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- Implements responsive web design with a modular template system
- Provides robust error handling with proper HTTP status codes
- Features panic recovery middleware for server stability
- Achieves high test coverage with comprehensive unit and integration tests
- **Recently optimized for simplicity and idiomatic Go best practices**

## ✨ Recent Optimizations (September 2025)

### Code Simplifications Applied:
- **Simplified Repository Pattern**: Streamlined the ETL pipeline from complex multi-step process to clear, sequential operations
- **Reduced Handler Duplication**: Created common template data structures to eliminate repetitive code patterns
- **Improved Image Caching**: Simplified caching logic with better error handling and resource management
- **Enhanced Error Handling**: Consolidated validation patterns using helper methods
- **Go Best Practices**: Applied idiomatic Go patterns throughout the codebase

### Performance Improvements:
- Reduced memory allocations through better slice initialization
- Eliminated unnecessary temporary data structures
- Streamlined API data processing pipeline
- Improved template rendering with common data structures

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

## 🏗️ Architecture (Simplified)

The application follows a clean, simplified architecture that separates concerns into distinct packages.

```
cmd/server/                 # Main application entry point and server setup
internal/
    ├── data/               # Simplified data management layer
    │   ├── repository.go   # Clean repository pattern with sequential processing
    │   ├── domain.go       # Domain models
    │   └── api.go          # API data structures
    ├── handlers/           # HTTP handlers with inline struct patterns
    │   ├── handlers.go     # Simplified request handlers with inline data structs
    │   └── utils.go        # Shared utilities and helpers
    └── config/             # Centralized configuration
templates/                  # Go templates for rendering HTML
static/                     # Static assets (CSS, JS, images)
```

## 🎯 Key Improvements Made

### KISS Principle Applied

**Before**: Complex ETL pipeline with multiple return values and nested transformations
```go
// Old complex pattern
finalArtists, totalConcerts, countrySet := r.buildArtists(apiArtists, apiRelations)
cachedCount, downloadedCount, failedCount := r.cacheArtistImages(finalArtists)
finalArtists = r.sortAndLinkArtists(finalArtists)
finalLocations := r.buildLocations(finalArtists)
r.populateData(finalArtists, finalLocations, totalConcerts, countrySet, cachedCount, downloadedCount, failedCount)
```

**After**: Simple, sequential data processing
```go
// New simplified pattern
artists := r.processArtists(apiArtists, apiRelations)
cachedCount, downloadedCount := r.cacheImages(artists)
locations := r.createLocations(artists)
r.loadProcessedData(artists, locations, cachedCount, downloadedCount)
```

### Inline Template Data Structures

**Before**: Separate template data file with complex inheritance
```go
// templates.go - separate file with many types
type PageData struct { ... }
type HomeData struct { PageData; ... }
type ArtistsData struct { PageData; ... }
```

**After**: Inline struct definitions in handlers
```go
// Directly in handler methods
data := struct {
    Title    string
    ExtraCSS string
    Artists  []data.Artist
}{
    Title:    "Artists",
    ExtraCSS: "artists.css", 
    Artists:  artists,
}
```
### Idiomatic Go Patterns

**Request Validation**: Consolidated repetitive validation logic:
```go
// Before: Repeated in every handler
if r.Method != http.MethodGet {
    h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
    return
}
if r.URL.Path != expectedPath {
    h.Error(w, r, http.StatusNotFound, "Page not found")
    return
}

// After: Single helper method
if !h.validateGETRequest(w, r, expectedPath) {
    return
}
```

**Cache Logging Fix**: Improved cache statistics tracking:
```go
// Before: Always showed 0 cached images
Data loaded successfully with warm cache - 52 artists (Loaded 0 images from cache)

// After: Shows actual cached image count  
Data loaded successfully with warm cache - 52 artists (Loaded 52 images from cache)
```

### Benefits of Inline Template Structs

1. **Locality**: Template data is defined exactly where it's used
2. **Simplicity**: No need to maintain separate template struct file
3. **Flexibility**: Each handler can define exactly what it needs
4. **Type Safety**: Direct use of domain types instead of `interface{}`
if !h.validateGETRequest(w, r, expectedPath) {
    return
}
```

### Simplified Repository Pattern

The `data` package implements a clean repository pattern with sequential processing:

1. **Data Loading**: `repo.LoadData()` processes data in clear steps
2. **Simple Processing**: Each step has a single responsibility
3. **Fast Lookups**: Pre-computed maps and sorted slices for O(1) access
4. **Clear Flow**: API data → Process → Cache → Index → Ready

### Data Flow
1. **Startup:** Create `data.Repository` instance
2. **Load:** `repo.LoadData()` fetches and processes all data sequentially
3. **Serve:** Handlers perform fast lookups using pre-computed indexes
4. **Render:** Templates receive structured data through common patterns

## 🧪 Testing & Quality

The project maintains high test coverage with simplified, maintainable tests:

- **`internal/data`**: Unit tests for repository logic using mock HTTP server
- **`internal/handlers`**: HTTP handler tests with comprehensive route coverage
- **`internal/handlers`**: Integration tests for the HTTP handlers, ensuring they respond correctly to different requests and render the appropriate templates.
- **`cmd/server`**: Integration tests for the server setup, routing, and middleware.

```bash
# Run all tests with coverage
go test ./... -cover

# Generate an HTML coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## 📝 Recent Updates (September 2025)

### Major Refactoring

The application underwent a significant refactoring to improve performance, simplicity, and maintainability.

- **Decoupled Data Structures:** Created a clear separation between raw API data models (`internal/data/api.go`) and the application's internal domain models (`internal/data/domain.go`).
- **Pre-computed Repository:** The `Repository` now holds fully pre-computed and query-ready data. All sorting, slug generation, and statistical calculations are performed once at startup in the `LoadData` function.
- **Simplified Handlers:** HTTP handlers are now much simpler. They are only responsible for fetching pre-computed data from the repository and passing it to templates. All business logic has been moved out of the handlers and into the data layer.
- **Zero Runtime Computation:** Getter methods on the repository (`GetArtists`, `GetArtistBySlug`, etc.) now perform simple map lookups or return pre-sorted slices, resulting in much faster response times.
- **Modular Templates:** The template system was refactored to use a base template (`base.tmpl`), making the individual page templates cleaner and easier to maintain.
- **Improved Testing:** The test suite was rewritten to use a mock HTTP server, providing faster and more reliable tests for the data and handler packages.
