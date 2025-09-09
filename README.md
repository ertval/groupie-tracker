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
│   ├── handlers/                # HTTP handlers with template system
│   ├── storage/                 # In-memory data storage
│   └── search/                  # Search functionality
├── templates/                   # Go HTML templates (✅ COMPLETED)
│   ├── base.tmpl               # Master layout with conditional blocks
│   ├── home.tmpl               # Home page with statistics
│   ├── artists.tmpl            # Artists listing page
│   ├── artist_detail.tmpl      # Individual artist details
│   ├── locations.tmpl          # Concert locations page
│   └── error.tmpl              # Error handling (404/500)
├── static/                      # Static assets
│   ├── css/                    # CSS files (ready for styling)
│   │   ├── base.css           # Base styles
│   │   ├── home.css           # Home page styles
│   │   ├── artists.css        # Artists page styles
│   │   ├── artist_detail.css  # Artist detail styles
│   │   ├── locations.css      # Locations page styles
│   │   └── errors.css         # Error page styles
│   ├── js/                    # JavaScript files
│   └── img/                   # Images
├── tests/                     # Test files
└── docs/                      # Documentation
```

## 🎨 Template System (✅ COMPLETED)

The application uses a sophisticated Go HTML template system with inheritance and conditional rendering:

### Template Architecture
- **Master Layout**: `base.tmpl` provides the common HTML structure
- **Conditional Content**: Base template conditionally includes content blocks based on page title
- **Unique Content Blocks**: Each page has its own content block to prevent naming conflicts

### Template Files
```
templates/
├── base.tmpl           # Master layout with conditional content inclusion
├── home.tmpl           # {{define "home-content"}} block
├── artists.tmpl        # {{define "artists-content"}} block  
├── artist_detail.tmpl  # {{define "artist-detail-content"}} block
├── locations.tmpl      # {{define "locations-content"}} block
└── error.tmpl          # {{define "error-content"}} block
```

### Template Inheritance Pattern
Each page template follows this pattern:
```go
{{template "base.tmpl" .}}
{{define "unique-content-name"}}
<!-- Page-specific content here -->
{{end}}
```

The base template conditionally includes content:
```go
{{if eq .Title "Home"}}
    {{template "home-content" .}}
{{else if eq .Title "Artists"}}
    {{template "artists-content" .}}
{{else if contains .Title "Artist"}}
    {{template "artist-detail-content" .}}
{{else if eq .Title "Locations"}}
    {{template "locations-content" .}}
{{else}}
    {{template "error-content" .}}
{{end}}
```

### Custom Template Functions
- `sub` - Subtraction: `{{sub .Total 1}}`
- `add` - Addition: `{{add .Index 1}}`
- `contains` - String matching: `{{contains .Title "Artist"}}`

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
go run ./cmd/server
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
go build -o groupie-tracker ./cmd/server
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
- `GET /` - Home page with search functionality and statistics
- `GET /artists` - Artists listing page with search and filters
- `GET /artists/{id}` - Individual artist detail page with concert info
- `GET /locations` - Concert locations page with statistics

### API Routes
- `GET /api/search?q={query}` - Search artists by name or member
- `GET /api/suggest?q={query}` - Get search suggestions
- `POST /api/refresh` - Refresh data from external API
- `GET /healthz` - Health check endpoint

### Static Assets
- `/static/css/*.css` - Page-specific stylesheets
- `/static/js/*.js` - JavaScript for interactive features
- `/static/img/*` - Images and assets

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

## 🎨 CSS/JS Developer Guide

### 📋 Current Template Structure & CSS Integration

The Go templates are **fully implemented and working**. Each template automatically loads its corresponding CSS file:

#### Template → CSS Mapping
```
Page Template          → CSS File Loaded
├── home.tmpl          → /static/css/base.css + /static/css/home.css
├── artists.tmpl       → /static/css/base.css + /static/css/artists.css
├── artist_detail.tmpl → /static/css/base.css + /static/css/artist_detail.css
├── locations.tmpl     → /static/css/base.css + /static/css/locations.css
└── error.tmpl         → /static/css/base.css + /static/css/errors.css
```

### 🏗️ HTML Structure & CSS Classes

#### Base Template Structure (base.tmpl)
```html
<header class="site-header">
  <div class="container">
    <h1><a href="/">Groupie Tracker</a></h1>
    <nav>
      <a href="/">Home</a> • <a href="/artists">Artists</a> • <a href="/locations">Locations</a>
    </nav>
  </div>
</header>

<main class="main-content">
  <div class="container">
    <!-- Page-specific content inserted here -->
  </div>
</main>
```

#### Key CSS Classes to Style
- `.site-header` - Main navigation header
- `.container` - Content wrapper (consistent width/padding)
- `.main-content` - Main content area
- `.stats-section` - Statistics cards container (home page)
- `.stats-grid` - Grid layout for stat cards
- `.stat-card` - Individual statistic cards
- `.featured-artists` - Featured artists section
- `.artist-grid` - Grid layout for artist cards
- `.artist-card` - Individual artist cards
- `.artist-image` - Artist images
- `.artist-info` - Artist information container
- `.members-count` - Member count display
- `.search-container` - Search input container
- `.suggestions-dropdown` - Search suggestions dropdown

### 📊 Template Data Structures

#### Home Page Data (home.tmpl)
```go
type HomeData struct {
    Title         string           // "Home"
    Artists       []models.Artist  // All artists for featured section
    TotalMembers  int             // Total member count across all artists
    TotalCountries int            // Total unique countries
    TotalConcerts int             // Total concert count
    ExtraCSS      string          // "home.css"
    ExtraJS       string          // Future JS file name
}
```

#### Artists Page Data (artists.tmpl)
```go
type ArtistsData struct {
    Title    string           // "Artists"
    Artists  []models.Artist  // All artists for listing
    ExtraCSS string          // "artists.css"
    ExtraJS  string          // Future JS file name
}
```

#### Artist Detail Data (artist_detail.tmpl)
```go
type ArtistDetailData struct {
    Title         string            // "{ArtistName} - Groupie Tracker"
    Artist        models.Artist     // Current artist details
    Relations     *models.Relation  // Concert dates and locations
    TotalConcerts int              // Concert count for this artist
    PrevArtist    *models.Artist   // Previous artist (navigation)
    NextArtist    *models.Artist   // Next artist (navigation)
    ExtraCSS      string           // "artist_detail.css"
    ExtraJS       string           // Future JS file name
}
```

#### Locations Page Data (locations.tmpl)
```go
type LocationsData struct {
    Title           string              // "Locations"
    LocationStats   []LocationStat      // Location statistics
    TotalCountries  int                // Total unique countries
    TotalConcerts   int                // Total concerts across all locations
    ExtraCSS        string             // "locations.css"
    ExtraJS         string             // Future JS file name
}

type LocationStat struct {
    Location     string  // "paris-france"
    DisplayName  string  // "Paris, France"
    ConcertCount int     // Number of concerts at this location
}
```

### 🔍 JavaScript Integration Points

#### Search Functionality
- **Search Input**: `#search-input` - Main search field
- **Suggestions**: `#search-suggestions` - Dropdown container
- **API Endpoints**:
  - `GET /api/search?q={query}` - Full search results
  - `GET /api/suggest?q={query}` - Autocomplete suggestions

#### Interactive Elements
- **Artist Cards**: `.artist-card` - Click handlers for navigation
- **Navigation**: Previous/Next artist buttons on detail pages
- **Statistics**: `.stat-card` - Potential click interactions
- **Search Suggestions**: `.suggestion-item` - Click to select

### 🎯 Styling Priorities

#### 1. **Base Styles (base.css)**
- Navigation header styling
- Container layouts and responsive design
- Typography and color scheme
- Button and link styles

#### 2. **Home Page (home.css)**
- Statistics cards grid layout
- Featured artists grid
- Search input styling
- Hero section if desired

#### 3. **Artists Page (artists.css)**
- Artist listing grid/layout
- Search and filter controls
- Artist card hover effects

#### 4. **Artist Detail (artist_detail.css)**
- Artist profile layout
- Concert information display
- Navigation buttons (prev/next)
- Member list styling

#### 5. **Locations Page (locations.css)**
- Location statistics display
- Geographic information layout
- Concert count visualizations

#### 6. **Error Pages (errors.css)**
- 404/500 error page styling
- Error message display
- Navigation back to main site

### 🚀 Development Workflow

1. **Start the server**: `cd cmd/server && go run .`
2. **View pages**: Visit `http://localhost:8080` to see current templates
3. **Live reload**: Restart server after CSS changes to see updates
4. **Test all pages**: 
   - Home: `http://localhost:8080/`
   - Artists: `http://localhost:8080/artists`
   - Artist Detail: `http://localhost:8080/artists/1` (Queen)
   - Locations: `http://localhost:8080/locations`
   - Error: `http://localhost:8080/nonexistent`

### 💡 CSS Development Tips

- **Responsive Design**: All templates include viewport meta tag
- **CSS Grid/Flexbox**: Use modern layout techniques for `.artist-grid`, `.stats-grid`
- **Hover Effects**: Add transitions to `.artist-card`, `.stat-card`
- **Loading States**: Consider skeleton screens for dynamic content
- **Accessibility**: Ensure proper contrast ratios and focus states

## Development Status

✅ **Template System**: Fully implemented and tested
🎨 **Next Phase**: CSS Styling and JavaScript Enhancement

**Completed Work:**
- ✅ All 6 Go HTML templates created and working
- ✅ Template inheritance system with conditional content blocks
- ✅ Custom template functions (sub, add, contains)
- ✅ Data structures and handlers fully integrated
- ✅ Server running without errors
- ✅ All endpoints tested and functional

**Ready for CSS/JS Development:**
- 🎨 CSS files are linked and ready for styling
- 🎨 HTML structure is stable and semantic
- 🎨 Template data is available for JavaScript integration
- 🎨 Search API endpoints are ready for frontend implementation

See [todo.md](todo.md) for detailed development progress.
