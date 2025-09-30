# API Documentation

This document provides detailed information about the internal API structure and data models used in the Groupie Tracker application.

## Core Data Models

### Artist Model

The `Artist` struct is the primary domain model representing a music artist or band.

```go
type Artist struct {
    ID           int       `json:"id"`           // Unique identifier from external API
    Name         string    `json:"name"`         // Display name of the artist/band
    Slug         string    `json:"slug"`         // URL-friendly version for routing
    Members      []string  `json:"members"`      // All band member names
    CreationYear int       `json:"creation_year"` // Year the artist/band was formed
    FirstAlbum   string    `json:"first_album"`   // Date of first album release
    Image        string    `json:"image"`         // URL to artist's image
    Concerts     []Concert `json:"concerts"`      // All concert performances
    Countries    []string  `json:"countries"`     // Unique countries performed in
    ConcertCount int       `json:"concert_count"` // Total number of concerts
}
```

**Methods:**
- `GetMemberCount() int` - Returns the number of members in the band
- `HasPerformedInCountry(country string) bool` - Checks if artist performed in specified country

### Concert Model

Represents a single concert performance with location and temporal information.

```go
type Concert struct {
    Location string `json:"location"` // Formatted venue/city name
    Country  string `json:"country"`  // Extracted country name
    Date     string `json:"date"`     // Original date string from API
    Year     int    `json:"year"`     // Extracted year for filtering
}
```

### Location Model

Aggregated venue data with performance statistics.

```go
type Location struct {
    Name          string    `json:"name"`           // Formatted location name
    Slug          string    `json:"slug"`           // URL-friendly version
    Artists       []string  `json:"artists"`        // Names of artists who performed
    ArtistCount   int       `json:"artist_count"`   // Number of unique artists
    TotalConcerts int       `json:"total_concerts"` // Total concerts at this venue
    EarliestYear  int       `json:"earliest_year"`  // Year of first concert
    LatestYear    int       `json:"latest_year"`    // Year of most recent concert
    Concerts      []Concert `json:"concerts"`       // All concerts at this location
}
```

**Methods:**
- `GetCountry() string` - Extracts country from location name
- `GetYearRange() string` - Returns formatted year range (e.g., "1990-2020")

## Search and Filtering

### Search Parameters

```go
type SearchParams struct {
    Query   string  `form:"q" json:"query"`        // Main search term
    Filters Filters `form:"filters" json:"filters"` // Additional filtering criteria
}
```

### Search Result

```go
type SearchResult struct {
    Artists      []Artist `json:"artists"`       // Matching artists
    TotalResults int      `json:"total_results"` // Count of matches
    Query        string   `json:"query"`         // Original search query
}
```

### Filter Model

```go
type Filters struct {
    CreationYearMin   int      `form:"creation_year_min"`   // Minimum creation year
    CreationYearMax   int      `form:"creation_year_max"`   // Maximum creation year
    FirstAlbumYearMin int      `form:"first_album_year_min"` // Minimum album year
    FirstAlbumYearMax int      `form:"first_album_year_max"` // Maximum album year
    MemberCounts      []int    `form:"member_counts"`       // Exact member counts
    Countries         []string `form:"countries"`           // Performance countries
}
```

**Methods:**
- `IsEmpty() bool` - Returns true if no filters are applied

### Search Suggestions

```go
type SearchSuggestion struct {
    Text        string               `json:"text"`        // Suggestion text
    Type        SearchSuggestionType `json:"type"`        // Category of suggestion
    Description string               `json:"description"` // Additional context
    URL         string               `json:"url"`         // Direct link (optional)
    ArtistID    int                  `json:"artist_id,omitempty"` // Related artist
}
```

**Suggestion Types:**
- `SuggestionTypeArtist` - Artist name suggestions
- `SuggestionTypeMember` - Band member suggestions  
- `SuggestionTypeLocation` - Concert location suggestions
- `SuggestionTypeFirstAlbum` - First album suggestions
- `SuggestionTypeCreation` - Formation year suggestions

## Application Statistics

```go
type AppStats struct {
    TotalArtists     int `json:"total_artists"`     // Number of artists
    TotalLocations   int `json:"total_locations"`   // Number of unique locations
    TotalConcerts    int `json:"total_concerts"`    // Total concert count
    EarliestYear     int `json:"earliest_year"`     // Oldest concert year
    LatestYear       int `json:"latest_year"`       // Most recent concert year
    CachedImages     int `json:"cached_images"`     // Locally cached images
    DownloadedImages int `json:"downloaded_images"` // Images downloaded
}
```

**Methods:**
- `GetYearRange() string` - Returns formatted year coverage
- `GetAverageConcertsPerArtist() float64` - Calculates average concerts per artist
- `GetAverageConcertsPerLocation() float64` - Calculates average concerts per location

## Service Layer APIs

### DataService

Handles data processing and transformation from API to domain models.

```go
type DataService struct{}

// ProcessAPIData converts API data to domain models with full enrichment
func (ds *DataService) ProcessAPIData(
    apiArtists []api.APIArtist, 
    apiRelations []api.APIRelationIndex
) ([]models.Artist, []models.Location, models.AppStats)
```

**Key Features:**
- Single-pass processing for optimal performance
- Automatic slug generation for URLs
- Country extraction from location strings
- Year parsing from various date formats
- Concert count aggregation

### SearchService

Provides comprehensive search functionality.

```go
type SearchService struct{}

// Search performs multi-field search across all artist data
func (ss *SearchService) Search(
    artists []models.Artist, 
    params models.SearchParams
) models.SearchResult

// GenerateSuggestions creates autocomplete suggestions
func (ss *SearchService) GenerateSuggestions(
    artists []models.Artist
) []models.SearchSuggestion

// FilterSuggestions filters suggestions based on query
func (ss *SearchService) FilterSuggestions(
    suggestions []models.SearchSuggestion, 
    query string
) []models.SearchSuggestion
```

**Search Fields:**
- Artist names (case-insensitive)
- Band member names
- Concert locations and countries
- Creation years
- First album dates

### FilterService

Handles advanced filtering capabilities.

```go
type FilterService struct{}

// FilterArtists applies multi-criteria filtering
func (fs *FilterService) FilterArtists(
    artists []models.Artist, 
    filters models.Filters
) []models.Artist

// GetFilterOptions computes available filter options
func (fs *FilterService) GetFilterOptions(
    artists []models.Artist
) models.FilterOptions

// ParseFiltersFromForm extracts filters from HTTP form data
func (fs *FilterService) ParseFiltersFromForm(
    formValues map[string][]string
) models.Filters
```

**Filter Logic:**
- All filters use AND logic (must match ALL criteria)
- Zero values indicate "no filter applied"
- Range filters are inclusive on both ends
- Array filters use OR logic within the array

## Data Store API

### DataStore

Thread-safe in-memory data storage with fast lookups.

```go
type DataStore struct {
    // Thread-safe methods for data access
}

// Core data access
func (ds *DataStore) GetAllArtists() []models.Artist
func (ds *DataStore) GetAllLocations() []models.Location
func (ds *DataStore) GetStats() models.AppStats

// Fast lookups
func (ds *DataStore) GetArtistByID(id int) (models.Artist, bool)
func (ds *DataStore) GetArtistBySlug(slug string) (models.Artist, bool)
func (ds *DataStore) GetLocationBySlug(slug string) (models.Location, bool)

// Search data
func (ds *DataStore) GetSuggestions() []models.SearchSuggestion

// Counts
func (ds *DataStore) GetArtistCount() int
func (ds *DataStore) GetLocationCount() int
```

**Thread Safety:**
- All methods are thread-safe using RWMutex
- Read operations allow concurrent access
- Write operations (during initialization) are exclusive
- Copy-on-read pattern prevents external modification

## HTTP API Endpoints

### Web Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page with featured artists |
| GET | `/artists` | Artists listing with filtering |
| POST | `/artists` | Artists with applied filters |
| GET | `/artists/{slug}` | Individual artist details |
| GET | `/search` | Search results page |
| POST | `/search` | Search with filters |
| GET | `/locations` | Locations listing |
| GET | `/locations/{slug}` | Individual location details |

### API Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/suggestions?q={query}` | Search suggestions JSON |

### Static Routes

| Path | Description |
|------|-------------|
| `/static/css/*` | Stylesheets |
| `/static/js/*` | JavaScript files |
| `/static/img/*` | Images |
| `/static/favicon.ico` | Favicon |

## External API Integration

### Groupie Tracker API

The application integrates with the official Groupie Tracker API:

**Base URL:** `https://groupietrackers.herokuapp.com/api`

**Endpoints:**
- `GET /artists` - Returns array of artist objects
- `GET /relation` - Returns concert relations data

**API Models:**

```go
// External API artist structure
type APIArtist struct {
    ID           int      `json:"id"`
    Name         string   `json:"name"`
    Members      []string `json:"members"`
    CreationYear int      `json:"creationDate"`  // Note: different field name
    FirstAlbum   string   `json:"firstAlbum"`
    Image        string   `json:"image"`
}

// External API relations structure
type APIRelationIndex struct {
    ID             int                 `json:"id"`
    DatesLocations map[string][]string `json:"datesLocations"`
}
```

**Error Handling:**
- Timeout protection (10 seconds)
- HTTP status code validation
- JSON parsing error handling
- Concurrent fetch with error aggregation

## Template Data Structures

### Common Template Data

All templates receive a base data structure:

```go
type TemplateData struct {
    Title    string      // Page title
    ExtraCSS string      // Additional CSS file
    // ... page-specific fields
}
```

### Page-Specific Data

**Home Page:**
```go
struct {
    Title          string
    ExtraCSS       string
    Suggestions    []models.SearchSuggestion
    Artists        []models.Artist  // Featured artists
    TotalArtists   int
    TotalLocations int
}
```

**Artists Page:**
```go
struct {
    Title         string
    ExtraCSS      string
    Artists       []models.Artist
    Filters       models.Filters
    FilterOptions models.FilterOptions
    IsFiltered    bool
    ResultCount   int
}
```

**Search Page:**
```go
struct {
    Title         string
    ExtraCSS      string
    Query         string
    Artists       []models.Artist
    TotalResults  int
    Filters       models.Filters
    FilterOptions models.FilterOptions
}
```

## Error Handling

### Error Types

The application uses structured error handling:

```go
// HTTP errors are handled with appropriate status codes
// 400 - Bad Request (invalid form data)
// 404 - Not Found (invalid URLs, missing resources)
// 500 - Internal Server Error (application errors)
```

### Error Response Format

```go
type ErrorData struct {
    Title   string  // "Error"
    Status  int     // HTTP status code
    Message string  // User-friendly error message
}
```

### Recovery Mechanisms

- **Panic Recovery**: Middleware prevents application crashes
- **Template Fallback**: Plain text response if template fails
- **API Retry**: Exponential backoff for external API calls
- **Graceful Degradation**: Core functionality works without JavaScript

## Performance Optimizations

### Data Processing
- **Single-Pass Algorithm**: All data processed in one iteration
- **Efficient Aggregation**: Minimal memory allocations
- **Strategic Indexing**: O(1) lookups for common operations

### Memory Management
- **Copy-on-Read**: Thread-safe data access without locks on reads
- **Slice Pre-allocation**: Capacity hints for better memory usage
- **String Interning**: Shared strings for common values

### HTTP Performance
- **Template Caching**: Templates compiled once at startup
- **Static File Serving**: Direct file serving for assets
- **Connection Reuse**: HTTP client with connection pooling
- **Concurrent Processing**: Parallel API fetching

### Caching Strategy
- **Application-Level**: Full dataset cached in memory
- **Template-Level**: Compiled templates cached
- **Suggestion-Level**: Search suggestions pre-computed
- **Index-Level**: Fast lookup maps for common queries

This API documentation provides a comprehensive overview of the application's internal structure and data models. For implementation details, refer to the inline documentation in the source code.