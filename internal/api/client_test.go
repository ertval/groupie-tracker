package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/models"
)

func TestClient_FetchArtists(t *testing.T) {
	// Mock API server
	mockArtists := []models.Artist{
		{
			ID:           1,
			Name:         "Queen",
			Image:        "https://example.com/queen.jpg",
			Members:      []string{"Freddie Mercury", "Brian May"},
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
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/artists" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockArtists)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	artists, err := client.FetchArtists(context.Background())
	if err != nil {
		t.Fatalf("FetchArtists() error = %v", err)
	}

	if len(artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(artists))
	}

	if artists[0].Name != "Queen" {
		t.Errorf("Expected first artist to be Queen, got %s", artists[0].Name)
	}

	if artists[1].Name != "Gorillaz" {
		t.Errorf("Expected second artist to be Gorillaz, got %s", artists[1].Name)
	}
}

func TestClient_FetchLocations(t *testing.T) {
	mockLocations := []models.Location{
		{
			ID:        1,
			Locations: []string{"london-uk", "manchester-uk"},
		},
		{
			ID:        2,
			Locations: []string{"new_york-usa", "los_angeles-usa"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/locations" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]models.Location{
			"index": mockLocations,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	locations, err := client.FetchLocations(context.Background())
	if err != nil {
		t.Fatalf("FetchLocations() error = %v", err)
	}

	if len(locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(locations))
	}
}

func TestClient_FetchDates(t *testing.T) {
	mockDates := []models.Date{
		{
			ID:    1,
			Dates: []string{"23-08-2019", "22-08-2019"},
		},
		{
			ID:    2,
			Dates: []string{"25-08-2019", "26-08-2019"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dates" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]models.Date{
			"index": mockDates,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	dates, err := client.FetchDates(context.Background())
	if err != nil {
		t.Fatalf("FetchDates() error = %v", err)
	}

	if len(dates) != 2 {
		t.Errorf("Expected 2 dates, got %d", len(dates))
	}
}

func TestClient_FetchRelations(t *testing.T) {
	mockRelations := []models.Relation{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"london-uk":    {"23-08-2019", "22-08-2019"},
				"new_york-usa": {"25-08-2019"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/relation" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]models.Relation{
			"index": mockRelations,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	relations, err := client.FetchRelations(context.Background())
	if err != nil {
		t.Fatalf("FetchRelations() error = %v", err)
	}

	if len(relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(relations))
	}
}

func TestClient_FetchAllData(t *testing.T) {
	// Mock complete API response
	mockData := struct {
		Artists   []models.Artist   `json:"artists"`
		Locations []models.Location `json:"locations"`
		Dates     []models.Date     `json:"dates"`
		Relations []models.Relation `json:"relations"`
	}{
		Artists: []models.Artist{
			{
				ID:           1,
				Name:         "Queen",
				Members:      []string{"Freddie Mercury"},
				CreationYear: 1970,
				FirstAlbum:   "14-12-1973",
			},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"23-08-2019"}},
		},
		Relations: []models.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"london-uk": {"23-08-2019"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/artists":
			json.NewEncoder(w).Encode(mockData.Artists)
		case "/api/locations":
			json.NewEncoder(w).Encode(map[string][]models.Location{"index": mockData.Locations})
		case "/api/dates":
			json.NewEncoder(w).Encode(map[string][]models.Date{"index": mockData.Dates})
		case "/api/relation":
			json.NewEncoder(w).Encode(map[string][]models.Relation{"index": mockData.Relations})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	data, err := client.FetchAllData(context.Background())
	if err != nil {
		t.Fatalf("FetchAllData() error = %v", err)
	}

	if len(data.Artists) != 1 {
		t.Errorf("Expected 1 artist, got %d", len(data.Artists))
	}

	if len(data.Locations) != 1 {
		t.Errorf("Expected 1 location, got %d", len(data.Locations))
	}

	if len(data.Dates) != 1 {
		t.Errorf("Expected 1 date, got %d", len(data.Dates))
	}

	if len(data.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(data.Relations))
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	// Test timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, 50*time.Millisecond)

	ctx := context.Background()
	_, err := client.FetchArtists(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Test 404 error
	server404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server404.Close()

	client404 := NewClient(server404.URL, 5*time.Second)
	_, err = client404.FetchArtists(ctx)
	if err == nil {
		t.Error("Expected 404 error, got nil")
	}
}

func TestClient_InvalidJSON(t *testing.T) {
	// Test invalid JSON response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	_, err := client.FetchArtists(context.Background())
	if err == nil {
		t.Error("Expected JSON parsing error, got nil")
	}
}
