# Groupie Tracker

A robust, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. Built with idiomatic Go, following the KISS (Keep It Simple, Stupid) principle and clean architecture.

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates with SEO-friendly URLs
- Implements responsive web design with a modular, base-template structure
- Provides robust error handling with proper HTTP status codes
- Features panic recovery middleware for server stability
- Achieves high test coverage with comprehensive unit and integration tests

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

## 🏗️ Architecture

The application follows a clean architecture that separates concerns into distinct packages.

```
cmd/server/                 # Main application entry point and server setup
internal/
    ├── data/               # Core data management
    │   ├── api.go          # Structs for raw API responses
    │   ├── domain.go       # Internal, pre-computed data models
    │   └── repository.go   # The repository, responsible for data loading and access
    └── handlers/           # HTTP request handlers
        └── handlers.go     # All HTTP endpoints and template rendering logic
templates/                  # Go templates for rendering HTML
static/                     # Static assets (CSS, JS, images)
```

### Repository Pattern

The `data` package implements a repository pattern. The `Repository` is the single source of truth for all application data. It fetches raw data from the external API, processes it into clean internal models, and pre-computes all necessary values at startup. This ensures that the HTTP handlers are fast and simple, as they only need to read from the repository without performing any calculations.

### Data Flow
1.  **Startup:** The `main` function creates a `data.Repository` instance.
2.  **Data Loading:** `repo.LoadData()` is called once.
    - It fetches data from all API endpoints.
    - It transforms the raw API data into the application's internal domain models (`Artist`, `Location`).
    - It performs all necessary computations (slug generation, sorting, calculating stats, building relationships for navigation).
    - It stores the final, query-ready data in slices and maps within the repository.
3.  **Request Handling:**
    - An HTTP request comes in.
    - The corresponding handler function is called.
    - The handler calls a getter method on the repository (e.g., `repo.GetArtistBySlug(...)`).
    - The getter method performs a fast, simple lookup in a pre-computed map or returns a pre-sorted slice.
    - The handler passes the retrieved data to a template for rendering.

## 🧪 Testing & Quality

The project aims for high test coverage to ensure reliability and maintainability.

- **`internal/data`**: Unit tests for the repository's data loading and processing logic, using a mock HTTP server to provide consistent test data.
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
