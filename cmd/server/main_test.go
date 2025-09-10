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

func TestServer_Routes(t *testing.T) {
	// Setup test store with data
	store := storage.NewStore()
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	store.LoadData(testData)

	// Create handlers
	apiClient := api.NewClient("https://groupietrackers.herokuapp.com", 10*time.Second)
	service := service.NewService(store)
	h := handlers.NewHandlers(store, service, apiClient)

	// Create router
	mux := createRouter(h)

	// Test routes
	tests := []struct {
		path         string
		expectedCode int
	}{
		{"/", http.StatusOK},
		{"/artists", http.StatusOK},
		{"/locations", http.StatusOK},
		{"/healthz", http.StatusOK},
		{"/nonexistent", http.StatusNotFound},
	}

	for _, tt := range tests {
		req, err := http.NewRequest("GET", tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != tt.expectedCode {
			t.Errorf("Handler for %s returned wrong status code: got %v want %v", tt.path, status, tt.expectedCode)
		}
	}
}

func TestNewServer(t *testing.T) {
	// This test would require network access, so we'll just test the structure
	t.Skip("Skipping server creation test as it requires network access")
}

func TestGetPort(t *testing.T) {
	port := getPort()
	if port == "" {
		t.Error("getPort() returned empty string")
	}
}
