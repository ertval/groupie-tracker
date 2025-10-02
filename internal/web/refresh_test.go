package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/conf"
)

// createMockAPIServer creates a mock API server for testing
func createMockAPIServer(t *testing.T) (*httptest.Server, *api.Client) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970, "firstAlbum": "14-12-1973", "members": ["Freddie Mercury"], "image": "https://example.com/queen.jpg"}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"London-UK": ["14-12-2022"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))

	// Configure test environment
	originalAPIURL := conf.APIBaseURL
	conf.APIBaseURL = mockServer.URL
	t.Cleanup(func() {
		conf.APIBaseURL = originalAPIURL
		mockServer.Close()
	})

	return mockServer, api.NewClient(mockServer.URL, 5*time.Second)
}

// TestManualRefresh tests the manual refresh endpoint
func TestManualRefresh(t *testing.T) {
	// Create test app
	_, mockClient := createMockAPIServer(t)
	app, err := NewApp(mockClient)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Test POST request (should succeed)
	req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("POST /api/refresh: expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	// Check response body contains expected message
	body := w.Body.String()
	if body == "" {
		t.Error("POST /api/refresh: expected non-empty response body")
	}

	// Test GET request (should fail with 405)
	req = httptest.NewRequest(http.MethodGet, "/api/refresh", nil)
	w = httptest.NewRecorder()
	app.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/refresh: expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestGetStore tests the thread-safe getStore method
func TestGetStore(t *testing.T) {
	_, mockClient := createMockAPIServer(t)
	app, err := NewApp(mockClient)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Test that getStore returns a non-nil store
	store := app.getStore()
	if store == nil {
		t.Error("getStore() returned nil")
	}

	// Test that we can call it multiple times
	store2 := app.getStore()
	if store2 == nil {
		t.Error("getStore() second call returned nil")
	}

	// Should be the same store
	if store != store2 {
		t.Error("getStore() returned different store instances")
	}
}

// TestShutdown tests the graceful shutdown functionality
func TestShutdown(t *testing.T) {
	_, mockClient := createMockAPIServer(t)
	app, err := NewApp(mockClient)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = app.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	// Verify ticker is stopped (channel should be closed)
	// Note: We can't directly test if ticker is stopped, but we can verify shutdown completes
}

// TestConcurrentStoreAccess tests concurrent access to the store
func TestConcurrentStoreAccess(t *testing.T) {
	_, mockClient := createMockAPIServer(t)
	app, err := NewApp(mockClient)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Create multiple goroutines that access the store concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				store := app.getStore()
				_ = store.Artists()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
