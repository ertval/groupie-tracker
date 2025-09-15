package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

// TestDevPanicEndpoint ensures the /dev/trigger-panic route returns HTTP 500 and is handled by InternalErrorHandler.
func TestDevPanicEndpoint(t *testing.T) {
	// Prepare minimal store with one artist
	store := storage.NewStore()
	store.LoadData(models.APIResponse{
		Artists: []models.Artist{{ID: 1, Name: "Panic Test"}},
	})

	apiClient := api.NewClient(DefaultAPIURL, 5*time.Second)
	svc := service.NewService(store)
	h := handlers.NewHandlers(store, svc, apiClient)

	// Build router and middleware
	mux := createRouter(h)

	// Request the dev panic endpoint
	req := httptest.NewRequest(http.MethodGet, "/dev/trigger-panic", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d, body: %q", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}
}
