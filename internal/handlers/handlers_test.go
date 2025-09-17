// Package handlers provides tests for HTTP handlers.
package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/repository"
)

func createTestRepository() *repository.Repository {
	// Create repository without loading data for basic handler testing
	repo := repository.NewRepository("http://test-api", time.Second*5)
	return repo
}

func TestNewHandler(t *testing.T) {
	repo := createTestRepository()

	// Test that we can create a handler without loading templates
	handler := &Handler{repo: repo}

	// Handler constructed directly; verify repository field is set correctly.
	if handler.repo != repo {
		t.Error("Handler should store the provided repository")
	}

	// Note: We don't test NewHandler directly since it requires template files
}

func TestHealthHandler(t *testing.T) {
	repo := createTestRepository()
	handler := &Handler{repo: repo}

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Request-Time", "2023-01-01T00:00:00Z")
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content-type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestHealthHandlerInvalidMethod(t *testing.T) {
	repo := createTestRepository()
	handler := &Handler{repo: repo}

	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDevPanicHandler(t *testing.T) {
	repo := createTestRepository()
	handler := &Handler{repo: repo}

	req := httptest.NewRequest("GET", "/dev/panic", nil)
	w := httptest.NewRecorder()

	// Test that panic actually occurs
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to occur")
		}
	}()

	handler.DevPanic(w, req)
}

func TestCreateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Queen", "queen"},
		{"Foo Fighters", "foo-fighters"},
		{"AC/DC", "ac-dc"},
		{"Green Day", "green-day"},
		{"Twenty One Pilots", "twenty-one-pilots"},
		{"", ""},
		{"   ", ""},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!@#$%Characters", "special-characters"},
	}

	for _, test := range tests {
		result := createSlug(test.input)
		if result != test.expected {
			t.Errorf("createSlug(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// Test the render method error handling (indirectly)
func TestRenderWithNilTemplates(t *testing.T) {
	repo := createTestRepository()
	handler := &Handler{repo: repo, templates: nil}

	// Create a recorder to capture the response
	w := httptest.NewRecorder()

	// Call render directly - this should handle nil templates gracefully
	handler.render(w, "nonexistent.tmpl", struct{}{})

	// Should write some error response
	if w.Body.Len() == 0 {
		t.Error("Expected some response body when templates are nil")
	}
}

// Test repository access methods
func TestHandlerRepoAccess(t *testing.T) {
	repo := createTestRepository()
	handler := &Handler{repo: repo}

	// Test that we can access repository stats
	stats := handler.repo.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	// Test getting empty artists list
	artists := handler.repo.GetArtists()
	if artists == nil {
		t.Error("Artists should not be nil, even if empty")
	}

	// Test getting empty locations list
	locations := handler.repo.GetLocations()
	if locations == nil {
		t.Error("Locations should not be nil, even if empty")
	}
}

// Test template-dependent handlers with mock template
func TestHandlersWithoutTemplates(t *testing.T) {
	repo := createTestRepository()
	handler := &Handler{repo: repo, templates: nil}

	tests := []struct {
		name     string
		method   string
		path     string
		handler  func(http.ResponseWriter, *http.Request)
		wantCode int
	}{
		{"Home", "GET", "/", handler.Home, http.StatusInternalServerError},
		{"Home Invalid Method", "POST", "/", handler.Home, http.StatusMethodNotAllowed},
		{"Home Invalid Path", "GET", "/invalid", handler.Home, http.StatusInternalServerError},
		{"Artists", "GET", "/artists", handler.Artists, http.StatusInternalServerError},
		{"Artists Invalid Method", "POST", "/artists", handler.Artists, http.StatusMethodNotAllowed},
		{"Artists Invalid Path", "GET", "/artists/invalid", handler.Artists, http.StatusInternalServerError},
		{"Artist Detail Missing", "GET", "/artists/nonexistent", handler.ArtistDetail, http.StatusInternalServerError},
		{"Artist Detail Empty", "GET", "/artists/", handler.ArtistDetail, http.StatusInternalServerError},
		{"Artist Detail Invalid Method", "POST", "/artists/1", handler.ArtistDetail, http.StatusMethodNotAllowed},
		{"Locations", "GET", "/locations", handler.Locations, http.StatusInternalServerError},
		{"Locations Invalid Method", "POST", "/locations", handler.Locations, http.StatusMethodNotAllowed},
		{"Locations Invalid Path", "GET", "/locations/invalid", handler.Locations, http.StatusInternalServerError},
		{"Location Detail Missing", "GET", "/locations/nonexistent", handler.LocationDetail, http.StatusInternalServerError},
		{"Location Detail Empty", "GET", "/locations/", handler.LocationDetail, http.StatusInternalServerError},
		{"Location Detail Invalid Method", "POST", "/locations/test", handler.LocationDetail, http.StatusMethodNotAllowed},
		{"NotFound", "GET", "/404", handler.NotFound, http.StatusInternalServerError},
		{"InternalError", "GET", "/500", func(w http.ResponseWriter, r *http.Request) {
			handler.InternalError(w, r, "Test error")
		}, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}

			// All template-dependent handlers should write something to the body
			if w.Body.Len() == 0 {
				t.Error("Expected response body")
			}
		})
	}
}

func TestStaticFilesHandler(t *testing.T) {
	repo := createTestRepository()
	handler := &Handler{repo: repo}

	tests := []struct {
		name     string
		path     string
		method   string
		wantCode int
	}{
		{
			name:     "Static file not found returns 404",
			path:     "/static/nonexistent.css",
			method:   "GET",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Favicon not found returns 404",
			path:     "/favicon.ico",
			method:   "GET",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Non-static path returns 404",
			path:     "/some/random/path",
			method:   "GET",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Invalid method returns 405",
			path:     "/static/test.css",
			method:   "POST",
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			name:     "Invalid method for favicon returns 405",
			path:     "/favicon.ico",
			method:   "POST",
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.StaticFiles(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

// --- Begin merged dev tests from handlers_dev_test.go ---

func setupHandler(t *testing.T) *Handler {
	repo := repository.NewRepository("https://groupietrackers.herokuapp.com", 10*time.Second)
	if err := repo.LoadData(context.Background()); err != nil {
		t.Fatalf("failed to load data for tests: %v", err)
	}

	// Create minimal in-memory templates to avoid filesystem dependencies.
	tplText := `{{define "error.tmpl"}}<html><body><h1>{{.Title}}</h1><p>{{.Message}}</p></body></html>{{end}}` +
		`{{define "artists.tmpl"}}<html><body><h1>Artists</h1></body></html>{{end}}` +
		`{{define "home.tmpl"}}<html><body><h1>Home</h1></body></html>{{end}}`

	tpl := template.New("")
	if _, err := tpl.Parse(strings.TrimSpace(tplText)); err != nil {
		t.Fatalf("failed to parse in-memory templates: %v", err)
	}

	return &Handler{repo: repo, templates: tpl}
}

func TestDev404(t *testing.T) {
	h := setupHandler(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/404", nil)
	h.Dev404(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if body := rr.Body.String(); body == "" {
		t.Fatalf("expected non-empty body for 404")
	}
}

func TestDev500(t *testing.T) {
	h := setupHandler(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/500", nil)
	h.Dev500(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDevTemplateError(t *testing.T) {
	h := setupHandler(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/template-error", nil)
	h.DevTemplateError(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if got := rr.Body.String(); got == "" {
		t.Fatalf("expected non-empty body for template error")
	}
}

func TestDevPanicRecoveredByMiddleware(t *testing.T) {
	h := setupHandler(t)

	// Wrap the DevPanic handler with the recovery middleware used in server.go
	recovered := withRecoveryLocal(http.HandlerFunc(h.DevPanic))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dev/panic", nil)

	recovered.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected recovered panic to produce 500, got %d", rr.Code)
	}
}

// withRecoveryLocal mirrors the panic recovery middleware from the server
// to allow unit testing without importing the server package.
func withRecoveryLocal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Return a simple 500 response like server middleware
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- End merged dev tests ---
