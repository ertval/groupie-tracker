package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/storage"
)

// TestIntegration tests the complete application integration
func TestIntegration(t *testing.T) {
	// Setup test store with real data
	store := setupIntegrationStore(t)
	h := handlers.NewHandlers(store)

	// Create test server
	router := createTestRouter(h)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("Complete User Journey", func(t *testing.T) {
		// Test home page
		resp, err := http.Get(server.URL + "/")
		if err != nil {
			t.Fatalf("Failed to get home page: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for home page, got %d", resp.StatusCode)
		}

		// Test artists page
		resp, err = http.Get(server.URL + "/artists")
		if err != nil {
			t.Fatalf("Failed to get artists page: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for artists page, got %d", resp.StatusCode)
		}

		// Test specific artist detail
		resp, err = http.Get(server.URL + "/artists/1")
		if err != nil {
			t.Fatalf("Failed to get artist detail: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for artist detail, got %d", resp.StatusCode)
		}

		// Test locations page
		resp, err = http.Get(server.URL + "/locations")
		if err != nil {
			t.Fatalf("Failed to get locations page: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for locations page, got %d", resp.StatusCode)
		}
	})

	t.Run("API Endpoints", func(t *testing.T) {
		// Test search API
		resp, err := http.Get(server.URL + "/api/search?q=Queen")
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for search, got %d", resp.StatusCode)
		}

		var searchResponse handlers.SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
			t.Fatalf("Failed to decode search response: %v", err)
		}

		if len(searchResponse.Artists) == 0 {
			t.Error("Expected search results for 'Queen', got none")
		}

		// Test suggestions API
		resp, err = http.Get(server.URL + "/api/suggest?q=Que")
		if err != nil {
			t.Fatalf("Failed to get suggestions: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for suggestions, got %d", resp.StatusCode)
		}

		var suggestResponse handlers.SuggestResponse
		if err := json.NewDecoder(resp.Body).Decode(&suggestResponse); err != nil {
			t.Fatalf("Failed to decode suggest response: %v", err)
		}

		if len(suggestResponse.Suggestions) == 0 {
			t.Error("Expected suggestions for 'Que', got none")
		}

		// Test health check
		resp, err = http.Get(server.URL + "/healthz")
		if err != nil {
			t.Fatalf("Failed to get health check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for health check, got %d", resp.StatusCode)
		}

		var healthResponse handlers.HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		if healthResponse.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got %s", healthResponse.Status)
		}
	})

	t.Run("Error Handling", func(t *testing.T) {
		// Test 404 for non-existent artist
		resp, err := http.Get(server.URL + "/artists/99999")
		if err != nil {
			t.Fatalf("Failed to get non-existent artist: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent artist, got %d", resp.StatusCode)
		}

		// Test 404 for invalid path
		resp, err = http.Get(server.URL + "/invalid-path")
		if err != nil {
			t.Fatalf("Failed to get invalid path: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for invalid path, got %d", resp.StatusCode)
		}

		// Test invalid artist ID format
		resp, err = http.Get(server.URL + "/artists/invalid")
		if err != nil {
			t.Fatalf("Failed to get artist with invalid ID: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid artist ID, got %d", resp.StatusCode)
		}
	})

	t.Run("Event/Action Functionality", func(t *testing.T) {
		// Test live search (event/action requirement)
		resp, err := http.Get(server.URL + "/api/search?q=foo")
		if err != nil {
			t.Fatalf("Failed to perform live search: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for live search, got %d", resp.StatusCode)
		}

		// Verify this is a client-server communication
		if resp.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected JSON response for API endpoint")
		}

		// Test autocomplete suggestions (another event/action)
		resp, err = http.Get(server.URL + "/api/suggest?q=gori")
		if err != nil {
			t.Fatalf("Failed to get autocomplete: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for autocomplete, got %d", resp.StatusCode)
		}
	})
}

// TestServerStability ensures the server doesn't crash under load
func TestServerStability(t *testing.T) {
	store := setupIntegrationStore(t)
	h := handlers.NewHandlers(store)
	router := createTestRouter(h)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("Concurrent Requests", func(t *testing.T) {
		const numRequests = 50
		done := make(chan bool, numRequests)

		// Send multiple concurrent requests
		for i := 0; i < numRequests; i++ {
			go func(id int) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Server panicked on request %d: %v", id, r)
					}
					done <- true
				}()

				endpoints := []string{
					"/",
					"/artists",
					"/artists/1",
					"/locations",
					"/api/search?q=test",
					"/api/suggest?q=te",
					"/healthz",
				}

				endpoint := endpoints[id%len(endpoints)]
				resp, err := http.Get(server.URL + endpoint)
				if err != nil {
					t.Errorf("Request %d failed: %v", id, err)
					return
				}
				defer resp.Body.Close()

				// Server should not crash (status should be valid HTTP status)
				if resp.StatusCode < 200 || resp.StatusCode >= 600 {
					t.Errorf("Invalid status code %d for request %d", resp.StatusCode, id)
				}
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			<-done
		}
	})
}

// TestDataConsistency verifies data consistency across the application
func TestDataConsistency(t *testing.T) {
	store := setupIntegrationStore(t)

	t.Run("Artist Data Consistency", func(t *testing.T) {
		artists := store.GetAllArtists()
		if len(artists) == 0 {
			t.Fatal("No artists loaded")
		}

		// Verify each artist has valid data
		for _, artist := range artists {
			if err := artist.Validate(); err != nil {
				t.Errorf("Artist %s failed validation: %v", artist.Name, err)
			}

			// Verify first album date format
			if artist.FirstAlbum != "" {
				_, err := artist.GetFirstAlbumDate()
				if err != nil {
					t.Errorf("Artist %s has invalid first album date %s: %v",
						artist.Name, artist.FirstAlbum, err)
				}
			}
		}
	})

	t.Run("Search Functionality Consistency", func(t *testing.T) {
		// Test case-insensitive search
		results1 := store.SearchArtists("QUEEN")
		results2 := store.SearchArtists("queen")
		results3 := store.SearchArtists("Queen")

		if len(results1) != len(results2) || len(results2) != len(results3) {
			t.Error("Case-insensitive search returned different results")
		}

		// Test member search
		memberResults := store.SearchArtists("Freddie")
		found := false
		for _, artist := range memberResults {
			for _, member := range artist.Members {
				if strings.Contains(strings.ToLower(member), "freddie") {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("Member search should find artists with matching member names")
		}
	})
}

// Helper functions

func setupIntegrationStore(t *testing.T) *storage.Store {
	store := storage.NewStore()
	client := api.NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := client.FetchAllData(ctx)
	if err != nil {
		t.Fatalf("Failed to load data from API: %v", err)
	}

	storeData := storage.StoreData{
		Artists:   data.Artists,
		Locations: data.Locations,
		Dates:     data.Dates,
		Relations: data.Relations,
	}
	store.LoadData(storeData)

	return store
}

func createTestRouter(h *handlers.Handlers) http.Handler {
	mux := http.NewServeMux()

	// Web routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			h.NotFoundHandler(w, r)
			return
		}
		h.HomeHandler(w, r)
	})
	mux.HandleFunc("/artists", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/artists" {
			h.NotFoundHandler(w, r)
			return
		}
		h.ArtistsHandler(w, r)
	})
	mux.HandleFunc("/artists/", h.ArtistDetailHandler)
	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/locations" {
			h.NotFoundHandler(w, r)
			return
		}
		h.LocationsHandler(w, r)
	})

	// API routes
	mux.HandleFunc("/api/search", h.SearchHandler)
	mux.HandleFunc("/api/suggest", h.SuggestHandler)
	mux.HandleFunc("/healthz", h.HealthHandler)

	return mux
}
