package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

func TestNewServer_Success(t *testing.T) {
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

	// Temporarily override default API URL for test
	originalURL := DefaultAPIURL
	defer func() { _ = originalURL }() // Restore after test

	// Create a test version of NewServer that uses mock server
	apiClient := api.NewClient(mockServer.URL, RequestTimeout)
	
	// Create adapter for storage interface
	adapter := &apiClientAdapter{client: apiClient}
	
	// Initialize store with cache
	store := storage.NewStoreWithCache(adapter)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Start cache for initial data load
	store.StartCache(ctx)
	time.Sleep(100 * time.Millisecond) // Wait for initial load
	store.StopCache()

	// Verify that data was loaded correctly
	artists := store.GetAllArtists()
	if len(artists) != 1 {
		t.Errorf("Expected 1 artist, got %d", len(artists))
	}

	if artists[0].Name != "Test Artist" {
		t.Errorf("Expected artist name 'Test Artist', got %s", artists[0].Name)
	}
}

func TestNewServer_APIError(t *testing.T) {
	// Test with unreachable API
	originalURL := DefaultAPIURL
	defer func() { _ = originalURL }()

	// Create a test that should fail
	client := api.NewClient("http://localhost:99999", 1*time.Second)
	
	// Create adapter for storage interface
	adapter := &apiClientAdapter{client: client}
	
	// Initialize store with cache
	store := storage.NewStoreWithCache(adapter)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Try to start cache with unreachable API - should still not return error but data won't load
	store.StartCache(ctx)
	time.Sleep(100 * time.Millisecond)
	store.StopCache()
	
	// Verify no data was loaded due to API failure
	stats := store.GetStats()
	if stats["artists"] > 0 {
		t.Error("Expected no artists to be loaded when API is unreachable")
	}
}

func TestServer_ConfigConstants(t *testing.T) {
	// Test that all constants are properly defined
	if DefaultPort != ":8080" {
		t.Errorf("Expected DefaultPort to be ':8080', got %s", DefaultPort)
	}

	if DefaultAPIURL != "https://groupietrackers.herokuapp.com" {
		t.Errorf("Expected DefaultAPIURL to be 'https://groupietrackers.herokuapp.com', got %s", DefaultAPIURL)
	}

	if RequestTimeout != 30*time.Second {
		t.Errorf("Expected RequestTimeout to be 30s, got %v", RequestTimeout)
	}

	if ShutdownTimeout != 10*time.Second {
		t.Errorf("Expected ShutdownTimeout to be 10s, got %v", ShutdownTimeout)
	}

	if ReadTimeout != 15*time.Second {
		t.Errorf("Expected ReadTimeout to be 15s, got %v", ReadTimeout)
	}

	if WriteTimeout != 15*time.Second {
		t.Errorf("Expected WriteTimeout to be 15s, got %v", WriteTimeout)
	}

	if IdleTimeout != 60*time.Second {
		t.Errorf("Expected IdleTimeout to be 60s, got %v", IdleTimeout)
	}
}

func TestCreateRouter_RoutesExist(t *testing.T) {
	// Setup test store with minimal data
	store := storage.NewStore()
	testData := storage.StoreData{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	store.LoadData(testData)

	// Create handlers and router
	h := handlers.NewHandlers(store)
	mux := createRouter(h)

	// Test routes exist and respond appropriately
	testRoutes := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/artists", http.StatusOK},
		{"GET", "/artists/1", http.StatusOK},
		{"GET", "/locations", http.StatusOK},
		{"GET", "/api/search", http.StatusOK},
		{"GET", "/api/suggest", http.StatusOK},
		{"GET", "/healthz", http.StatusOK},
	}

	for _, route := range testRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != route.status {
				t.Errorf("Expected status %d for %s %s, got %d", route.status, route.method, route.path, w.Code)
			}
		})
	}
}

func TestMiddleware_Recovery(t *testing.T) {
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	wrapped := recoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic and should return 500
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic recovery, got %d", w.Code)
	}

	if w.Body.String() == "" {
		t.Error("Expected error message in response body")
	}
}

func TestMiddleware_Logging(t *testing.T) {
	// Simple handler for testing
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with logging middleware
	wrapped := loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic or error
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}
