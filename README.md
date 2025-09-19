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

### Key Changes
- Simplified repository ETL to sequential processing in `internal/data/repository.go`
- Handlers are thin and fetch pre-computed data from the repository (`internal/handlers/handlers.go`)
- Centralized configuration in `internal/config/config.go` (tests modify these variables)
- Templates are self-contained under `templates/` and rendered via `Handler.render`

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

### Project Principles
- Keep handlers thin: business logic belongs in `internal/data`
- Centralize runtime settings in `internal/config` and override in tests
- Templates are self-contained; avoid template inheritance

### Data Flow Overview
1. Startup: `cmd/server` creates a `data.Repository` and calls `repo.LoadData()`
2. Load: repository fetches API data, processes it, caches images (optional), and builds indexes
3. Serve: handlers query repository getters like `GetArtists()` and `GetArtistBySlug()`
4. Render: handlers pass small inline structs to templates in `templates/`

go test ./... -cover
## 🧪 Testing & Quality

Run internal unit tests (clean):
```bash
go test ./internal/... -v
```

Run audit/e2e tests (may have mixed package declarations):
```bash
go test ./tests/... 
```

Coverage report for internal packages:
```bash
go test -cover ./internal/...
```

## Recent Notes
- The repository pattern and template system were simplified in 2025 to reduce runtime cost and improve testability.
- Handlers and templates show examples in `internal/handlers` and `templates/` respectively.
