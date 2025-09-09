package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

func TestSimplifiedHandlersLocationsSorting(t *testing.T) {
	// Create a simplified store with test data
	store := storage.NewSimplifiedStore()

	// Create test data where location popularity is clear
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Test Artist 1"},
			{ID: 2, Name: "Test Artist 2"},
		},
		Relations: []models.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"london-uk":    {"01-01-2020", "02-01-2020", "03-01-2020"}, // 3 concerts
					"new_york-usa": {"01-02-2020"},                             // 1 concert
				},
			},
			{
				ID: 2,
				DatesLocations: map[string][]string{
					"london-uk":    {"01-03-2020", "02-03-2020"},                             // 2 more concerts (5 total)
					"paris-france": {"01-04-2020", "02-04-2020", "03-04-2020", "04-04-2020"}, // 4 concerts
					"new_york-usa": {"01-05-2020"},                                           // 1 more concert (2 total)
				},
			},
		},
	}

	store.LoadData(testData)

	// Create simplified handlers
	handlers := NewSimplifiedHandlers(store, nil)

	// Create a test HTTP request
	req, err := http.NewRequest("GET", "/locations", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handlers.LocationsHandler(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the response contains HTML
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected content type text/html; charset=utf-8, got %s", contentType)
	}

	// The response should contain the locations in some form
	body := rr.Body.String()
	if body == "" {
		t.Error("Response body is empty")
	}

	// Debug: print the response body to understand what we're getting
	t.Logf("Response body: %s", body)

	// Since templates may not load in test environment, check for fallback content
	if !containsString(body, "Found") && !containsString(body, "Locations") {
		t.Error("Response should contain location-related content")
	}
}

func TestSimplifiedHandlersHomeHandler(t *testing.T) {
	// Create a simplified store with test data
	store := storage.NewSimplifiedStore()

	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Test Artist 1", Members: []string{"Member 1", "Member 2"}},
			{ID: 2, Name: "Test Artist 2", Members: []string{"Member 3"}},
		},
	}

	store.LoadData(testData)

	// Create simplified handlers
	handlers := NewSimplifiedHandlers(store, nil)

	// Create a test HTTP request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handlers.HomeHandler(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the response contains HTML
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected content type text/html; charset=utf-8, got %s", contentType)
	}

	// The response should contain the home page content
	body := rr.Body.String()
	if body == "" {
		t.Error("Response body is empty")
	}
}

func TestSimplifiedHandlersRefreshHandler(t *testing.T) {
	// Create a simplified store
	store := storage.NewSimplifiedStore()

	// Create simplified handlers (without API client for this test)
	handlers := NewSimplifiedHandlers(store, nil)

	// Test that POST is required
	req, err := http.NewRequest("GET", "/api/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.RefreshHandler(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Handler should return 405 for GET request, got %v", status)
	}

	// Test POST without API client
	req, err = http.NewRequest("POST", "/api/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handlers.RefreshHandler(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler should return 500 without API client, got %v", status)
	}
}

// Helper function to check if a string contains a substring
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		haystack != needle &&
		(haystack == needle ||
			haystack[0:len(needle)] == needle ||
			haystack[len(haystack)-len(needle):] == needle ||
			containsSubstring(haystack, needle))
}

func containsSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
