package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

func TestRefreshHandler(t *testing.T) {
	// Setup mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/artists":
			w.Header().Set("Content-Type", "application/json")
			artists := []models.Artist{
				{ID: 1, Name: "Refreshed Artist", CreationYear: 2023, Members: []string{"New Member"}, FirstAlbum: "01-01-2023", Image: "test.jpg"},
			}
			json.NewEncoder(w).Encode(artists)
		case "/api/locations":
			w.Header().Set("Content-Type", "application/json")
			locations := map[string][]models.Location{
				"index": {{ID: 1, Locations: []string{"refreshed-location"}}},
			}
			json.NewEncoder(w).Encode(locations)
		case "/api/dates":
			w.Header().Set("Content-Type", "application/json")
			dates := map[string][]models.Date{
				"index": {{ID: 1, Dates: []string{"01-01-2023"}}},
			}
			json.NewEncoder(w).Encode(dates)
		case "/api/relation":
			w.Header().Set("Content-Type", "application/json")
			relations := map[string][]models.Relation{
				"index": {{ID: 1, DatesLocations: map[string][]string{"refreshed-location": {"01-01-2023"}}}},
			}
			json.NewEncoder(w).Encode(relations)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Setup store and handlers
	store := storage.NewStore()
	h := NewHandlers(store)

	// Set API client
	client := api.NewClient(mockServer.URL, 5*time.Second)
	h.SetAPIClient(client)

	t.Run("Successful Refresh", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/refresh", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(h.RefreshHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check response
		var response struct {
			Status  string         `json:"status"`
			Message string         `json:"message"`
			Stats   map[string]int `json:"stats"`
		}
		err = json.NewDecoder(rr.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Could not decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got %s", response.Status)
		}

		// Verify data was refreshed
		artists := store.GetAllArtists()
		if len(artists) != 1 {
			t.Errorf("Expected 1 artist after refresh, got %d", len(artists))
		}

		if artists[0].Name != "Refreshed Artist" {
			t.Errorf("Expected refreshed artist name, got %s", artists[0].Name)
		}
	})

	t.Run("Method Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/refresh", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(h.RefreshHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
		}
	})

	t.Run("No API Client", func(t *testing.T) {
		// Create handler without API client
		storeNoAPI := storage.NewStore()
		hNoAPI := NewHandlers(storeNoAPI)

		req, err := http.NewRequest("POST", "/api/refresh", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(hNoAPI.RefreshHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}
