package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_FetchAll(t *testing.T) {
	// Mock API server that serves all endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/artists":
			mockArtists := []Artist{
				{
					ID:           1,
					Name:         "Queen",
					Image:        "https://example.com/queen.jpg",
					Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
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
			json.NewEncoder(w).Encode(mockArtists)

		case "/api/relation":
			mockRelations := map[string][]Relation{
				"index": {
					{
						ID: 1,
						DatesLocations: map[string][]string{
							"london-uk":     {"14-12-1975", "15-12-1975"},
							"manchester-uk": {"20-12-1975"},
						},
					},
					{
						ID: 2,
						DatesLocations: map[string][]string{
							"new_york-usa":    {"26-03-2001", "27-03-2001"},
							"los_angeles-usa": {"30-03-2001"},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(mockRelations)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	response, err := client.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll() error = %v", err)
	}

	// Test artists
	if len(response.Artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(response.Artists))
	}

	if response.Artists[0].Name != "Queen" {
		t.Errorf("Expected first artist to be Queen, got %s", response.Artists[0].Name)
	}

	if len(response.Artists[0].Members) != 4 {
		t.Errorf("Expected Queen to have 4 members, got %d", len(response.Artists[0].Members))
	}

	if response.Artists[1].Name != "Gorillaz" {
		t.Errorf("Expected second artist to be Gorillaz, got %s", response.Artists[1].Name)
	}

	// Test relations
	if len(response.Relations) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(response.Relations))
	}

	queenRelation := response.Relations[0]
	if queenRelation.ID != 1 {
		t.Errorf("Expected first relation to have ID 1, got %d", queenRelation.ID)
	}

	if len(queenRelation.DatesLocations) != 2 {
		t.Errorf("Expected Queen to have 2 locations, got %d", len(queenRelation.DatesLocations))
	}

	if len(queenRelation.DatesLocations["london-uk"]) != 2 {
		t.Errorf("Expected Queen to have 2 dates in London, got %d", len(queenRelation.DatesLocations["london-uk"]))
	}
}

func TestClient_FetchAll_Timeout(t *testing.T) {
	// Mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Artist{})
	}))
	defer server.Close()

	// Create client with very short timeout
	client := NewClient(server.URL, 50*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.FetchAll(ctx)
	if err == nil {
		t.Error("Expected timeout error, but got none")
	}
}

func TestClient_FetchAll_InvalidURL(t *testing.T) {
	client := NewClient("invalid-url", 5*time.Second)

	_, err := client.FetchAll(context.Background())
	if err == nil {
		t.Error("Expected error for invalid URL, but got none")
	}
}

func TestClient_FetchAll_EmptyResponse(t *testing.T) {
	// Mock server that returns empty data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/artists":
			json.NewEncoder(w).Encode([]Artist{})
		case "/api/relation":
			json.NewEncoder(w).Encode(map[string][]Relation{"index": {}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	response, err := client.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll() error = %v", err)
	}

	if len(response.Artists) != 0 {
		t.Errorf("Expected 0 artists, got %d", len(response.Artists))
	}

	if len(response.Relations) != 0 {
		t.Errorf("Expected 0 relations, got %d", len(response.Relations))
	}
}

func TestNewClient(t *testing.T) {
	baseURL := "https://example.com"
	timeout := 10 * time.Second

	client := NewClient(baseURL, timeout)

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL to be %s, got %s", baseURL, client.baseURL)
	}

	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected HTTP client timeout to be %v, got %v", timeout, client.httpClient.Timeout)
	}
}
