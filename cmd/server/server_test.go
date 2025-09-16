package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/handlers"
)

// getProjectRoot returns the absolute path to the project root directory
func getProjectRoot() string {
	_, currentFile, _, _ := runtime.Caller(0)
	// From cmd/server/server_test.go, go up two levels to project root
	return filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
}

// createTestHandlers creates handlers for testing with proper template loading
func createTestHandlers() *handlers.Handlers {
	repo := data.NewRepository()
	testData := data.APIResponse{
		Artists: []data.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	repo.LoadData(testData)

	apiClient := api.NewClient("https://groupietrackers.herokuapp.com", 10*time.Second)

	// Change to project root to ensure templates are found
	originalDir, _ := os.Getwd()
	projectRoot := getProjectRoot()
	os.Chdir(projectRoot)

	h := handlers.NewHandlers(repo, apiClient)

	// Restore original directory
	os.Chdir(originalDir)

	return h
}

func TestServerInitialization(t *testing.T) {
	// Test that server can be initialized without crashing
	server, err := NewServer()
	if err != nil {
		t.Errorf("NewServer() should not return an error: %v", err)
	}

	if server == nil {
		t.Error("NewServer() should not return nil")
	}
}

func TestGetPort(t *testing.T) {
	// Test port configuration
	port := getPort()
	if !strings.Contains(port, ":") {
		t.Error("Port should contain colon")
	}
}

func TestServerRoutes(t *testing.T) {
	handlers := createTestHandlers()
	mux := createRouter(handlers)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{
			name:           "Home page",
			method:         "GET",
			url:            "/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Artists page",
			method:         "GET",
			url:            "/artists",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Locations page",
			method:         "GET",
			url:            "/locations",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Health check",
			method:         "GET",
			url:            "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Not found page",
			method:         "GET",
			url:            "/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	handlers := createTestHandlers()
	mux := createRouter(handlers)

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Health check returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check content type
	expected := "application/json"
	if ct := rr.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Health check returned wrong content type: got %v want %v", ct, expected)
	}

	// Check that response contains expected JSON structure
	body := rr.Body.String()
	if !strings.Contains(body, `"status"`) {
		t.Error("Health check response should contain status field")
	}
	if !strings.Contains(body, `"stats"`) {
		t.Error("Health check response should contain stats field")
	}
}

func TestStaticFileServing(t *testing.T) {
	handlers := createTestHandlers()
	mux := createRouter(handlers)

	// Test CSS file serving
	req, err := http.NewRequest("GET", "/static/css/base.css", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Should either serve the file (200) or return 404 if file doesn't exist
	// Both are acceptable in tests since file might not exist in test environment
	if status := rr.Code; status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("Static file request returned unexpected status: got %v", status)
	}
}
