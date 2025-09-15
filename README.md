# Groupie Tracker

A clean, modern web application that displays information about bands and artists by consuming data from the Groupie Trackers API. The application provides a simple, focused interface to explore artist information, concert locations, and dates.

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates
- Implements responsive web design with clean templates
- Provides robust error handling with proper HTTP status codes

## 📊 API Data Structure

The application consumes four main API endpoints:

1. **Artists** (`/api/artists`) - Band/artist information including:
   - Name(s), image, creation year
   - First album date and members

2. **Locations** (`/api/locations`) - Concert venues and locations

3. **Dates** (`/api/dates`) - Concert dates (past and upcoming)

4. **Relations** (`/api/relation`) - Links between artists, locations, and dates

## 🏗️ Project Structure (Clean Architecture - September 2025)

```
groupie-tracker/
├── cmd/
│   └── server/
│       ├── main.go           # Application entry point
│       ├── server.go         # HTTP server configuration and routing
│       ├── main_test.go      # Main server tests
│       └── server_test.go    # Server functionality tests
├── internal/
│   ├── api/                  # API client and data fetching
│   │   ├── client.go         # HTTP client for external API
│   │   └── client_test.go    # Client tests
│   ├── models/               # Data structures and validation
│   │   ├── models.go         # Core data models
│   │   └── models_test.go    # Model validation tests
│   ├── storage/              # Unified data storage layer
│   │   ├── store.go          # Single store implementation
│   │   └── store_test.go     # Storage tests
│   ├── service/              # Business logic layer
│   │   ├── service.go        # Business logic and calculations
│   │   └── service_test.go   # Service tests
│   └── handlers/             # HTTP request handlers
│       ├── handlers.go       # HTTP handlers for all routes
│       └── handlers_test.go  # Handler tests
├── templates/                # HTML templates
│   ├── base.tmpl            # Base template
│   ├── home.tmpl            # Home page template
│   ├── artists.tmpl         # Artists listing template
│   ├── artist_detail.tmpl   # Individual artist template
│   ├── locations.tmpl       # Locations page template
│   └── error.tmpl           # Error page template
├── static/                   # Static assets
│   └── css/                 # Stylesheets
├── tests/                    # End-to-end and integration tests
├── doc/                      # Documentation
├── go.mod                    # Go module file
├── README.md                 # This file
└── LICENSE                   # Project license
```

## 🏛️ Architecture Overview

The application follows a clean, layered architecture:

### Storage Layer (`internal/storage`)
- **Single Store**: Unified data store handling all operations
- **Thread-safe**: Concurrent read/write operations
- **Auto-refresh**: Periodic API data updates (1-hour intervals)
- **Caching**: Optional API client integration for data refresh
- **Data Management**: Efficient storage and retrieval of API data

### Service Layer (`internal/service`) 
- **Business Logic**: All calculations and data processing
- **Location Statistics**: Concert frequency and popularity metrics
- **Data Aggregation**: Total counts, unique locations, etc.
- **Clean Interface**: Simple, focused methods

### Handler Layer (`internal/handlers`)
- **Single Handler Struct**: All HTTP handlers in one place
- **Template Management**: Centralized template loading with error handling
- **Error Handling**: Proper HTTP 500 errors when templates fail
- **Strict URL Validation**: URLs with extra path segments return 404

### API Client (`internal/api`)
- **HTTP Client**: Fetches data from external API
- **Timeout Handling**: Request timeout management
- **Error Recovery**: Graceful error handling

## 🚀 Getting Started

### Prerequisites
- Go 1.19 or higher
- Internet connection (for API data)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd groupie-tracker
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Run the application**
   ```bash
   go run ./cmd/server/
   ```

4. **Access the application**
   Open your browser and navigate to `http://localhost:8080`

### Building for Production

```bash
# Build the executable
go build -o server ./cmd/server/

# Run the executable
./server
```

### Environment Variables

- `PORT`: Server port (default: 8080)
- `API_URL`: External API URL (default: https://groupietrackers.herokuapp.com)

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run Specific Package Tests
```bash
# Storage tests
go test ./internal/storage/ -v

# Service tests
go test ./internal/service/ -v

# Handler tests
go test ./internal/handlers/ -v

# Server tests
go test ./cmd/server/ -v
```

### Test Coverage
```bash
go test ./... -cover
```

## 📄 API Endpoints

### Web Routes
- `GET /` - Home page
- `GET /artists` - Artists listing
- `GET /artists/{slug}` - Individual artist page (strict validation)
- `GET /locations` - Locations page
- `GET /locations/{slug}` - Individual location page

### API Routes
- `GET /healthz` - Health check

### Static Files
- `/static/css/*` - Stylesheets

## 📋 Features

### Core Features
- ✅ **Artist Listings**: Browse all artists with sorting
- ✅ **Artist Details**: Detailed artist information with concert data
- ✅ **Location Statistics**: Concert locations with frequency data
- ✅ **Location Details**: Detailed location pages with artist information
- ✅ **Responsive Design**: Mobile-friendly interface
- ✅ **Proper Error Pages**: Custom 404/500 error pages

### Technical Features
- ✅ **Clean Architecture**: Well-separated concerns
- ✅ **Comprehensive Testing**: Unit and integration tests
- ✅ **Auto-refresh**: Automatic API data updates every hour
- ✅ **Strict Error Handling**: Template failures return HTTP 500
- ✅ **URL Validation**: Extra path segments in URLs return 404
- ✅ **Logging**: Request logging and error tracking
- ✅ **Health Checks**: Application health monitoring
- ✅ **Data Caching**: In-memory data storage
- ✅ **Template Engine**: Server-side HTML rendering
- ✅ **Graceful Shutdown**: Clean server termination

## 🎨 Design Principles

1. **Simplicity**: Clean, focused code without unnecessary complexity
2. **Separation of Concerns**: Each layer has a single responsibility  
3. **Testability**: Comprehensive test coverage for all components
4. **Performance**: Efficient data structures and caching
5. **Maintainability**: Clear naming and documentation
6. **Robust Error Handling**: Proper HTTP status codes and error responses
7. **URL Integrity**: Strict validation of request paths

## 🔧 Development

### Code Organization
- **Models**: Data structures with validation
- **Storage**: Single source of truth for application data
- **Service**: Business logic and calculations
- **Handlers**: HTTP request/response handling
- **Templates**: HTML presentation layer

### Testing Strategy
- **Unit Tests**: Individual component testing
- **Integration Tests**: Component interaction testing
- **Handler Tests**: HTTP endpoint testing
- **Mock Objects**: External dependency mocking

### Performance Considerations
- **In-Memory Storage**: Fast data access
- **Template Caching**: Pre-compiled templates
- **Request Logging**: Performance monitoring
- **Concurrent Safety**: Thread-safe operations

## 📖 Documentation

Additional documentation can be found in the `doc/` directory:
- API documentation
- Architecture decisions
- Testing strategies
- Deployment guides

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Groupie Trackers API](https://groupietrackers.herokuapp.com/api) for providing the data
- Go community for excellent tooling and libraries
- Contributors and reviewers

---

**Note**: This project has been simplified to focus on core functionality with robust error handling. Template failures now return proper HTTP 500 errors, and URL validation is strict to ensure API integrity. The codebase is maintainable, testable, and follows clean architecture principles.
