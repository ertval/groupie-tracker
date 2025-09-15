package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

// getTestStore creates a store with test data
func getTestStore() *storage.Store {
	store := storage.NewStore()

	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Test Artist 1", CreationYear: 2000, FirstAlbum: "01-01-2001", Members: []string{"Member 1", "Member 2"}},
			{ID: 2, Name: "Test Artist 2", CreationYear: 2010, FirstAlbum: "01-01-2011", Members: []string{"Member 3"}},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"new_york-usa", "london-uk"}},
			{ID: 2, Locations: []string{"paris-france", "tokyo-japan"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"01-01-2020", "02-01-2020"}},
			{ID: 2, Dates: []string{"03-01-2020", "04-01-2020"}},
		},
		Relations: []models.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"new_york-usa": {"01-01-2020", "02-01-2020"},
					"london-uk":    {"03-01-2020"},
				},
			},
			{
				ID: 2,
				DatesLocations: map[string][]string{
					"paris-france": {"05-01-2020"},
					"tokyo-japan":  {"06-01-2020", "07-01-2020"},
				},
			},
		},
	}

	store.LoadData(testData)
	return store
}

// getTestHandlers creates handlers with test data
func getTestHandlers() *Handlers {
	store := getTestStore()
	service := service.NewService(store)
	apiClient := api.NewClient("https://example.com", 10*time.Second)
	return NewHandlers(store, service, apiClient)
}

func TestNewHandlers(t *testing.T) {
	store := getTestStore()
	service := service.NewService(store)
	apiClient := api.NewClient("https://example.com", 10*time.Second)

	handlers := NewHandlers(store, service, apiClient)

	if handlers == nil {
		t.Fatal("NewHandlers() returned nil")
	}

	if handlers.store != store {
		t.Error("Handlers store not set correctly")
	}

	if handlers.apiClient != apiClient {
		t.Error("Handlers apiClient not set correctly")
	}

	if handlers.service == nil {
		t.Error("Handlers service not initialized")
	}
}

func TestHandlers_HomeHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HomeHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "text/html; charset=utf-8"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}

	// Check body contains expected content
	body := rr.Body.String()
	if !strings.Contains(body, "Welcome to Groupie Tracker") {
		t.Error("Response body should contain 'Welcome to Groupie Tracker'")
	}
}

func TestHandlers_HomeHandler_WrongMethod(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HomeHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandlers_ArtistsHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/artists", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.ArtistsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "text/html; charset=utf-8"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}
}

func TestHandlers_LocationsHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/locations", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.LocationsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "text/html; charset=utf-8"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}
}

func TestHandlers_ArtistDetailHandler(t *testing.T) {
	handlers := getTestHandlers()

	// Test by ID
	req, err := http.NewRequest("GET", "/artists/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.ArtistDetailHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "text/html; charset=utf-8"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}
}

func TestHandlers_ArtistDetailHandler_NotFound(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/artists/999", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.ArtistDetailHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestHandlers_HealthHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HealthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "application/json"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}

	// Parse response JSON
	var response struct {
		Status string         `json:"status"`
		Stats  map[string]int `json:"stats"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response.Status)
	}
}

func TestHandlers_NotFoundHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.NotFoundHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Check content type
	expected := "text/html; charset=utf-8"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}
}

func TestHandlers_InternalErrorHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handlers.InternalErrorHandler(rr, req, "Test error")

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// Check content type
	expected := "text/html; charset=utf-8"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Handler returned wrong content type: got %v want %v", ct, expected)
	}
}
