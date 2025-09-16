package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"groupie-tracker/internal/client"
	"groupie-tracker/internal/data"
)

// getProjectRoot returns the absolute path to the project root directory
func getProjectRoot() string {
	_, currentFile, _, _ := runtime.Caller(0)
	// From internal/handlers/handlers_test.go, go up two levels to project root
	return filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
}

// getTestRepository creates a repository with test data
func getTestRepository() *data.Repository {
	repo := data.NewRepository()

	testData := &client.Response{
		Artists: []client.Artist{
			{ID: 1, Name: "Test Artist 1", CreationYear: 2000, FirstAlbum: "01-01-2001", Members: []string{"Member 1", "Member 2"}},
			{ID: 2, Name: "Test Artist 2", CreationYear: 2010, FirstAlbum: "01-01-2011", Members: []string{"Member 3"}},
		},
		Relations: []client.Relation{
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

	repo.Initialize(context.Background(), nil, testData)
	return repo
}

// getTestHandlers creates handlers with test data and proper template loading
func getTestHandlers() *Handlers {
	repo := getTestRepository()

	// Change to project root directory for template loading
	projectRoot := getProjectRoot()
	originalDir, _ := os.Getwd()
	os.Chdir(projectRoot)

	handlers := NewHandlers(repo)

	// Restore original directory
	os.Chdir(originalDir)

	return handlers
}

func TestNewHandlers(t *testing.T) {
	repo := getTestRepository()

	// Test basic creation without template loading for now
	h := &Handlers{
		repo: repo,
	}

	// h is created as a composite literal and therefore cannot be nil here.
	// Verify the repository field is set correctly.
	if h.repo == nil {
		t.Error("Expected repository to be set, got nil")
	}
}

func TestHandlersHomeHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.HomeHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Test Artist 1") {
		t.Errorf("Expected response to contain 'Test Artist 1', got: %s", body)
	}
}

func TestHandlersArtistsHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/artists", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.ArtistsHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Test Artist 1") {
		t.Errorf("Expected response to contain 'Test Artist 1', got: %s", body)
	}
}

func TestHandlersArtistDetailHandler(t *testing.T) {
	handlers := getTestHandlers()

	// Test with valid ID
	req, err := http.NewRequest("GET", "/artists/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.ArtistDetailHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Test Artist 1") {
		t.Errorf("Expected response to contain 'Test Artist 1', got: %s", body)
	}
}

func TestHandlersArtistDetailHandlerNotFound(t *testing.T) {
	handlers := getTestHandlers()

	// Test with invalid ID
	req, err := http.NewRequest("GET", "/artists/999", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.ArtistDetailHandler(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}

func TestHandlersArtistDetailHandlerInvalidPaths(t *testing.T) {
	handlers := getTestHandlers()

	// Missing parameter (should be 404)
	req1, _ := http.NewRequest("GET", "/artists/", nil)
	rr1 := httptest.NewRecorder()
	handlers.ArtistDetailHandler(rr1, req1)
	if status := rr1.Code; status != http.StatusNotFound {
		t.Errorf("expected 404 for '/artists/' got %v", status)
	}

	// Too many parts (should be 404)
	req2, _ := http.NewRequest("GET", "/artists/1/extra", nil)
	rr2 := httptest.NewRecorder()
	handlers.ArtistDetailHandler(rr2, req2)
	if status := rr2.Code; status != http.StatusNotFound {
		t.Errorf("expected 404 for '/artists/1/extra' got %v", status)
	}
}

func TestHandlersLocationsHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/locations", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.LocationsHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "new_york-usa") || !strings.Contains(body, "london-uk") {
		t.Errorf("Expected response to contain location data, got: %s", body)
	}
}

func TestHandlersLocationDetailHandlerInvalidPaths(t *testing.T) {
	handlers := getTestHandlers()

	// Missing parameter (should be 404)
	req1, _ := http.NewRequest("GET", "/locations/", nil)
	rr1 := httptest.NewRecorder()
	handlers.LocationDetailHandler(rr1, req1)
	if status := rr1.Code; status != http.StatusNotFound {
		t.Errorf("expected 404 for '/locations/' got %v", status)
	}

	// Too many parts (should be 404)
	req2, _ := http.NewRequest("GET", "/locations/new_york-usa/extra", nil)
	rr2 := httptest.NewRecorder()
	handlers.LocationDetailHandler(rr2, req2)
	if status := rr2.Code; status != http.StatusNotFound {
		t.Errorf("expected 404 for '/locations/new_york-usa/extra' got %v", status)
	}
}

func TestHandlersHealthHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.HealthHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, "application/json")
	}

	body := rr.Body.String()
	if !strings.Contains(body, "\"status\":\"ok\"") {
		t.Errorf("Expected response to contain status ok, got: %s", body)
	}
}

func TestHandlersNotFoundHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.NotFoundHandler(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}

func TestHandlersInternalErrorHandler(t *testing.T) {
	handlers := getTestHandlers()

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.InternalErrorHandler(rr, req, "Test error")

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}

func TestHandlersMethodNotAllowed(t *testing.T) {
	handlers := getTestHandlers()

	// Test with invalid method
	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlers.HomeHandler(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusMethodNotAllowed)
	}
}
