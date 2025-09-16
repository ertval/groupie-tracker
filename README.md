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

## 🏗️ Project Structure (Clean Architecture - December 2025)

```
groupie-tracker/
├── cmd/
│   └── server/
│       ├── main.go           # Application entry point
│       ├── server.go         # HTTP server configuration and routing
│       └── server_test.go    # Server functionality tests
├── internal/
│   ├── api/                  # External API client (1:1 API mapping)
│   │   ├── client.go         # HTTP client and API data structures
│   │   └── client_test.go    # Client tests
│   ├── data/                 # Application domain models and repository
│   │   ├── data.go           # Domain models and unified repository
│   │   └── data_test.go      # Repository and model tests
│   └── handlers/             # HTTP request handlers
│       ├── handlers.go       # HTTP handlers with API client adapter
│       └── handlers_test.go  # Handler tests
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

The application follows a simplified clean architecture with clear package separation:

### API Package (`internal/api`)
- **External API Client**: HTTP client for fetching data from Groupie Trackers API
- **API Data Structures**: 1:1 mapping with external API response format (Artist, Location, Date, Relation)
- **Network Layer**: Handles timeouts, errors, and API communication
- **No Dependencies**: Independent package with no internal dependencies

### Data Package (`internal/data`) 
- **Application Domain Models**: Business-focused data structures (Artist, Relation, LocationStat, etc.)
- **Unified Repository**: Single Repository struct with all data management logic
- **Business Logic**: Data processing, validation, and utility functions
- **Precomputed Indexes**: Performance optimizations for searches and lookups
- **Interface-based Design**: Uses APIClient interface to avoid import cycles

### Handler Package (`internal/handlers`)
- **HTTP Request Handling**: All web request processing and response generation
- **API Client Adapter**: Bridge between API package types and data package types
- **Template Management**: Centralized template loading and rendering
- **Error Handling**: Proper HTTP status codes and error pages
- **Clean Interface**: Uses interfaces to maintain separation of concerns

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
# API client tests
go test ./internal/api/ -v

# Data layer tests (models + repository)
go test ./internal/data/ -v

# Handler tests
go test ./internal/handlers/ -v

# Service tests
go test ./internal/service/ -v

# Handler tests
go test ./internal/handlers/ -v

# API client tests
go test ./internal/api/ -v

# Server tests
go test ./cmd/server/ -v
```

### Test Coverage
```bash
# Overall coverage (97+ tests across 6 packages)
go test ./... -cover

# Generate HTML coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**Current Test Statistics (December 2025):**
- ✅ 97+ comprehensive tests
- ✅ 6 packages with full test coverage
- ✅ 65%+ overall code coverage
- ✅ All packages: API (62.4%), Data (61.8%), Handlers (71.9%)

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

## 📖 Recent Improvements (December 2025)

### ✨ Architecture Refactoring
- **Simplified Package Structure**: Reduced from 5 internal packages to 3 focused packages
- **Clear Separation of Concerns**: API client, domain models, and HTTP handlers in separate packages
- **Eliminated Import Cycles**: Interface-based design prevents circular dependencies
- **1:1 API Mapping**: API package types directly mirror external API responses

### 🔧 Code Quality Improvements
- **Unified Repository**: Single data store combining models and repository logic
- **Interface-based Communication**: APIClient interface bridges packages without direct dependencies
- **Enhanced Test Coverage**: 97+ comprehensive tests across all packages
- **Clean Architecture**: Clear boundaries between external API, domain logic, and presentation

### 📚 Documentation
- **Updated Architecture Diagrams**: Reflects current simplified structure
- **Comprehensive README**: Current test statistics and coverage information
- **Package Documentation**: Clear responsibilities and interfaces documented

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
