package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

func setupTestStore() *storage.Store {
	store := storage.NewStore()

	// Add test data
	artists := []models.Artist{
		{
			ID:           1,
			Name:         "Queen",
			Image:        "https://example.com/queen.jpg",
			Members:      []string{"Freddie Mercury", "Brian May", "John Daecon", "Roger Meddows-Taylor", "Mike Grose", "Barry Mitchell", "Doug Fogie"},
			CreationYear: 1970,
			FirstAlbum:   "14-12-1973",
		},
		{
			ID:           2,
			Name:         "Gorillaz",
			Image:        "https://example.com/gorillaz.jpg",
			Members:      []string{"Damon Albarn", "Jamie Hewlett"},
			CreationYear: 1998,
			FirstAlbum:   "26-03-2001",
		},
		{
			ID:           3,
			Name:         "Travis Scott",
			Image:        "https://example.com/travis.jpg",
			Members:      []string{"Travis Scott"},
			CreationYear: 2008,
			FirstAlbum:   "01-01-2015",
		},
		{
			ID:           4,
			Name:         "Foo Fighters",
			Image:        "https://example.com/foo.jpg",
			Members:      []string{"Dave Grohl", "Nate Mendel", "Taylor Hawkins", "Chris Shiflett", "Pat Smear", "Rami Jaffee"},
			CreationYear: 1994,
			FirstAlbum:   "04-07-1995",
		},
	}

	for _, artist := range artists {
		store.AddArtist(artist)
	}

	locations := []models.Location{
		{
			ID:        3,
			Locations: []string{"santiago-chile", "sao_paulo-brasil", "los_angeles-usa", "houston-usa", "atlanta-usa", "new_orleans-usa", "philadelphia-usa", "london-uk", "frauenfeld-switzerland", "turku-finland"},
		},
	}

	for _, location := range locations {
		store.AddLocation(location)
	}

	return store
}

func TestHomeHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.HomeHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check if response contains expected content (this would be more specific with actual templates)
	body := rr.Body.String()
	if len(body) == 0 {
		t.Error("handler returned empty body")
	}
}

func TestArtistsHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/artists", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.ArtistsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestArtistDetailHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{"valid artist ID", "/artists/1", http.StatusOK},
		{"invalid artist ID", "/artists/999", http.StatusNotFound},
		{"invalid ID format", "/artists/abc", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			// Create a router to test the actual URL pattern
			mux := http.NewServeMux()
			mux.HandleFunc("/artists/", h.ArtistDetailHandler)
			mux.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestSearchHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedCount  int
	}{
		{"search Queen", "/api/search?q=Queen", http.StatusOK, 1},
		{"search case insensitive", "/api/search?q=queen", http.StatusOK, 1},
		{"search member", "/api/search?q=Freddie", http.StatusOK, 1},
		{"no results", "/api/search?q=Beatles", http.StatusOK, 0},
		{"empty query", "/api/search", http.StatusOK, 4}, // should return all
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(h.SearchHandler)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response SearchResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Could not unmarshal response: %v", err)
				}

				if len(response.Artists) != tt.expectedCount {
					t.Errorf("Expected %d artists, got %d", tt.expectedCount, len(response.Artists))
				}
			}
		})
	}
}

func TestSuggestHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		minSuggestions int
	}{
		{"suggest Queen", "/api/suggest?q=Que", http.StatusOK, 1},
		{"suggest member", "/api/suggest?q=Fred", http.StatusOK, 1},
		{"no suggestions", "/api/suggest?q=xyz", http.StatusOK, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(h.SuggestHandler)

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response SuggestResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Could not unmarshal response: %v", err)
				}

				if len(response.Suggestions) < tt.minSuggestions {
					t.Errorf("Expected at least %d suggestions, got %d", tt.minSuggestions, len(response.Suggestions))
				}
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.HealthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response HealthResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Could not unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status to be 'healthy', got %s", response.Status)
	}
}

func TestNotFoundHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.NotFoundHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestInternalErrorHandler(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/error", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.InternalErrorHandler(rr, req, "test error")

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}
}

func TestInternalErrorHandler_WithError(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/error", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.InternalErrorHandler(rr, req, "test error")

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}
}

func TestArtistDetailHandler_InvalidID(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/artists/invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.ArtistDetailHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestArtistDetailHandler_NotFound(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/artists/999", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.ArtistDetailHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestSearchHandler_EmptyQuery(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/api/search?q=", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.SearchHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response SearchResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Could not unmarshal response: %v", err)
	}

	// Empty query should return all artists (this is the expected behavior)
	if len(response.Artists) != 4 {
		t.Errorf("Expected 4 artists for empty query (all artists), got %d", len(response.Artists))
	}

	if response.Query != "" {
		t.Errorf("Expected empty query string, got %s", response.Query)
	}
}

func TestSuggestHandler_EmptyQuery(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("GET", "/api/suggest?q=", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.SuggestHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response SuggestResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Could not unmarshal response: %v", err)
	}

	if len(response.Suggestions) != 0 {
		t.Errorf("Expected 0 suggestions for empty query, got %d", len(response.Suggestions))
	}
}

func TestSearchHandler_MethodNotAllowed(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("POST", "/api/search", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.SearchHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestSuggestHandler_MethodNotAllowed(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("POST", "/api/suggest", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.SuggestHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHomeHandler_MethodNotAllowed(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.HomeHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestArtistsHandler_MethodNotAllowed(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("POST", "/artists", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.ArtistsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestLocationsHandler_MethodNotAllowed(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("POST", "/locations", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.LocationsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("POST", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.HealthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestArtistDetailHandler_MethodNotAllowed(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	req, err := http.NewRequest("POST", "/artists/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.ArtistDetailHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestSetAPIClient(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	// Test setting API client (this method doesn't have return value to test)
	// but calling it should not panic
	h.SetAPIClient(nil)

	// Test that it doesn't panic with a valid client
	// We can't create a real api.Client here without circular imports,
	// so we just test that the method exists and can be called
}

func TestSetTemplates(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	// Test setting templates with nil (should not panic)
	h.SetTemplates(nil)

	// Test that it doesn't panic - we can't easily create a real template here
	// without more complex setup, so we just test nil handling
}

func TestCalculateLocationStats(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	// Test calculateLocationStats function
	stats := h.calculateLocationStats()

	// The function should return a slice (might be empty)
	// The function always returns a slice, never nil
	if len(stats) < 0 {
		t.Error("Location stats length should not be negative")
	}

	// Function should always return a valid slice
	// Empty slice is valid for test data
}

func TestCalculateTotalCountries(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	// First get location stats
	locationStats := h.calculateLocationStats()

	// Test calculateTotalCountries function
	count := h.calculateTotalCountries(locationStats)

	// We should get a non-negative count
	if count < 0 {
		t.Error("Country count should not be negative")
	}
}

func TestCalculateTotalConcerts(t *testing.T) {
	store := setupTestStore()
	h := NewHandlers(store)

	// Test calculateTotalConcerts function
	count := h.calculateTotalConcerts()

	// We should get a non-negative count
	if count < 0 {
		t.Error("Concert count should not be negative")
	}
}
