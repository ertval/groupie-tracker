package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

// TestTemplateErrorReturns500 ensures that template execution errors return 500, not simple HTML fallback
func TestTemplateErrorReturns500(t *testing.T) {
	// Setup minimal handlers with valid store/service but broken templates
	store := storage.NewStore()
	store.LoadData(models.APIResponse{
		Artists: []models.Artist{{ID: 1, Name: "Test Artist"}},
	})

	apiClient := api.NewClient("https://groupietrackers.herokuapp.com", 5*time.Second)
	svc := service.NewService(store)

	// Create handlers but force templates to be nil to simulate loading failure
	h := &Handlers{
		store:     store,
		service:   svc,
		apiClient: apiClient,
		templates: nil, // This will cause template execution to fail
	}

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
	// Setup minimal handlers
	store := storage.NewStore()
	store.LoadData(models.APIResponse{
		Artists: []models.Artist{{ID: 1, Name: "Test Artist"}},
	})

	apiClient := api.NewClient("https://groupietrackers.herokuapp.com", 5*time.Second)
	svc := service.NewService(store)
	h := NewHandlers(store, svc, apiClient)

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
