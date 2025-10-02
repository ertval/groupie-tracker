# Phase 6: Testing & Validation - Implementation Guide

## Overview
This phase focuses on comprehensive testing and validation of the refactored codebase to ensure all functionality works as expected and performance is maintained. The goal is to verify that the simplified architecture doesn't introduce regressions and that all new features work correctly.

## Step-by-Step Implementation

### Step 1: Update Unit Tests for New Data Layer
**File to modify:** `internal/data/data_test.go`

```go
package data

import (
	"strings"
	"testing"
)

// TestArtistHelperMethods tests the new helper methods on Artist struct
func TestArtistHelperMethods(t *testing.T) {
	artist := &Artist{
		Name:         "Test Artist",
		Members:      []string{"Member 1", "Member 2", "Member 3"},
		CreationYear: 2000,
		FirstAlbum:   "01-01-2005",
		Concerts: []Concert{
			{Date: parseDate("01-01-2010"), Location: "london-uk"},
			{Date: parseDate("01-01-2011"), Location: "new-york-usa"},
		},
	}
	
	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{
			name:     "MemberCount",
			expected: 3,
			actual:   artist.MemberCount(),
		},
		{
			name:     "ConcertCount",
			expected: 2,
			actual:   artist.ConcertCount(),
		},
		{
			name:     "FirstAlbumYear",
			expected: 2005,
			actual:   artist.FirstAlbumYear(),
		},
		{
			name:     "Slug",
			expected: "test-artist",
			actual:   artist.Slug(),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s() = %v, want %v", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

// TestLocationHelperMethods tests the new helper methods on Location struct
func TestLocationHelperMethods(t *testing.T) {
	location := Location{
		Name: "london-uk",
		Concerts: []Concert{
			{Date: parseDate("01-01-2010"), Location: "london-uk", ArtistID: 1},
			{Date: parseDate("01-01-2011"), Location: "london-uk", ArtistID: 2},
			{Date: parseDate("01-01-2010"), Location: "london-uk", ArtistID: 1}, // Same artist, different date
		},
	}
	
	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{
			name:     "TotalConcerts",
			expected: 3,
			actual:   location.TotalConcerts(),
		},
		{
			name:     "ArtistCount",
			expected: 2,
			actual:   location.ArtistCount(),
		},
		{
			name:     "Slug",
			expected: "london-uk",
			actual:   location.Slug(),
		},
		{
			name:     "Country",
			expected: "UK",
			actual:   location.Country(),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s() = %v, want %v", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

// TestFilterArtists tests all artist filtering functionality with the new functional approach
func TestFilterArtists(t *testing.T) {
	store := createTestStore(t)

	tests := []struct {
		name    string
		params  ArtistFilterParams
		wantMin int
		check   func(t *testing.T, artists []*Artist)
	}{
		{
			name: "Filter by creation year range 1995-2000",
			params: ArtistFilterParams{
				CreationYear: RangeFilter[int]{Min: intPtr(1995), Max: intPtr(2000)},
			},
			wantMin: 7,
			check: func(t *testing.T, artists []*Artist) {
				for _, artist := range artists {
					if artist.CreationYear < 1995 || artist.CreationYear > 2000 {
						t.Errorf("Artist %s has creation year %d which is outside the range [1995, 2000]", 
							artist.Name, artist.CreationYear)
					}
				}
			},
		},
		{
			name: "Filter by member counts",
			params: ArtistFilterParams{
				MemberCounts: []int{1, 5}, // Solo artists or 5-member bands
			},
			check: func(t *testing.T, artists []*Artist) {
				for _, artist := range artists {
					memberCount := artist.MemberCount()
					if memberCount != 1 && memberCount != 5 {
						t.Errorf("Artist %s has %d members, expected 1 or 5", artist.Name, memberCount)
					}
				}
			},
		},
		{
			name: "Filter by countries",
			params: ArtistFilterParams{
				Countries: []string{"USA", "UK"},
			},
			wantMin: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := store.FilterArtists(tt.params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterArtists() returned %d artists, want at least %d", len(results), tt.wantMin)
				t.Logf("Got artists: %v", getArtistNames(results))
			}

			if tt.check != nil {
				tt.check(t, results)
			}
		})
	}
}

// TestFilterLocations tests location filtering functionality
func TestFilterLocations(t *testing.T) {
	store := createLocationStore(t)

	tests := []struct {
		name    string
		params  LocationFilterParams
		wantMin int
	}{
		{
			name: "Filter by concert count range",
			params: LocationFilterParams{
				ConcertCount: RangeFilter[int]{Min: intPtr(1), Max: intPtr(10)},
			},
			wantMin: 1,
		},
		{
			name: "Filter by artist count range",
			params: LocationFilterParams{
				ArtistCount: RangeFilter[int]{Min: intPtr(1), Max: intPtr(5)},
			},
			wantMin: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := store.FilterLocations(tt.params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterLocations() returned %d locations, want at least %d", len(results), tt.wantMin)
			}
		})
	}
}

// TestSearchArtists tests artist search functionality with various queries and filters
func TestSearchArtists(t *testing.T) {
	store := createSearchStore()

	tests := []struct {
		name        string
		query       string
		filters     ArtistFilterParams
		expectedIDs []int
	}{
		{
			name:        "Empty query returns all artists",
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

// TestSearchSuggestions tests the search suggestions functionality
func TestSearchSuggestions(t *testing.T) {
	store := createTestStore()

	tests := []struct {
		name       string
		query      string
		maxResults int
		validate   func(t *testing.T, suggestions []SimpleSuggestion)
	}{
		{
			name:       "Returns suggestions for 'queen'",
			query:      "queen",
			maxResults: 5,
			validate: func(t *testing.T, suggestions []SimpleSuggestion) {
				if len(suggestions) == 0 {
					t.Fatal("Expected suggestions for query 'queen'")
				}
				if !strings.Contains(strings.ToLower(suggestions[0].Text), "queen") {
					t.Errorf("Expected first suggestion to contain 'queen', got %q", suggestions[0].Text)
				}
			},
		},
		{
			name:       "Respects max results limit",
			query:      "a",
			maxResults: 2,
			validate: func(t *testing.T, suggestions []SimpleSuggestion) {
				if len(suggestions) > 2 {
					t.Errorf("Expected at most 2 suggestions, got %d", len(suggestions))
				}
			},
		},
		{
			name:       "Empty query returns no suggestions",
			query:      "",
			maxResults: 5,
			validate: func(t *testing.T, suggestions []SimpleSuggestion) {
				if len(suggestions) != 0 {
					t.Errorf("Expected no suggestions for empty query, got %d", len(suggestions))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := store.FilterSearchSuggestions(tt.query, tt.maxResults)
			tt.validate(t, suggestions)
		})
	}
}

// Helper functions for creating test stores
func createTestStore(t *testing.T) *Store {
	t.Helper()

	mockArtists := []Artist{
		{
			ID:           1, 
			Name:         "Queen", 
			Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"}, 
			CreationYear: 1970, 
			FirstAlbum:   "14-07-1973",
			Concerts:     []Concert{{Date: parseDate("01-01-1980"), Location: "london-uk"}, {Date: parseDate("01-01-1981"), Location: "new-york-usa"}},
		},
		{
			ID:           2, 
			Name:         "Gorillaz", 
			Members:      []string{"Damon Albarn", "Jamie Hewlett"}, 
			CreationYear: 1998, 
			FirstAlbum:   "26-03-2001",
			Concerts:     []Concert{{Date: parseDate("01-01-2001"), Location: "london-uk"}},
		},
		{
			ID:           3, 
			Name:         "Travis Scott", 
			Members:      []string{"Jacques Berman Webster II"}, 
			CreationYear: 2008, 
			FirstAlbum:   "2015",
			Concerts:     []Concert{{Date: parseDate("01-01-2015"), Location: "houston-tx-usa"}},
		},
		// Add more test artists to reach the expected counts
		{
			ID: 4, Name: "SOJA", Members: []string{"Jacob Hemphill"}, CreationYear: 1997, FirstAlbum: "2000",
			Concerts: []Concert{{Date: parseDate("01-01-2000"), Location: "washington-usa"}},
		},
		{
			ID: 5, Name: "Mamonas Assassinas", Members: []string{"Dinho"}, CreationYear: 1995, FirstAlbum: "1995",
			Concerts: []Concert{{Date: parseDate("01-01-1995"), Location: "sao-paulo-brazil"}},
		},
		{
			ID: 6, Name: "Thirty Seconds to Mars", Members: []string{"Jared Leto"}, CreationYear: 1998, FirstAlbum: "2002",
			Concerts: []Concert{{Date: parseDate("01-01-2002"), Location: "los-angeles-ca-usa"}},
		},
		{
			ID: 7, Name: "Nickelback", Members: []string{"Chad Kroeger"}, CreationYear: 1995, FirstAlbum: "1996",
			Concerts: []Concert{{Date: parseDate("01-01-1996"), Location: "vancouver-ca"}},
		},
		{
			ID: 8, Name: "Linkin Park", Members: []string{"Chester Bennington"}, CreationYear: 1996, FirstAlbum: "2000",
			Concerts: []Concert{{Date: parseDate("01-01-2000"), Location: "los-angeles-ca-usa"}},
		},
		{
			ID: 9, Name: "Coldplay", Members: []string{"Chris Martin"}, CreationYear: 1996, FirstAlbum: "2000",
			Concerts: []Concert{{Date: parseDate("01-01-2000"), Location: "london-uk"}},
		},
		{
			ID: 10, Name: "Red Hot Chili Peppers", Members: []string{"Anthony Kiedis"}, CreationYear: 1982, FirstAlbum: "1991",
			Concerts: []Concert{{Date: parseDate("01-01-1991"), Location: "los-angeles-ca-usa"}},
		},
		{
			ID: 11, Name: "Aerosmith", Members: []string{"Steven Tyler"}, CreationYear: 1970, FirstAlbum: "1972",
			Concerts: []Concert{{Date: parseDate("01-01-1972"), Location: "boston-usa"}},
		},
		{
			ID: 12, Name: "AC/DC", Members: []string{"Bon Scott"}, CreationYear: 1973, FirstAlbum: "1975",
			Concerts: []Concert{{Date: parseDate("01-01-1975"), Location: "sydney-australia"}},
		},
		{
			ID: 13, Name: "U2", Members: []string{"Bono"}, CreationYear: 1976, FirstAlbum: "1980",
			Concerts: []Concert{{Date: parseDate("01-01-1980"), Location: "dublin-ireland"}},
		},
		{
			ID: 14, Name: "The Beatles", Members: []string{"John Lennon"}, CreationYear: 1960, FirstAlbum: "1963",
			Concerts: []Concert{{Date: parseDate("01-01-1963"), Location: "liverpool-uk"}},
		},
		{
			ID: 15, Name: "The Rolling Stones", Members: []string{"Mick Jagger"}, CreationYear: 1962, FirstAlbum: "1964",
			Concerts: []Concert{{Date: parseDate("01-01-1964"), Location: "london-uk"}},
		},
	}

	return NewStoreFromFixtures(mockArtists, nil)
}

func createSearchStore() *Store {
	artists := []Artist{
		{
			ID:           1,
			Name:         "Queen",
			Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Concerts: []Concert{
				{Date: parseDate("14-07-1973"), Location: "london-uk"},
				{Date: parseDate("12-12-1975"), Location: "paris-france"},
			},
		},
		{
			ID:           2,
			Name:         "Phil Collins",
			Members:      []string{"Phil Collins"},
			CreationYear: 1981,
			FirstAlbum:   "05-02-1981",
			Concerts: []Concert{
				{Date: parseDate("05-02-1981"), Location: "london-uk"},
				{Date: parseDate("10-10-1985"), Location: "new-york-usa"},
			},
		},
		{
			ID:           3,
			Name:         "Pink Floyd",
			Members:      []string{"David Gilmour", "Roger Waters"},
			CreationYear: 1965,
			FirstAlbum:   "05-08-1967",
			Concerts: []Concert{
				{Date: parseDate("05-08-1967"), Location: "london-uk"},
				{Date: parseDate("20-03-1973"), Location: "london-uk"},
				{Date: parseDate("15-05-1979"), Location: "dortmund-germany"},
			},
		},
	}

	return NewStoreFromFixtures(artists, nil)
}

func createLocationStore(t *testing.T) *Store {
	// Create a store with some locations for testing
	artists := []Artist{
		{
			ID:           1,
			Name:         "Test Artist 1",
			Members:      []string{"Member 1"},
			CreationYear: 2000,
			FirstAlbum:   "2005",
			Concerts: []Concert{
				{Date: parseDate("2010"), Location: "Location A", ArtistID: 1},
				{Date: parseDate("2011"), Location: "Location A", ArtistID: 1},
			},
		},
		{
			ID:           2,
			Name:         "Test Artist 2",
			Members:      []string{"Member 2"},
			CreationYear: 2005,
			FirstAlbum:   "2010",
			Concerts: []Concert{
				{Date: parseDate("2015"), Location: "Location B", ArtistID: 2},
				{Date: parseDate("2016"), Location: "Location C", ArtistID: 2},
			},
		},
	}

	return NewStoreFromFixtures(artists, nil)
}

// Helper functions
func getArtistNames(artists []*Artist) []string {
	names := make([]string, len(artists))
	for i, artist := range artists {
		names[i] = artist.Name
	}
	return names
}

func intPtr(i int) *int {
	return &i
}

// Helper to parse date for testing
func parseDate(dateStr string) time.Time {
	layout := "02-01-2006"
	if len(dateStr) == 4 { // Year only
		layout = "2006"
	}
	t, _ := time.Parse(layout, dateStr)
	return t
}
```

### Step 2: Create Web Layer Tests
**File to create:** `internal/web/web_test.go`

```go
package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"groupie-tracker/internal/data"
)

// TestHomeHandler tests the home page handler
func TestHomeHandler(t *testing.T) {
	app := createTestApp(t)
	
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	app.Home(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Home") {
		t.Error("Expected response to contain 'Home'")
	}
}

// TestArtistsHandler tests the artists page handler
func TestArtistsHandler(t *testing.T) {
	app := createTestApp(t)
	
	req := httptest.NewRequest("GET", "/artists", nil)
	rec := httptest.NewRecorder()
	
	app.Artists(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
}

// TestArtistDetailHandler tests the artist detail page handler
func TestArtistDetailHandler(t *testing.T) {
	app := createTestApp(t)
	
	// Test with a known artist slug
	req := httptest.NewRequest("GET", "/artists/queen", nil)
	rec := httptest.NewRecorder()
	
	app.ArtistDetail(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	
	// Test with unknown artist
	req = httptest.NewRequest("GET", "/artists/unknown-artist", nil)
	rec = httptest.NewRecorder()
	
	app.ArtistDetail(rec, req)
	
	resp = rec.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// TestLocationsHandler tests the locations page handler
func TestLocationsHandler(t *testing.T) {
	app := createTestApp(t)
	
	req := httptest.NewRequest("GET", "/locations", nil)
	rec := httptest.NewRecorder()
	
	app.Locations(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
}

// TestLocationDetailHandler tests the location detail page handler
func TestLocationDetailHandler(t *testing.T) {
	app := createTestApp(t)
	
	// Test with a known location slug
	req := httptest.NewRequest("GET", "/locations/london-uk", nil)
	rec := httptest.NewRecorder()
	
	app.LocationDetail(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	
	// Test with unknown location
	req = httptest.NewRequest("GET", "/locations/unknown-location", nil)
	rec = httptest.NewRecorder()
	
	app.LocationDetail(rec, req)
	
	resp = rec.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// TestSearchHandler tests the search page handler
func TestSearchHandler(t *testing.T) {
	app := createTestApp(t)
	
	// Test GET request
	req := httptest.NewRequest("GET", "/search", nil)
	rec := httptest.NewRecorder()
	
	app.Search(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	
	// Test POST request with search query
	form := url.Values{}
	form.Add("q", "queen")
	
	req = httptest.NewRequest("POST", "/search", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	
	app.Search(rec, req)
	
	resp = rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Queen") {
		t.Error("Expected response to contain 'Queen' after searching for 'queen'")
	}
}

// TestSuggestionsAPI tests the search suggestions API
func TestSuggestionsAPI(t *testing.T) {
	app := createTestApp(t)
	
	req := httptest.NewRequest("GET", "/api/suggestions?q=queen", nil)
	rec := httptest.NewRecorder()
	
	app.SuggestionsAPI(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
	
	var suggestions []data.SimpleSuggestion
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &suggestions); err != nil {
		t.Errorf("Could not unmarshal suggestions: %v", err)
	}
	
	// Should have at least one suggestion for "queen"
	if len(suggestions) == 0 {
		t.Error("Expected at least one suggestion for 'queen'")
	}
}

// TestMethodRestriction tests the method restriction middleware
func TestMethodRestriction(t *testing.T) {
	app := createTestApp(t)
	
	// Try accessing /artists with a disallowed method
	req := httptest.NewRequest("PUT", "/artists", nil)
	rec := httptest.NewRecorder()
	
	// Use the restrictMethod middleware as it would be in routes
	handler := app.restrictMethod(app.Artists, "GET", "POST")
	handler(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Allow") != "GET, POST" {
		t.Errorf("Expected Allow header 'GET, POST', got %s", resp.Header.Get("Allow"))
	}
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	app := createTestApp(t)
	
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	
	app.Health(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
	
	// Parse the response to ensure it has expected fields
	var health map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &health); err != nil {
		t.Errorf("Could not unmarshal health response: %v", err)
	}
	
	if status, ok := health["status"]; !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", status)
	}
}

// TestErrorHandlers tests the various error handling functions
func TestErrorHandlers(t *testing.T) {
	app := createTestApp(t)
	
	// Test 404 error
	req := httptest.NewRequest("GET", "/test-404", nil)
	rec := httptest.NewRecorder()
	
	app.NotFoundError(rec, req, "Test not found")
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
	
	// Test 500 error
	rec = httptest.NewRecorder()
	app.Error(rec, req, http.StatusInternalServerError, "Test error")
	
	resp = rec.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
	
	// Test 400 error
	rec = httptest.NewRecorder()
	app.BadRequestError(rec, req, "Test bad request")
	
	resp = rec.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// TestStaticFilesHandler tests the static file serving
func TestStaticFilesHandler(t *testing.T) {
	app := createTestApp(t)
	
	// Try accessing a non-existent static file
	req := httptest.NewRequest("GET", "/static/nonexistent.css", nil)
	rec := httptest.NewRecorder()
	
	app.StaticFiles(rec, req)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file, got %d", resp.StatusCode)
	}
	
	// Try accessing the favicon
	req = httptest.NewRequest("GET", "/favicon.ico", nil)
	rec = httptest.NewRecorder()
	
	app.StaticFiles(rec, req)
	
	// This might return 404 if favicon doesn't exist, which is expected
	resp = rec.Result()
	if resp.StatusCode != http.StatusNotFound {
		// If favicon exists, we'd expect 200; if not, 404 is fine
		if resp.StatusCode != http.StatusOK {
			t.Logf("Favicon request returned %d (may be expected if file doesn't exist)", resp.StatusCode)
		}
	}
}

// TestTemplateRendering tests the template rendering function
func (app *App) TestTemplateRendering(t *testing.T) {
	// Create a mock data structure
	mockData := struct {
		Title string
	}{
		Title: "Test Page",
	}
	
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	// Try to render a template that doesn't exist to test error handling
	app.render(rec, req, "nonexistent.tmpl", mockData)
	
	resp := rec.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for nonexistent template, got %d", resp.StatusCode)
	}
}

// Helper function to create a test app with fixtures
func createTestApp(t *testing.T) *App {
	// Create a mock store with test data
	mockArtists := []data.Artist{
		{
			ID:           1,
			Name:         "Queen",
			Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Concerts: []data.Concert{
				{Date: parseDate("14-07-1973"), Location: "london-uk"},
				{Date: parseDate("12-12-1975"), Location: "paris-france"},
			},
		},
		{
			ID:           2,
			Name:         "Gorillaz",
			Members:      []string{"Damon Albarn", "Jamie Hewlett"},
			CreationYear: 1998,
			FirstAlbum:   "26-03-2001",
			Concerts: []data.Concert{
				{Date: parseDate("26-03-2001"), Location: "london-uk"},
				{Date: parseDate("20-07-2005"), Location: "manchester-uk"},
			},
		},
		{
			ID:           3,
			Name:         "Travis Scott",
			Members:      []string{"Jacques Berman Webster II"},
			CreationYear: 2008,
			FirstAlbum:   "2015",
			Concerts: []data.Concert{
				{Date: parseDate("01-01-2015"), Location: "houston-tx-usa"},
				{Date: parseDate("15-06-2018"), Location: "los-angeles-ca-usa"},
			},
		},
	}
	
	// Create locations data
	mockLocations := []data.Location{
		{
			Name: "london-uk",
			Concerts: []data.Concert{
				{Date: parseDate("14-07-1973"), Location: "london-uk", ArtistID: 1},
				{Date: parseDate("26-03-2001"), Location: "london-uk", ArtistID: 2},
			},
		},
		{
			Name: "paris-france",
			Concerts: []data.Concert{
				{Date: parseDate("12-12-1975"), Location: "paris-france", ArtistID: 1},
			},
		},
	}
	
	store := data.NewStoreFromFixtures(mockArtists, mockLocations)
	
	// Create the app with the test store
	app := &App{
		store: store,
	}
	
	// Load templates
	app.loadTemplates()
	
	// Set up the handler chain for testing
	app.Handler = withMiddleware(app.createServeMux())
	
	return app
}

// Helper to parse date for testing
func parseDate(dateStr string) time.Time {
	layout := "02-01-2006"
	if len(dateStr) == 4 { // Year only
		layout = "2006"
	}
	t, _ := time.Parse(layout, dateStr)
	return t
}
```

### Step 3: Create Integration Tests
**File to create:** `tests/integration_test.go`

```go
package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/web"
)

// TestEndToEnd tests the complete application flow
func TestEndToEnd(t *testing.T) {
	// This test would require a mock API client for proper integration testing
	// For now, we'll test with fixtures
	
	// Create the application with test data
	app, err := web.NewAppWithFixtures() // This function would need to be created
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}
	
	// Create a test server
	ts := httptest.NewServer(app.Handler)
	defer ts.Close()
	
	// Test home page
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("Failed to get home page: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for home page, got %d", resp.StatusCode)
	}
	
	// Test artists page
	resp, err = http.Get(ts.URL + "/artists")
	if err != nil {
		t.Fatalf("Failed to get artists page: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for artists page, got %d", resp.StatusCode)
	}
	
	// Test locations page
	resp, err = http.Get(ts.URL + "/locations")
	if err != nil {
		t.Fatalf("Failed to get locations page: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for locations page, got %d", resp.StatusCode)
	}
	
	// Test search functionality
	formData := strings.NewReader("q=queen")
	resp, err = http.Post(ts.URL+"/search", "application/x-www-form-urlencoded", formData)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for search, got %d", resp.StatusCode)
	}
	
	// Verify search results contain the query term
	body := make([]byte, 10000)
	resp.Body.Read(body)
	if !strings.Contains(string(body), "Queen") {
		t.Error("Expected search results to contain 'Queen'")
	}
}

// TestAPIClientsIntegration tests the API client integration
func TestAPIClientsIntegration(t *testing.T) {
	// Test the API client with the actual external API (use with caution in CI)
	client := api.NewClient("https://groupietrackers.herokuapp.com", api.DefaultTimeout)
	
	// Test fetching artists
	artists, err := client.FetchArtists()
	if err != nil {
		t.Skipf("Skipping API integration test due to network error: %v", err)
	}
	
	if len(artists) == 0 {
		t.Error("Expected to fetch at least some artists from the API")
	}
	
	// Test fetching relations
	relations, err := client.FetchRelations()
	if err != nil {
		t.Skipf("Skipping API integration test due to network error: %v", err)
	}
	
	if len(relations.Index) == 0 {
		t.Error("Expected to fetch at least some relations from the API")
	}
}

// TestRouteAccess tests that all defined routes respond appropriately
func TestRouteAccess(t *testing.T) {
	app := createTestApp(t) // Uses fixtures
	ts := httptest.NewServer(app.Handler)
	defer ts.Close()
	
	routes := []struct {
		path string
	}{
		{"/"},
		{"/artists"},
		{"/locations"},
		{"/search"},
		{"/health"},
		{"/api/suggestions?q=test"},
		{"/static/"},
		{"/favicon.ico"},
	}
	
	for _, route := range routes {
		t.Run("route_"+strings.ReplaceAll(route.path, "/", "_"), func(t *testing.T) {
			resp, err := http.Get(ts.URL + route.path)
			if err != nil {
				t.Fatalf("Failed to access %s: %v", route.path, err)
			}
			
			// Most routes should return 200, 404, or 405 (method not allowed)
			if resp.StatusCode != http.StatusOK && 
			   resp.StatusCode != http.StatusNotFound && 
			   resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("Route %s returned unexpected status: %d", route.path, resp.StatusCode)
			}
		})
	}
}

// TestConcurrentAccess tests the application under concurrent load
func TestConcurrentAccess(t *testing.T) {
	app := createTestApp(t) // Uses fixtures
	ts := httptest.NewServer(app.Handler)
	defer ts.Close()
	
	// Create multiple concurrent requests
	numRequests := 10
	errs := make(chan error, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(requestNum int) {
			resp, err := http.Get(ts.URL + "/")
			if err != nil {
				errs <- fmt.Errorf("request %d failed: %v", requestNum, err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				errs <- fmt.Errorf("request %d returned status %d, expected 200", requestNum, resp.StatusCode)
				return
			}
			errs <- nil
		}(i)
	}
	
	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
}
```

### Step 4: Create Benchmark Tests
**File to create:** `internal/data/data_bench_test.go`

```go
package data

import (
	"testing"
)

// BenchmarkFilterArtists benchmarks the artist filtering performance
func BenchmarkFilterArtists(b *testing.B) {
	store := createBenchmarkStore(b)
	
	filters := ArtistFilterParams{
		CreationYear: RangeFilter[int]{Min: intPtr(1990), Max: intPtr(2000)},
		MemberCounts: []int{1, 2, 3, 4, 5},
		Countries:    []string{"USA", "UK", "Canada"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.FilterArtists(filters)
	}
}

// BenchmarkSearchArtists benchmarks the search performance
func BenchmarkSearchArtists(b *testing.B) {
	store := createBenchmarkStore(b)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.SearchArtists("queen", ArtistFilterParams{})
	}
}

// BenchmarkSearchWithFilters benchmarks search with filters
func BenchmarkSearchWithFilters(b *testing.B) {
	store := createBenchmarkStore(b)
	
	filters := ArtistFilterParams{
		CreationYear: RangeFilter[int]{Min: intPtr(1970), Max: intPtr(1980)},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.SearchArtists("rock", filters)
	}
}

// BenchmarkTokenization benchmarks the search tokenization process
func BenchmarkTokenizeArtist(b *testing.B) {
	artist := &Artist{
		Name:         "Queen",
		Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
		CreationYear: 1970,
		FirstAlbum:   "14-07-1973",
		Concerts:     make([]Concert, 50), // Large number of concerts for realistic testing
	}
	
	// Fill concerts with dummy data
	for i := range artist.Concerts {
		artist.Concerts[i] = Concert{
			Location: "location-" + string(rune(i)),
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenizeArtist(artist)
	}
}

// createBenchmarkStore creates a store with a larger dataset for performance testing
func createBenchmarkStore(b *testing.B) *Store {
	// Create a larger test dataset
	artists := make([]Artist, 100)
	for i := 0; i < 100; i++ {
		members := make([]string, 0)
		for j := 0; j < 5; j++ {
			members = append(members, "Member "+string(rune('A'+j)) + fmt.Sprintf("-%d", i))
		}
		
		concerts := make([]Concert, 0)
		for j := 0; j < 10; j++ {
			country := []string{"usa", "uk", "canada", "australia", "germany"}[j%5]
			city := fmt.Sprintf("city-%d-%d", i, j)
			concerts = append(concerts, Concert{
				Location: city + "-" + country,
			})
		}
		
		artists[i] = Artist{
			ID:           i,
			Name:         "Artist " + fmt.Sprintf("%d", i),
			Members:      members,
			CreationYear: 1970 + (i % 50),
			FirstAlbum:   fmt.Sprintf("01-01-%d", 1970+(i%50)),
			Concerts:     concerts,
		}
	}
	
	return NewStoreFromFixtures(artists, nil)
}
```

### Step 5: Update Main Test File
**File to modify:** `internal/data/fixtures.go` (add helper for testing)

```go
// Add to fixtures.go to support testing

// NewStoreFromFixturesWithPrecomputedTokens creates a Store populated with fixtures
// and precomputes search tokens for testing
func NewStoreFromFixturesWithPrecomputedTokens(artists []Artist, locations []Location) *Store {
	store := NewStoreFromFixtures(artists, locations)
	
	// Precompute search tokens for all artists
	for i := range store.artists {
		artist := store.artists[i]
		artist.searchTokens = tokenizeArtist(artist)
	}
	
	// Precompute search tokens for all locations
	for i := range store.locations {
		location := &store.locations[i]
		location.searchTokens = tokenizeLocation(*location)
	}
	
	return store
}
```

### Step 6: Add Quality Gate Scripts
**File to create:** Create a test script (as a Go file or shell script)

Create a Go script to run all tests:

```go
// File: test_runner.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("Running comprehensive test suite...")
	
	tests := []struct {
		name string
		cmd  string
		args []string
	}{
		{"Format check", "go", []string{"fmt", "./..."}},
		{"Vet check", "go", []string{"vet", "./..."}},
		{"Staticcheck", "go", []string{"run", "honnef.co/go/tools/cmd/staticcheck@latest", "./..."}},
		{"Unit tests", "go", []string{"test", "./..."}},
		{"Unit tests with coverage", "go", []string{"test", "-coverprofile=coverage.out", "./..."}},
		{"Build check", "go", []string{"build", "./cmd/server/"}},
	}
	
	allPassed := true
	
	for _, test := range tests {
		fmt.Printf("Running %s... ", test.name)
		
		cmd := exec.Command(test.cmd, test.args...)
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			fmt.Printf("FAILED\n")
			fmt.Printf("Error: %s\n", err)
			fmt.Printf("Output: %s\n", string(output))
			allPassed = false
		} else {
			fmt.Printf("PASSED\n")
		}
	}
	
	if !allPassed {
		log.Fatal("Some tests failed!")
	}
	
	fmt.Println("All tests passed!")
}
```

### Step 7: Document Testing Approach
**File to create:** `doc/TESTING_STRATEGY.md`

```markdown
# Testing Strategy

## Overview
This document outlines the comprehensive testing strategy for the Groupie Tracker application after the simplification refactor.

## Test Types

### 1. Unit Tests
- Test individual functions and methods in isolation
- Located in `_test.go` files alongside the code
- Focus on the new helper methods and utilities
- Validate functional filtering and search logic

### 2. Integration Tests
- Test the interaction between different components
- Validate the complete request/response cycle
- Test API client integration (with external API or mocks)

### 3. Performance Tests
- Benchmark the filtering and search functionality
- Ensure performance hasn't regressed after simplification
- Validate that helper methods are efficient

### 4. End-to-End Tests
- Test the complete application flow
- Validate that all pages render correctly with new view models
- Test form processing and navigation

## Quality Gates

### Pre-commit
1. Run `go fmt`
2. Run `go vet`
3. Run `staticcheck` (if available)
4. Run unit tests

### Continuous Integration
1. Run all tests with coverage
2. Perform build verification
3. Run benchmarks to detect performance regressions
4. Validate code quality tools

## Test Coverage Goals
- Data layer: 80%+
- Web layer: 70%+
- Utilities: 90%+
- Overall: 75%+

## Testing Tools
- Go's built-in testing package for unit and integration tests
- `httptest` for HTTP handler testing
- Standard library for mocking and test helpers
- No external testing frameworks to maintain simplicity

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/data -v

# Run benchmarks
go test -bench=. ./internal/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```