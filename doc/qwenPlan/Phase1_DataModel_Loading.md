# Phase 1: Data Model & Loading Overhaul - Implementation Guide

## Overview
This phase focuses on simplifying the core data structures and loading process by removing cached fields, implementing helper methods, and streamlining the data loading pipeline. The goal is to create leaner structs with computed values rather than stored caches.

## Step-by-Step Implementation

### Step 1: Update Artist Struct
**File to modify:** `internal/data/models.go`

**Before:**
```go
type Artist struct {
	ID              int
	Name            string
	Slug            string // URL-friendly identifier (e.g., "queen")
	Members         []string
	CreationYear    int
	FirstAlbum      string
	Image           string
	Concerts        []Concert
	DatesAtLocation map[string][]string // Cached - TO BE REMOVED
	ConcertCount    int                 // Cached - TO BE REMOVED
	Countries       []string            // Cached - TO BE REMOVED
	MemberCount     int                 // Cached - TO BE REMOVED
	FirstAlbumYear  int                 // Cached - TO BE REMOVED
}
```

**After:**
```go
type Artist struct {
	ID           int
	Name         string
	Members      []string
	CreationYear int
	FirstAlbum   string
	Image        string
	Concerts     []Concert
}
```

### Step 2: Add Helper Methods to Artist
**File to modify:** `internal/data/models.go` (or create `internal/data/artist_helpers.go`)

```go
// MemberCount returns the number of members in the band
func (a *Artist) MemberCount() int {
	return len(a.Members)
}

// ConcertCount returns the number of concerts the artist has
func (a *Artist) ConcertCount() int {
	return len(a.Concerts)
}

// FirstAlbumYear returns the parsed year from the first album date
func (a *Artist) FirstAlbumYear() int {
	return extractYearFromDate(a.FirstAlbum)
}

// Countries returns all unique countries where the artist performed
func (a *Artist) Countries() []string {
	countries := make(map[string]bool)
	for _, concert := range a.Concerts {
		country := extractCountryFromLocation(concert.Location)
		if country != "" {
			countries[country] = true
		}
	}
	
	result := make([]string, 0, len(countries))
	for country := range countries {
		result = append(result, country)
	}
	sort.Strings(result)
	return result
}

// Slug returns a URL-friendly identifier for the artist
func (a *Artist) Slug() string {
	return createSlug(a.Name)
}

// Parse concert dates once into time.Time
type Concert struct {
	Date     time.Time // Changed from string to time.Time
	Location string
}
```

### Step 3: Update Location Struct
**File to modify:** `internal/data/models.go`

**Before:**
```go
type Location struct {
	Name          string
	Slug          string // URL-friendly identifier (e.g., "london-uk")
	Country       string // Display-ready country extracted from the slug
	Artists       []ArtistAtLocation
	ArtistCount   int // Cached - TO BE REMOVED
	TotalConcerts int // Cached - TO BE REMOVED
	EarliestYear  int
	LatestYear    int
}
```

**After:**
```go
type Location struct {
	Name     string
	Concerts []Concert // Simplified - store all concerts directly
}
```

### Step 4: Add Helper Methods to Location
**File to modify:** `internal/data/models.go` (or create `internal/data/location_helpers.go`)

```go
// ArtistCount returns the number of unique artists that performed at this location
func (l *Location) ArtistCount() int {
	artistSet := make(map[int]bool)
	for _, concert := range l.Concerts {
		artistSet[concert.ArtistID] = true
	}
	return len(artistSet)
}

// TotalConcerts returns the total number of concerts at this location
func (l *Location) TotalConcerts() int {
	return len(l.Concerts)
}

// YearRange returns the earliest and latest concert years at this location
func (l *Location) YearRange() (int, int) {
	if len(l.Concerts) == 0 {
		return 0, 0
	}
	
	minYear, maxYear := 9999, 0
	for _, concert := range l.Concerts {
		year := concert.Date.Year()
		if year < minYear {
			minYear = year
		}
		if year > maxYear {
			maxYear = year
		}
	}
	
	if minYear == 9999 {
		return 0, 0
	}
	return minYear, maxYear
}

// Slug returns a URL-friendly identifier for the location
func (l *Location) Slug() string {
	return createSlug(l.Name)
}

// Country returns the country extracted from the location name
func (l *Location) Country() string {
	return extractCountryFromLocation(l.Name)
}

// ArtistsAtLocation returns artists and their concert counts at this location
func (l *Location) ArtistsAtLocation() []ArtistAtLocation {
	counts := make(map[int]int)
	for _, concert := range l.Concerts {
		counts[concert.ArtistID]++
	}
	
	result := make([]ArtistAtLocation, 0, len(counts))
	for artistID, count := range counts {
		// In a real implementation, you'd need to get the artist from the store
		// For now, we'll just include the ID and count
		result = append(result, ArtistAtLocation{
			ArtistID:     artistID,
			ConcertCount: count,
		})
	}
	
	// Sort by concert count (descending), then by artist name
	sort.Slice(result, func(i, j int) bool {
		if result[i].ConcertCount != result[j].ConcertCount {
			return result[i].ConcertCount > result[j].ConcertCount
		}
		// For actual sorting by artist name, we would need the artist objects
		return result[i].ArtistID < result[j].ArtistID
	})
	
	return result
}
```

### Step 5: Update Store Struct to Remove Cached Fields
**File to modify:** `internal/data/store.go`

**Before:**
```go
type Store struct {
	apiClient *api.Client
	withCache bool
	
	cacheEnabled bool
	
	// Core data collections - immutable after Load() completes, safe for concurrent reads
	artists         []*Artist             // All artists sorted alphabetically by name
	artistsByID     map[int]*Artist       // O(1) lookup by artist ID
	artistsBySlug   map[string]*Artist    // O(1) lookup by URL-friendly slug (e.g., "pink-floyd")
	artistPositions map[int]int           // Maps artist ID to its index in the sorted artists slice (for navigation)
	locations       []Location            // All concert locations aggregated from artist data
	locationsBySlug map[string]Location   // O(1) lookup by location slug (e.g., "london-uk")
	appStats        AppStats              // Precomputed statistics (total artists, locations, members, concerts, etc.)
	suggestions     []SearchSuggestion    // Precomputed search suggestions for autocomplete (artist names, members, locations)
	artistFilters   ArtistFilterOptions   // Available filter values (creation years, album years, member counts, countries)
	locationFilters LocationFilterOptions // Available location filter values (concert ranges, year ranges, countries)
	
	// Search result cache (LRU-style) - to be removed
	searchCacheMu   sync.Mutex           // Mutex protects concurrent access to cache maps
	searchCache     map[string][]*Artist // Maps normalized query strings to cached result slices
	searchOrder     []string             // Tracks query insertion order for LRU eviction
	searchCacheSize int                  // Maximum cache entries (50) before LRU eviction kicks in
	
	loadOnce sync.Once // Ensures Load() executes exactly once even if called concurrently
	loadErr  error     // Stores any error from the single Load() execution for return to all callers
}
```

**After:**
```go
type Store struct {
	apiClient *api.Client
	withCache bool
	
	cacheEnabled bool
	
	// Simplified core data collections - immutable after Load() completes
	artists   []*Artist   // All artists sorted alphabetically by name
	locations []Location  // All concert locations
	
	// Simplified indexes - only the essential ones
	artistsByID     map[int]*Artist     // O(1) lookup by artist ID
	artistsBySlug   map[string]*Artist  // O(1) lookup by URL-friendly slug
	locationsBySlug map[string]Location // O(1) lookup by location slug
	
	appStats        AppStats            // Precomputed statistics
	suggestions     []SearchSuggestion  // Search suggestions
	artistFilters   ArtistFilterOptions   // Filter options
	locationFilters LocationFilterOptions // Filter options
	
	loadOnce sync.Once // Ensures Load() executes exactly once
	loadErr  error     // Stores any error from the single Load() execution
}
```

### Step 6: Update the Loading Pipeline to Sequential Processing
**File to modify:** `internal/data/store.go`

**Replace the complex concurrent loading with sequential processing:**

```go
// loadData performs the multi-stage data loading pipeline sequentially
// Stage 1: Fetch artists and relations sequentially
// Stage 2: Transform raw API data into rich domain models with computed fields
// Stage 3: Optionally cache images
// Stage 4: Build indexes and metadata
func (s *Store) loadData(ctx context.Context) error {
	// Stage 1: Sequential API fetching
	artistsData, err := s.apiClient.FetchArtists(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch artists: %w", err)
	}
	
	relationsData, err := s.apiClient.FetchRelations(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch relations: %w", err)
	}

	// Stage 2: Transform raw API models into rich domain models with computed fields
	artists := s.processArtists(artistsData, relationsData)

	// Stage 3: Optional image caching
	var cachedImages, downloadedImages int
	if s.withCache {
		var cacheEnabled bool
		cacheEnabled, cachedImages, downloadedImages = s.cacheImages(artists)
		s.cacheEnabled = cacheEnabled
	} else {
		s.cacheEnabled = false
	}

	// Stage 4: Build indexes and metadata
	// This includes building lookup maps and calculating filter options
	s.artists = artists
	s.artistsByID, s.artistsBySlug = s.createArtistIndexes(artists)
	s.locations = s.createLocationsData(artists) // This will process all concerts
	s.locationsBySlug = s.createLocationIndexes(s.locations)
	s.artistFilters = s.calculateArtistFilterOptions(artists)
	s.locationFilters = s.calculateLocationFilterOptions(s.locations)
	s.suggestions = s.generateSearchSuggestions(artists)
	s.appStats = s.calculateStats(artists, s.locations, cachedImages, downloadedImages)

	return nil
}
```

### Step 7: Update processArtists to Use Time Parsing
**File to modify:** `internal/data/store.go`

```go
// addConcertData enriches artists with concert information from API relations.
func (s *Store) addConcertData(artists []*Artist, apiRelations api.Relation) []*Artist {
	// Index relations by artist ID for efficient lookup
	relationMap := make(map[int]api.RelationIndex)
	for _, rel := range apiRelations.Index {
		relationMap[rel.ID] = rel
	}

	// Add concert data to each artist
	for i := range artists {
		artist := artists[i]

		if rel, exists := relationMap[artist.ID]; exists {
			// Process each location and its dates
			for location, dates := range rel.DatesLocations {
				normalizedLoc := normalizeLocation(location)
				
				for _, dateStr := range dates {
					// Parse the date string into time.Time
					parsedDate, err := time.Parse("02-01-2006", dateStr) // DD-MM-YYYY format
					if err != nil {
						// Fallback to YYYY format if needed
						parsedDate, _ = time.Parse("2006", dateStr)
					}

					artist.Concerts = append(artist.Concerts, Concert{
						Date:     parsedDate,
						Location: normalizedLoc,
					})
				}
			}
		}
	}

	// Sort artists by name for consistent display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}
```

### Step 8: Simplify Create Index Functions
**File to modify:** `internal/data/store.go`

```go
// createArtistIndexes builds lookup maps for artists by ID and slug.
func (s *Store) createArtistIndexes(artists []*Artist) (map[int]*Artist, map[string]*Artist) {
	artistsByID := make(map[int]*Artist, len(artists))
	artistsBySlug := make(map[string]*Artist, len(artists))

	for _, artist := range artists {
		artistsByID[artist.ID] = artist
		artistsBySlug[artist.Slug()] = artist
	}

	return artistsByID, artistsBySlug
}

// createLocationIndexes creates the slug-based lookup map for locations.
func (s *Store) createLocationIndexes(locations []Location) map[string]Location {
	locationsBySlug := make(map[string]Location, len(locations))
	for _, location := range locations {
		locationsBySlug[location.Slug()] = location
	}
	return locationsBySlug
}
```

### Step 9: Precompile Regular Expressions
**File to modify:** Add at package level in `internal/data/store.go`

```go
// Precompile regular expressions at package level
var (
	slugRegex        *regexp.Regexp
	countryExtractionRegex *regexp.Regexp
	
	// Use sync.Once to ensure compilation happens only once
	regexOnce sync.Once
)

func initRegex() {
	regexOnce.Do(func() {
		slugRegex = regexp.MustCompile(`[^a-z0-9]+`)
		countryExtractionRegex = regexp.MustCompile(`[^a-z0-9-]+`)
	})
}

// createSlug converts display names into URL-friendly slugs.
func createSlug(name string) string {
	// Ensure regex is initialized
	initRegex()
	
	slug := slugRegex.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}
```

### Step 10: Update Data Access Methods
Update all methods in the Store to use helper methods instead of cached values:

```go
// Example of updating a method to use helper instead of cached value
func (s *Store) Artists() []*Artist {
	// Return the stored artists (now using helper methods to get derived values)
	return s.artists
}

// When you need specific calculations, use the helper methods
func (s *Store) GetArtistWithDetails(id int) (*Artist, bool) {
	artist, ok := s.artistsByID[id]
	if !ok {
		return nil, false
	}
	
	// The caller can now use helper methods like:
	// artist.MemberCount(), artist.ConcertCount(), etc.
	return artist, true
}
```

## Testing Strategy for Phase 1
1. Update existing unit tests to reflect new helper methods instead of cached fields
2. Ensure all existing functionality works as before
3. Verify performance hasn't regressed significantly (helper methods should be fast)
4. Test all access patterns that previously used cached values

## Rollout Considerations
- This is a breaking change that will require updates to all calling code
- Start with internal refactoring of data layer before updating web layer
- Thoroughly test all functionality after changes
- Update documentation and comments to reflect new approach