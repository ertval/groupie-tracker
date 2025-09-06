# Groupie Tracker

A web application that displays information about bands and artists by consuming data from the Groupie Trackers API. The application provides an interactive interface to explore artist information, concert locations, and dates.

## 🎯 Project Overview

Groupie Tracker is a Go-based web application that:
- Fetches data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- Displays artist information, concert locations, and dates
- Provides search and filtering functionality
- Implements client-server communication through interactive events

## 📊 API Data Structure

The application consumes four main API endpoints:

1. **Artists** (`/api/artists`) - Band/artist information including:
   - Name(s), image, creation year
   - First album date and members

2. **Locations** (`/api/locations`) - Concert venues and locations

3. **Dates** (`/api/dates`) - Concert dates (past and upcoming)

4. **Relations** (`/api/relation`) - Links between artists, locations, and dates

## 🏗️ Project Structure

```
groupie-tracker/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/                     # API client and data fetching
│   ├── models/                  # Data structures
│   ├── handlers/                # HTTP handlers
│   ├── storage/                 # In-memory data storage
│   └── search/                  # Search functionality
├── templates/                   # HTML templates
│   ├── base.tmpl
│   ├── artists.tmpl
│   ├── artist_detail.tmpl
│   └── locations.tmpl
├── static/                      # Static assets (CSS, JS, images)
│   ├── css/
│   ├── js/
│   └── img/
├── tests/                       # Test files
└── docs/                        # Documentation
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Internet connection (for API access)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd groupie-tracker
```

2. Initialize Go module:
```bash
go mod init groupie-tracker
```

3. Download dependencies:
```bash
go mod tidy
```

4. Run the application:
```bash
go run cmd/server/main.go
```

5. Open your browser and navigate to `http://localhost:8080`

### Development

Run tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -cover ./...
```

Build the application:
```bash
go build -o groupie-tracker cmd/server/main.go
```

## 🌐 Application Features

### 🎵 Core Functionality
- **Artist Discovery**: Browse 50+ artists with comprehensive information
- **Live Search**: Instant search with autocomplete suggestions and debouncing
- **Concert Locations**: View global concert venues across 100+ cities
- **Concert Dates**: Browse historical concert dates and tour information
- **Data Relations**: Explore connections between artists, locations, and dates

### 🎨 User Experience
- **Beautiful Design**: Modern gradient UI with smooth CSS animations
- **Responsive Layout**: Optimized for desktop, tablet, and mobile devices
- **Interactive Cards**: Hover effects and smooth transitions throughout
- **Loading States**: Elegant loading animations and error handling
- **Real-time Feedback**: Instant visual feedback for all user interactions

### ⚡ Performance & Technical
- **Fast Search**: Debounced live search with instant suggestions (300ms delay)
- **Concurrent Safety**: Thread-safe in-memory storage for high performance
- **Error Recovery**: Graceful error handling with user-friendly messages
- **Data Refresh**: Real-time data updates from the Groupie Trackers API
- **Memory Efficient**: Optimized data structures and caching strategies

### 🔍 Interactive Events (Client-Server Communication)
- **Live Search**: Real-time search suggestions with keyboard navigation
- **Data Refresh**: Manual refresh endpoint (`POST /api/refresh`) to update data
- **Advanced Filtering**: Dynamic filtering and searching capabilities
- **Auto-suggestions**: Smart suggestions based on artist names and members
- **Responsive UI**: Instant visual updates without page reloads

## 🧪 Testing

The project follows Test-Driven Development (TDD) principles:

- **Unit Tests**: Individual component testing
- **Integration Tests**: End-to-end functionality testing
- **Handler Tests**: HTTP endpoint testing
- **Audit Compliance Tests**: Validation against project requirements

### Test Specific Data Points

The application is tested against specific data points from the audit:

- ✅ Queen members verification (7 members including Freddie Mercury)
- ✅ Gorillaz first album date (26-03-2001)
- ✅ Travis Scott concert locations (10+ international venues)
- ✅ Foo Fighters members verification (6 current members)

## 🔗 API Endpoints

### Web Routes
- `GET /` - Home page with search functionality
- `GET /artists` - Artists listing page
- `GET /artists/{id}` - Individual artist detail page
- `GET /locations` - Concert locations page

### API Routes
- `GET /api/search?q={query}` - Search artists by name or member
- `GET /api/suggest?q={query}` - Get search suggestions
- `POST /api/refresh` - Refresh data from external API
- `GET /healthz` - Health check endpoint

### Static Assets
- `/static/css/main.css` - Main stylesheet with animations
- `/static/js/main.js` - JavaScript for live search and interactions
- Travis Scott concert locations
- Foo Fighters member list

## 📋 API Endpoints

### Web Routes
- `GET /` - Dashboard/Home page
- `GET /artists` - Artists listing with search and filters
- `GET /artists/{id}` - Individual artist details
- `GET /locations` - Locations overview

### API Routes
- `GET /api/search` - Search functionality
- `GET /api/suggest` - Autocomplete suggestions
- `POST /api/refresh` - Refresh data from external API
- `GET /healthz` - Health check endpoint

## 🛡️ Error Handling

The application includes comprehensive error handling:
- Custom 404 and 500 error pages
- Graceful degradation when API is unavailable
- Input validation and sanitization
- Server crash prevention with recovery middleware

## 🔧 Configuration

Environment variables:
- `PORT` - Server port (default: 8080)
- `API_BASE_URL` - Base URL for the Groupie Trackers API
- `TIMEOUT` - API request timeout (default: 30s)

## 📝 Development Guidelines

- **Code Quality**: All code must pass `go vet` and `golint`
- **Testing**: Maintain >80% test coverage
- **Documentation**: Update README for significant changes
- **Commits**: Small, focused commits with descriptive messages
- **Standards**: Follow Go best practices and conventions

## 🤝 Contributing

1. Follow TDD principles - write tests first
2. Ensure all tests pass before committing
3. Update documentation as needed
4. Follow the established project structure
5. Commit frequently with clear messages

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 References

- [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)
- [Go Documentation](https://golang.org/doc/)
- [HTTP Status Codes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status)
- [RESTful API Best Practices](https://restfulapi.net/)

---

## Development Status

🚧 **Current Phase**: Initial Setup and Core Implementation

See [todo.md](todo.md) for detailed development progress and next steps.
