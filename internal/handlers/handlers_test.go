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
