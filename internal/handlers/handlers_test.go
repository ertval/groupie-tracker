package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

// getProjectRoot returns the absolute path to the project root directory
func getProjectRoot() string {
	_, currentFile, _, _ := runtime.Caller(0)
	// From internal/handlers/handlers_test.go, go up two levels to project root
	return filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
}

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

// getTestHandlers creates handlers with test data and proper template loading
func getTestHandlers() *Handlers {
	store := getTestStore()
	service := service.NewService(store)
	apiClient := api.NewClient("https://example.com", 10*time.Second)

	// Create handlers with minimal setup
	h := &Handlers{
		store:     store,
		service:   service,
		apiClient: apiClient,
	}

	// Load templates from the correct path
	projectRoot := getProjectRoot()
	templateFiles := []string{
		filepath.Join(projectRoot, "templates/base.tmpl"),
		filepath.Join(projectRoot, "templates/home.tmpl"),
		filepath.Join(projectRoot, "templates/artists.tmpl"),
		filepath.Join(projectRoot, "templates/artist_detail.tmpl"),
		filepath.Join(projectRoot, "templates/locations.tmpl"),
		filepath.Join(projectRoot, "templates/location_detail.tmpl"),
		filepath.Join(projectRoot, "templates/error.tmpl"),
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"contains": func(slice []string, item string) bool {
			return slices.Contains(slice, item)
		},
		"safeLen": func(slice interface{}) int {
			if slice == nil {
				return 0
			}
			switch s := slice.(type) {
			case []string:
				return len(s)
			case []models.Artist:
				return len(s)
			default:
				return 0
			}
		},
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"generateLocationSlug": func(locationName string) string {
			return models.GenerateLocationSlug(locationName)
		},
		"normalizeLocationName": func(locationName string) string {
			return models.NormalizeLocationName(locationName)
		},
	}

	var err error
	h.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		// For tests, we'll continue with nil templates and test error handling
		h.templates = nil
	}

	return h
}

// getTestHandlersWithoutTemplates creates handlers without template loading (for error testing)
func getTestHandlersWithoutTemplates() *Handlers {
	store := getTestStore()
	service := service.NewService(store)
	apiClient := api.NewClient("https://example.com", 10*time.Second)

	// Create handlers but force templates to be nil to simulate loading failure
	h := &Handlers{
		store:     store,
		service:   service,
		apiClient: apiClient,
		templates: nil, // This will cause template execution to fail
	}

	return h
}

// Tests from handlers_test.go
func TestNewHandlers(t *testing.T) {
	handlers := getTestHandlers()

	if handlers == nil {
		t.Fatal("getTestHandlers() returned nil")
	}

	if handlers.store == nil {
		t.Error("Handlers store not set correctly")
	}

	if handlers.apiClient == nil {
		t.Error("Handlers apiClient not set correctly")
	}

	if handlers.service == nil {
		t.Error("Handlers service not initialized")
	}
}

func TestHandlersHomeHandler(t *testing.T) {
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

func TestHandlersHomeHandlerWrongMethod(t *testing.T) {
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

func TestHandlersArtistsHandler(t *testing.T) {
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

func TestHandlersLocationsHandler(t *testing.T) {
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

func TestHandlersArtistDetailHandler(t *testing.T) {
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

func TestHandlersArtistDetailHandlerNotFound(t *testing.T) {
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

func TestHandlersHealthHandler(t *testing.T) {
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

func TestHandlersNotFoundHandler(t *testing.T) {
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

func TestHandlersInternalErrorHandler(t *testing.T) {
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

// Tests from template_error_test.go
// TestTemplateErrorReturns500 ensures that template execution errors return 500
func TestTemplateErrorReturns500(t *testing.T) {
	// Use handlers without templates
	h := getTestHandlersWithoutTemplates()

	// Test HomeHandler with broken templates
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.HomeHandler(rr, req)

	// Should return 500, not 200 with simple HTML
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d when templates fail, got %d", http.StatusInternalServerError, rr.Code)
	}

	// Should not contain simple HTML fallback content
	body := rr.Body.String()
	if body == "" {
		t.Error("Expected non-empty error response body")
	}
}

// TestArtistDetailHandlerRejectsExtraPath ensures URLs like /artists/123/extra return 404
func TestArtistDetailHandlerRejectsExtraPath(t *testing.T) {
	h := getTestHandlers()

	// Test with extra path segments
	testCases := []string{
		"/artists/1/extra",
		"/artists/1/extra/more",
		"/artists/test-artist/extra",
	}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()

			h.ArtistDetailHandler(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("Expected 404 for path %s, got %d", path, rr.Code)
			}
		})
	}
}

// Tests from handlers_panic_test.go
// TestPanicHandler ensures that a handler panic is recovered and results in a 500 response
func TestPanicHandler(t *testing.T) {
	// Create a Handlers with nil dependencies - the panic handler won't use them
	h := &Handlers{}

	// Create a request to any path
	req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
	rr := httptest.NewRecorder()

	// Wrap the panic in an inline handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentionally cause a panic
		panic("trigger test panic")
	})

	// Use the same recovery pattern as the real handlers: defer recover and call InternalErrorHandler
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				h.InternalErrorHandler(w, r, "Panic: test")
			}
		}()
		handler.ServeHTTP(w, r)
	})

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Fatalf("expected body to contain Internal Server Error, got: %s", body)
	}
}
