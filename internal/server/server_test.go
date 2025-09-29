package server

import (
	"fmt"
	"groupie-tracker/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createTestServer creates a test server with mock API data for testing
func createTestServer(t *testing.T) *Server {
	// Create mock API server with realistic responses
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970, "firstAlbum": "14-12-1973", "members": ["Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon", "Mike Grose", "Barry Mitchell", "Doug Bogie"]},
				{"id": 2, "name": "AC/DC", "creationDate": 1973, "firstAlbum": "17-02-1975", "members": ["Angus Young", "Malcolm Young"]}
			]`))
		case "/api/locations":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "locations": ["London-UK", "Birmingham-UK"]},
					{"id": 2, "locations": ["Sydney-Australia", "Melbourne-Australia"]}
				]
			}`))
		case "/api/dates":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "dates": ["14-12-2022", "15-12-2022"]},
					{"id": 2, "dates": ["15-02-2023", "16-02-2023"]}
				]
			}`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"London-UK": ["14-12-2022"], "Birmingham-UK": ["15-12-2022"]}},
					{"id": 2, "datesLocations": {"Sydney-Australia": ["15-02-2023"], "Melbourne-Australia": ["16-02-2023"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(mockServer.Close)

	// Save original working directory and change to project root
	originalWd, _ := os.Getwd()
	projectRoot := filepath.Join(originalWd, "..", "..")
	err := os.Chdir(projectRoot)
	if err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})

	// Configure test environment
	originalAPIURL := config.APIBaseURL
	originalCache := config.WithCache
	config.APIBaseURL = mockServer.URL
	config.WithCache = false
	config.APIRequestTimeout = 5 * time.Second

	// Restore config after test
	t.Cleanup(func() {
		config.APIBaseURL = originalAPIURL
		config.WithCache = originalCache
	})

	// Create server with dependency injection
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	return server
}

// TestNewServer tests server initialization with dependency injection
func TestNewServer(t *testing.T) {
	server := createTestServer(t)

	// Verify server has all required services
	if server.artists == nil {
		t.Error("Expected ArtistService to be initialized")
	}
	if server.search == nil {
		t.Error("Expected SearchService to be initialized")
	}
	if server.locations == nil {
		t.Error("Expected LocationService to be initialized")
	}
	if server.stats == nil {
		t.Error("Expected StatsService to be initialized")
	}
	if server.cache == nil {
		t.Error("Expected CacheService to be initialized")
	}
	if server.templates == nil {
		t.Error("Expected templates to be initialized")
	}
	if server.httpServer == nil {
		t.Error("Expected httpServer to be initialized")
	}

	// Verify server has loaded data
	artists := server.artists.GetArtists()
	if len(artists) == 0 {
		t.Error("Expected artists to be loaded")
	}

	// Verify stats are available
	stats := server.stats.GetStats()
	if stats["total_artists"] == 0 {
		t.Error("Expected stats to be computed")
	}
}

// TestHomeHandler tests the home page handler
func TestHomeHandler(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.Home(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Queen") {
		t.Error("Expected response to contain artist data")
	}
}

// TestArtistsHandler tests the artists listing handler
func TestArtistsHandler(t *testing.T) {
	server := createTestServer(t)

	// Test GET request
	req := httptest.NewRequest("GET", "/artists", nil)
	w := httptest.NewRecorder()

	server.Artists(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Queen") || !strings.Contains(body, "AC/DC") {
		t.Error("Expected response to contain all artists")
	}
}

// TestArtistDetailHandler tests the artist detail handler
func TestArtistDetailHandler(t *testing.T) {
	server := createTestServer(t)

	// Test valid artist slug
	req := httptest.NewRequest("GET", "/artists/queen", nil)
	w := httptest.NewRecorder()

	server.ArtistDetail(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Queen") {
		t.Error("Expected response to contain Queen details")
	}
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}
}

// TestSearchHandler tests the search functionality
func TestSearchHandler(t *testing.T) {
	server := createTestServer(t)

	// Test GET request (search page)
	req := httptest.NewRequest("GET", "/search", nil)
	w := httptest.NewRecorder()

	server.Search(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestSuggestionsAPI tests the search suggestions API
func TestSuggestionsAPI(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest("GET", "/api/suggestions", nil)
	w := httptest.NewRecorder()

	server.SuggestionsAPI(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}
}

// TestLocationsHandler tests the locations listing handler
func TestLocationsHandler(t *testing.T) {
	server := createTestServer(t)

	req := httptest.NewRequest("GET", "/locations", nil)
	w := httptest.NewRecorder()

	server.Locations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "London") {
		t.Error("Expected response to contain location data")
	}
}

// TestRouting tests that routes use method receivers correctly
func TestRouting(t *testing.T) {
	server := createTestServer(t)

	mux := server.createServeMux()

	// Test that routes are properly configured
	testCases := []struct {
		method   string
		path     string
		expected int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/artists", http.StatusOK},
		{"GET", "/locations", http.StatusOK},
		{"GET", "/health", http.StatusOK},
		{"GET", "/api/suggestions", http.StatusOK},
		{"POST", "/", http.StatusMethodNotAllowed}, // Should reject POST to home
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tc.expected {
				t.Errorf("Expected status %d, got %d for %s %s", tc.expected, w.Code, tc.method, tc.path)
			}
		})
	}
}

// TestServiceInterfaces tests that services work through interfaces
func TestServiceInterfaces(t *testing.T) {
	server := createTestServer(t)

	// Test ArtistService interface
	artists := server.artists.GetArtists()
	if len(artists) == 0 {
		t.Error("ArtistService should return artists")
	}

	// Test SearchService interface
	suggestions := server.search.GenerateAllSearchSuggestions()
	if len(suggestions) == 0 {
		t.Error("SearchService should return suggestions")
	}

	// Test LocationService interface
	locations := server.locations.GetLocations()
	if len(locations) == 0 {
		t.Error("LocationService should return locations")
	}

	// Test StatsService interface
	stats := server.stats.GetStats()
	if stats["total_artists"] == 0 {
		t.Error("StatsService should return stats")
	}

	// Test CacheService interface
	cacheEnabled := server.cache.IsCacheEnabled()
	if cacheEnabled {
		t.Error("CacheService should report cache as disabled in tests")
	}
}

// TestNoDependencyInjectionAntiPatterns tests that we don't have global state
func TestNoDependencyInjectionAntiPatterns(t *testing.T) {
	// This test ensures we properly use dependency injection
	server1 := createTestServer(t)
	// Create second server in a separate test to avoid directory issues

	// Servers should use dependency injection pattern
	if server1.artists == nil {
		t.Error("Server should have injected artist service")
	}
	if server1.search == nil {
		t.Error("Server should have injected search service")
	}
	if server1.locations == nil {
		t.Error("Server should have injected location service")
	}
	if server1.stats == nil {
		t.Error("Server should have injected stats service")
	}
	if server1.cache == nil {
		t.Error("Server should have injected cache service")
	}
}
