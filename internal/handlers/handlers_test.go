// Package handlers provides tests for HTTP handlers.
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	if handler == nil {
		t.Fatal("Handler should not be nil")
	}

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
		{"Home", "GET", "/", handler.Home, http.StatusOK},
		{"Home Invalid Method", "POST", "/", handler.Home, http.StatusMethodNotAllowed},
		{"Home Invalid Path", "GET", "/invalid", handler.Home, http.StatusNotFound},
		{"Artists", "GET", "/artists", handler.Artists, http.StatusOK},
		{"Artists Invalid Method", "POST", "/artists", handler.Artists, http.StatusMethodNotAllowed},
		{"Artists Invalid Path", "GET", "/artists/invalid", handler.Artists, http.StatusNotFound},
		{"Artist Detail Missing", "GET", "/artists/nonexistent", handler.ArtistDetail, http.StatusNotFound},
		{"Artist Detail Empty", "GET", "/artists/", handler.ArtistDetail, http.StatusNotFound},
		{"Artist Detail Invalid Method", "POST", "/artists/1", handler.ArtistDetail, http.StatusMethodNotAllowed},
		{"Locations", "GET", "/locations", handler.Locations, http.StatusOK},
		{"Locations Invalid Method", "POST", "/locations", handler.Locations, http.StatusMethodNotAllowed},
		{"Locations Invalid Path", "GET", "/locations/invalid", handler.Locations, http.StatusNotFound},
		{"Location Detail Missing", "GET", "/locations/nonexistent", handler.LocationDetail, http.StatusNotFound},
		{"Location Detail Empty", "GET", "/locations/", handler.LocationDetail, http.StatusNotFound},
		{"Location Detail Invalid Method", "POST", "/locations/test", handler.LocationDetail, http.StatusMethodNotAllowed},
		{"NotFound", "GET", "/404", handler.NotFound, http.StatusNotFound},
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
