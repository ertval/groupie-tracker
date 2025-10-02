# Phase 3: Search & Suggestion Refactor - Implementation Guide

## Overview
This phase focuses on simplifying the search and suggestion system by removing complex caching, implementing a normalized token index, and consolidating matching logic. The goal is to create a cleaner, more maintainable search system that's easier to understand and modify.

## Step-by-Step Implementation

### Step 1: Remove Complex Caching System
**File to modify:** `internal/data/searches.go` and `internal/data/store.go`

**Remove these fields from Store struct:**
```go
// Remove these fields from the Store struct in store.go:
// searchCacheMu   sync.Mutex
// searchCache     map[string][]*Artist  
// searchOrder     []string
// searchCacheSize int
```

### Step 2: Add Token Index to Artist Model
**File to modify:** `internal/data/models.go`

```go
// Add a field to the Artist struct to store searchable tokens
type Artist struct {
	ID           int
	Name         string
	Members      []string
	CreationYear int
	FirstAlbum   string
	Image        string
	Concerts     []Concert
	
	// Add cached searchable tokens for efficient lookup
	// These are computed during loading and used for search
	searchTokens []string
}

// Add a similar field to Location
type Location struct {
	Name     string
	Concerts []Concert
	
	// Add cached searchable tokens for efficient lookup
	searchTokens []string
}
```

### Step 3: Create Search Utilities
**File to create:** `internal/data/search_utils.go`

```go
import (
	"regexp"
	"sort"
	"strings"
)

// Global compiled regex for tokenization
var (
	tokenSplitRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

// normalizeText converts text to lowercase and removes special characters for search
func normalizeText(text string) string {
	return strings.ToLower(text)
}

// tokenizeString creates search tokens from a string
func tokenizeString(text string) []string {
	if text == "" {
		return []string{}
	}
	
	// Split on non-alphanumeric characters and normalize
	tokens := tokenSplitRegex.Split(text, -1)
	
	var result []string
	for _, token := range tokens {
		normalized := normalizeText(strings.TrimSpace(token))
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	
	return result
}

// tokenizeArtist extracts searchable tokens from an artist
func tokenizeArtist(artist *Artist) []string {
	var tokens []string
	
	// Add name tokens
	tokens = append(tokens, tokenizeString(artist.Name)...)
	
	// Add member tokens
	for _, member := range artist.Members {
		tokens = append(tokens, tokenizeString(member)...)
	}
	
	// Add creation year as tokens
	tokens = append(tokens, tokenizeString(fmt.Sprintf("%d", artist.CreationYear))...)
	
	// Add first album tokens
	tokens = append(tokens, tokenizeString(artist.FirstAlbum)...)
	
	// Add country tokens
	for _, country := range artist.Countries() {
		tokens = append(tokens, tokenizeString(country)...)
	}
	
	// Add location tokens from concerts
	for _, concert := range artist.Concerts {
		tokens = append(tokens, tokenizeString(concert.Location)...)
	}
	
	// Remove duplicates and return
	return removeDuplicates(tokens)
}

// tokenizeLocation extracts searchable tokens from a location
func tokenizeLocation(location Location) []string {
	var tokens []string
	
	// Add location name tokens
	tokens = append(tokens, tokenizeString(location.Name)...)
	
	// Add country tokens
	tokens = append(tokens, tokenizeString(location.Country())...)
	
	// Remove duplicates and return
	return removeDuplicates(tokens)
}

// removeDuplicates removes duplicate tokens from a slice
func removeDuplicates(tokens []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, token := range tokens {
		if !seen[token] && token != "" {
			seen[token] = true
			result = append(result, token)
		}
	}
	
	return result
}

// containsToken checks if a token exists in a slice of tokens
func containsToken(tokens []string, token string) bool {
	for _, t := range tokens {
		if t == token {
			return true
		}
	}
	return false
}
```

### Step 4: Precompute Search Tokens During Loading
**File to modify:** `internal/data/store.go`

Update the `addConcertData` or related function to populate search tokens:

```go
// Update processArtists to include token computation
func (s *Store) processArtists(apiArtists []api.Artist, apiRelations api.Relation) []*Artist {
	artists := s.transformAPIArtists(apiArtists)
	artists = s.addConcertData(artists, apiRelations)
	
	// Compute search tokens for all artists
	for i := range artists {
		artist := artists[i]
		artist.searchTokens = tokenizeArtist(artist)
	}
	
	return artists
}

// Update createLocations to include token computation
func (s *Store) createLocations(artists []*Artist) []Location {
	// Build location map as before
	locationMap := make(map[string]*Location)
	artistConcertCount := make(map[string]map[int]int)

	for _, artist := range artists {
		for _, concert := range artist.Concerts {
			// Initialize location if not exists
			if _, exists := locationMap[concert.Location]; !exists {
				locationMap[concert.Location] = &Location{
					Name:     concert.Location,
					Concerts: []Concert{}, // Will be populated
				}
				artistConcertCount[concert.Location] = make(map[int]int)
			}

			// Add concert to location
			locationMap[concert.Location].Concerts = append(locationMap[concert.Location].Concerts, concert)
			artistConcertCount[concert.Location][artist.ID]++
		}
	}

	// Convert map to slice and compute tokens
	locations := make([]Location, 0, len(locationMap))
	for _, loc := range locationMap {
		// Compute search tokens for location
		loc.searchTokens = tokenizeLocation(*loc)
		locations = append(locations, *loc)
	}

	// Sort by concert count as before
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].TotalConcerts() > locations[j].TotalConcerts()
	})

	return locations
}
```

### Step 5: Simplify the SearchArtists Function
**File to modify:** `internal/data/searches.go`

```go
// Replace the existing SearchArtists function with a simplified version
// SearchArtists performs full-text search across artist names, members, and metadata with optional filtering.
// Uses a simplified approach with precomputed tokens instead of complex caching.
func (s *Store) SearchArtists(query string, filters ArtistFilterParams) SearchResult {
	normalizedQuery := normalizeSearchQuery(query)
	
	// Get all artists (apply filters first to reduce search space)
	artists := s.FilterArtists(filters)
	
	var matchingArtists []*Artist
	
	if normalizedQuery == "" {
		// If no query, return all filtered artists
		matchingArtists = artists
	} else {
		// Split query into tokens for comprehensive matching
		queryTokens := tokenizeString(normalizedQuery)
		
		for _, artist := range artists {
			if matchesSearchQuery(artist, queryTokens) {
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	return SearchResult{
		Artists:      matchingArtists,
		Query:        query,
		TotalResults: len(matchingArtists),
	}
}

// matchesSearchQuery determines if an artist matches the search query tokens.
// Uses precomputed tokens for efficient matching.
func matchesSearchQuery(artist *Artist, queryTokens []string) bool {
	if len(queryTokens) == 0 {
		return true
	}

	// Check if any query token matches any of the artist's tokens
	for _, queryToken := range queryTokens {
		if containsToken(artist.searchTokens, queryToken) {
			return true
		}
	}

	return false
}

// normalizeSearchQuery standardizes query strings for case-insensitive comparison.
func normalizeSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}
```

### Step 6: Simplify Suggestion System
**File to modify:** `internal/data/searches.go`

```go
// Replace the suggestion system with a simpler approach
// Instead of complex SearchSuggestionType, use a simpler model
type SimpleSuggestion struct {
	Text        string `json:"text"`
	Category    string `json:"category"`  // "artist", "member", "location", "year"
	URL         string `json:"url"`
	ArtistID    int    `json:"artistId"`
}

// Simplified search suggestions generation
func (s *Store) generateSearchSuggestions(artists []*Artist) []SimpleSuggestion {
	var suggestions []SimpleSuggestion
	
	for _, artist := range artists {
		// Add artist name suggestion
		suggestions = append(suggestions, SimpleSuggestion{
			Text:     artist.Name,
			Category: "artist",
			URL:      "/artists/" + artist.Slug(),
			ArtistID: artist.ID,
		})
		
		// Add member name suggestions
		for _, member := range artist.Members {
			suggestions = append(suggestions, SimpleSuggestion{
				Text:     member,
				Category: "member",
				URL:      "/search?q=" + member,
				ArtistID: artist.ID,
			})
		}
		
		// Add location suggestions from concerts
		for _, concert := range artist.Concerts {
			// Avoid duplicate location suggestions
			locationAdded := false
			for _, existing := range suggestions {
				if existing.Text == concert.Location && existing.Category == "location" {
					locationAdded = true
					break
				}
			}
			
			if !locationAdded {
				suggestions = append(suggestions, SimpleSuggestion{
					Text:     concert.Location,
					Category: "location",
					URL:      "/search?q=" + concert.Location,
					ArtistID: 0, // No specific artist for location suggestions
				})
			}
		}
		
		// Add creation year suggestion
		yearStr := fmt.Sprintf("%d", artist.CreationYear)
		suggestions = append(suggestions, SimpleSuggestion{
			Text:     yearStr,
			Category: "year",
			URL:      "/search?q=" + yearStr,
			ArtistID: 0,
		})
		
		// Add first album suggestion
		if artist.FirstAlbum != "" {
			suggestions = append(suggestions, SimpleSuggestion{
				Text:     artist.FirstAlbum,
				Category: "album",
				URL:      "/search?q=" + artist.FirstAlbum,
				ArtistID: 0,
			})
		}
	}
	
	// Sort suggestions by category and text
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Category != suggestions[j].Category {
			return suggestions[i].Category < suggestions[j].Category
		}
		return suggestions[i].Text < suggestions[j].Text
	})
	
	return suggestions
}

// Simplified suggestion filtering
func (s *Store) FilterSearchSuggestions(query string, maxResults int) []SimpleSuggestion {
	normalizedQuery := normalizeSearchQuery(query)
	if normalizedQuery == "" {
		return []SimpleSuggestion{}
	}

	if maxResults <= 0 {
		maxResults = 15 // Default limit
	}

	// Filter suggestions by query
	var matches []SimpleSuggestion
	for _, suggestion := range s.suggestions {
		if strings.Contains(normalizeSearchQuery(suggestion.Text), normalizedQuery) {
			matches = append(matches, suggestion)
			if len(matches) >= maxResults {
				break
			}
		}
	}

	return matches
}
```

### Step 7: Update Web Layer to Use Simplified Search
**File to modify:** `internal/web/handlers.go`

```go
// Update the Search handler to work with simplified search API
func (app *App) Search(w http.ResponseWriter, r *http.Request) {
	// Validate path using centralized utility
	if !app.validateExactPath(w, r, "/search") {
		return
	}

	var searchQuery string
	var appliedFilters data.ArtistFilterParams
	var searchResults data.SearchResult

	// Handle search submission
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}

		searchQuery = strings.TrimSpace(r.FormValue("q"))
		// Extract search term from datalist format "Name - type" if applicable
		searchQuery = extractSearchTerm(searchQuery)
		appliedFilters = parseArtistFilterParams(r)

		// Use simplified search API - pass filters separately
		searchResults = app.store.SearchArtists(searchQuery, appliedFilters)
	}

	filterOptions := app.store.GetArtistFilterOptions()

	// Generate all search suggestions for datalist
	allSuggestions := app.store.GenerateAllSearchSuggestions()

	dataOutput := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Suggestions    []data.SimpleSuggestion // Updated type
		Query          string
		Results        data.SearchResult
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsSearch       bool
	}{
		Title:          "Search",
		ExtraCSS:       "search.css",
		ExtraJS:        "",
		Suggestions:    allSuggestions,
		Query:          searchQuery,
		Results:        searchResults,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsSearch:       r.Method == http.MethodPost && searchQuery != "",
	}

	app.render(w, r, "search.tmpl", dataOutput)
}

// Update the SuggestionsAPI handler
func (app *App) SuggestionsAPI(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	// Use simplified filtering with reasonable limits
	const maxSuggestions = 15
	matchingSuggestions := app.store.FilterSearchSuggestions(query, maxSuggestions)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchingSuggestions)
}
```

### Step 8: Update Search Result Structure
**File to modify:** `internal/data/models.go`

```go
// Update SearchResult to work with simplified search
type SearchResult struct {
	Artists      []*Artist `json:"artists"`
	Query        string    `json:"query"`
	TotalResults int       `json:"totalResults"`
}
```

### Step 9: Update Tests for Simplified Search
**File to modify:** `internal/data/data_test.go`

```go
// Update the TestSearchArtists function to work with simplified API
func TestSearchArtists(t *testing.T) {
	store := createSearchStore()

	tests := []struct {
		name        string
		query       string
		filters     ArtistFilterParams
		expectedIDs []int
	}{
		{
			name:        "Empty query with no filters returns all artists",
			query:       "",
			filters:     ArtistFilterParams{},
			expectedIDs: []int{1, 2, 3},
		},
		{
			name:        "Artist name search - case insensitive",
			query:       "queen",
			filters:     ArtistFilterParams{},
			expectedIDs: []int{1},
		},
		{
			name:        "Member name search",
			query:       "Freddie Mercury",
			filters:     ArtistFilterParams{},
			expectedIDs: []int{1},
		},
		{
			name:  "Query with filters",
			query: "Phil",
			filters: ArtistFilterParams{
				CreationYear: RangeFilter[int]{Min: intPtr(1980), Max: intPtr(1985)},
			},
			expectedIDs: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.SearchArtists(tt.query, tt.filters)

			if len(result.Artists) != len(tt.expectedIDs) {
				t.Fatalf("SearchArtists(%q) returned %d artists, expected %d",
					tt.query, len(result.Artists), len(tt.expectedIDs))
			}

			foundIDs := make(map[int]bool)
			for _, artist := range result.Artists {
				foundIDs[artist.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("Missing expected artist ID %d", expectedID)
				}
			}

			if result.Query != tt.query {
				t.Errorf("Query mismatch: got %q, want %q", result.Query, tt.query)
			}

			if result.TotalResults != len(result.Artists) {
				t.Errorf("TotalResults %d mismatch actual count %d",
					result.TotalResults, len(result.Artists))
			}
		})
	}
}
```

### Step 10: Remove Old Search-Related Types and Functions
Remove the following from `models.go`:
- `SearchSuggestionType` enum
- `SearchSuggestion` struct 
- `SearchParams` struct
- Any other complex search-related types that are no longer needed

## Testing Strategy for Phase 3
1. Update all existing search tests to work with the new simplified API
2. Test search tokenization with various inputs (special characters, numbers, etc.)
3. Verify search performance hasn't regressed significantly
4. Test that filtering still works in combination with search
5. Ensure suggestion functionality works as expected
6. Test edge cases like empty queries, special characters, and long queries

## Rollout Considerations
- The simplified search system should be more maintainable and faster
- All existing search functionality should be preserved
- The API changes might require updates to the web layer
- Consider backward compatibility if other systems depend on complex search types
- Update any documentation that references the old complex search structures