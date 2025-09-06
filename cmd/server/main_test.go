package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

func TestServer_Routes(t *testing.T) {
	// Setup test store with data
	store := storage.NewStore()
	testData := storage.StoreData{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	store.LoadData(testData)

	// Create handlers
	h := handlers.NewHandlers(store)

	// Create router
	mux := createRouter(h)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{"home page", "GET", "/", http.StatusOK},
		{"artists page", "GET", "/artists", http.StatusOK},
		{"artist detail", "GET", "/artists/1", http.StatusOK},
		{"locations page", "GET", "/locations", http.StatusOK},
		{"search API", "GET", "/api/search?q=Queen", http.StatusOK},
		{"suggest API", "GET", "/api/suggest?q=Que", http.StatusOK},
		{"health check", "GET", "/healthz", http.StatusOK},
		{"not found", "GET", "/nonexistent", http.StatusNotFound},
		{"method not allowed", "POST", "/", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code for %s: got %v want %v", tt.url, status, tt.expectedStatus)
			}
		})
	}
}

func TestServer_Middleware(t *testing.T) {
	// Test that recovery middleware works
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	handler := recoveryMiddleware(panicHandler)

	req, err := http.NewRequest("GET", "/panic", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should recover and return 500 instead of crashing
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("expected status 500 after panic recovery, got %v", status)
	}
}

func TestLoadDataFromAPI_Success(t *testing.T) {
	// Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/artists":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"id":1,"name":"Test Artist","creationDate":2000,"members":["Member 1"],"firstAlbum":"01-01-2001","image":"test.jpg"}]`))
		case "/api/locations":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"locations":["test-location"]}]}`))
		case "/api/dates":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"dates":["01-01-2020"]}]}`))
		case "/api/relation":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"datesLocations":{"test-location":["01-01-2020"]}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	store := storage.NewStore()
	client := api.NewClient(mockServer.URL, 5*time.Second)

	err := loadDataFromAPI(store, client)
	if err != nil {
		t.Fatalf("loadDataFromAPI failed: %v", err)
	}

	// Verify data was loaded
	artists := store.GetAllArtists()
	if len(artists) != 1 {
		t.Errorf("Expected 1 artist, got %d", len(artists))
	}

	if artists[0].Name != "Test Artist" {
		t.Errorf("Expected artist name 'Test Artist', got %s", artists[0].Name)
	}
}

func TestLoadDataFromAPI_Error(t *testing.T) {
	// Test with non-existent server
	store := storage.NewStore()
	client := api.NewClient("http://localhost:99999", 1*time.Second)

	err := loadDataFromAPI(store, client)
	if err == nil {
		t.Error("Expected error when API is unreachable, got nil")
	}
}

func TestGetPort(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{"default port", "", ":8080"},
		{"custom port", "3000", ":3000"},
		{"port with colon", ":9000", ":9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock environment variable
			if tt.envValue != "" {
				t.Setenv("PORT", tt.envValue)
			}

			port := getPort()
			if port != tt.expected {
				t.Errorf("Expected port %s, got %s", tt.expected, port)
			}
		})
	}
}
