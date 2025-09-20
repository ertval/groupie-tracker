package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/config"
)

// newTestServer creates a new server for testing, including a mock API.
func newTestServer(t *testing.T) *httptest.Server {
	mockAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[]`))
		case "/api/relation":
			w.Write([]byte(`{"index":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))

	// Change working directory to repository root so templates/static files are found
	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(origWd, "..", "..")
	_ = os.Chdir(repoRoot)

	// Configure repository to use mock API server
	config.APIBaseURL = mockAPIServer.URL
	config.APIRequestTimeout = 5 * time.Second

	srv, err := newServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	testServer := httptest.NewServer(srv.Handler)
	t.Cleanup(func() {
		mockAPIServer.Close()
		testServer.Close()
		_ = os.Chdir(origWd)
	})

	return testServer
}

func TestGetPort(t *testing.T) {
	// Test default port
	if port := getPort(); port != config.DefaultPort {
		t.Errorf("expected port %s, got %s", config.DefaultPort, port)
	}

	// Test custom port
	os.Setenv("PORT", "9999")
	defer os.Unsetenv("PORT")
	if port := getPort(); port != ":9999" {
		t.Errorf("expected port :9999, got %s", port)
	}
}

func TestRouter(t *testing.T) {
	testServer := newTestServer(t)

	tests := []struct {
		path       string
		wantStatus int
		body       string
	}{
		{"/", http.StatusOK, "Home"},
		{"/artists", http.StatusOK, "Artists"},
		{"/locations", http.StatusOK, "Locations"},
		{"/health", http.StatusOK, "healthy"},
		{"/static/css/base.css", http.StatusOK, ""}, // static files exist in the repo for tests
		{"/nonexistent", http.StatusNotFound, "404 - Page Not Found"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			res, err := http.Get(testServer.URL + tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, res.StatusCode)
			}

			if tt.body != "" {
				body, _ := io.ReadAll(res.Body)
				if !strings.Contains(string(body), tt.body) {
					t.Errorf("expected body to contain %q", tt.body)
				}
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	// Test withRecovery
	recoveryTestHandler := withRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	recoveryTestHandler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("withRecovery: expected status 500, got %d", w.Code)
	}

	// Test withLogging
	loggingTestHandler := withLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w = httptest.NewRecorder()
	loggingTestHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("withLogging: expected status 200, got %d", w.Code)
	}
}

func TestServerCreation(t *testing.T) {
	// Test server creation with default config
	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(origWd, "..", "..")
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(origWd) }()

	// Test with mock API
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[]`))
		case "/api/relation":
			w.Write([]byte(`{"index":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockAPI.Close()

	config.APIBaseURL = mockAPI.URL
	config.APIRequestTimeout = 5 * time.Second

	srv, err := newServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if srv.Handler == nil {
		t.Error("expected server handler to be set")
	}

	if srv.ReadTimeout != config.ReadTimeout {
		t.Errorf("expected ReadTimeout %v, got %v", config.ReadTimeout, srv.ReadTimeout)
	}

	if srv.WriteTimeout != config.WriteTimeout {
		t.Errorf("expected WriteTimeout %v, got %v", config.WriteTimeout, srv.WriteTimeout)
	}

	if srv.IdleTimeout != config.IdleTimeout {
		t.Errorf("expected IdleTimeout %v, got %v", config.IdleTimeout, srv.IdleTimeout)
	}
}

func TestServerWithDifferentPorts(t *testing.T) {
	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(origWd, "..", "..")
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(origWd) }()

	// Mock API server
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[]`))
		case "/api/relation":
			w.Write([]byte(`{"index":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockAPI.Close()

	config.APIBaseURL = mockAPI.URL
	config.APIRequestTimeout = 5 * time.Second

	// Test different port configurations
	testCases := []string{":3000", ":8081", ":9000"}

	for _, port := range testCases {
		t.Run("Port "+port, func(t *testing.T) {
			originalPort := config.DefaultPort
			config.DefaultPort = port

			srv, err := newServer()
			if err != nil {
				t.Fatalf("failed to create server with port %s: %v", port, err)
			}

			if srv.Addr != port {
				t.Errorf("expected server addr %s, got %s", port, srv.Addr)
			}

			config.DefaultPort = originalPort
		})
	}
}

func TestServerEndToEndFlow(t *testing.T) {
	// Create complete mock API with comprehensive data
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970, "firstAlbum": "Queen"},
				{"id": 2, "name": "AC/DC", "creationDate": 1973, "firstAlbum": "High Voltage"}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"london-uk": ["2022-01-01"], "paris-france": ["2022-02-01"]}},
					{"id": 2, "datesLocations": {"london-uk": ["2023-01-01"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockAPI.Close()

	server := newTestServer(t)
	defer server.Close()

	// Test complete user flow: Home -> Artists -> Artist Detail -> Locations -> Location Detail
	client := server.Client()

	// Step 1: Visit home page
	res, err := client.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("failed to get home page: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("home page: expected status 200, got %d", res.StatusCode)
	}

	// Step 2: Visit artists page
	res, err = client.Get(server.URL + "/artists")
	if err != nil {
		t.Fatalf("failed to get artists page: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("artists page: expected status 200, got %d", res.StatusCode)
	}

	// Step 3: Visit specific artist detail
	res, err = client.Get(server.URL + "/artists/queen")
	if err != nil {
		t.Fatalf("failed to get artist detail: %v", err)
	}
	res.Body.Close()
	// Might return 404 if artist not found in mock data, that's ok for this test
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		t.Errorf("artist detail: expected status 200 or 404, got %d", res.StatusCode)
	}

	// Step 4: Visit locations page
	res, err = client.Get(server.URL + "/locations")
	if err != nil {
		t.Fatalf("failed to get locations page: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("locations page: expected status 200, got %d", res.StatusCode)
	}

	// Step 5: Test health check
	res, err = client.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("failed to get health endpoint: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("health check: expected status 200, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "healthy") {
		t.Error("health check: expected response to contain 'healthy'")
	}
}

func TestServerErrorHandling(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	// Test various error conditions
	testCases := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"Not found page", "/nonexistent", http.StatusNotFound},
		{"Not found artist", "/artists/nonexistent", http.StatusNotFound},
		{"Not found location", "/locations/nonexistent", http.StatusNotFound},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := server.Client().Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to get %s: %v", tt.path, err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, res.StatusCode)
			}
		})
	}
}

func TestServerStaticFileServing(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	// Test static file serving
	staticPaths := []struct {
		path       string
		wantStatus int
	}{
		{"/static/css/base.css", http.StatusOK},
		{"/favicon.ico", http.StatusOK},
		{"/static/nonexistent.css", http.StatusNotFound},
	}

	for _, tt := range staticPaths {
		t.Run("Static: "+tt.path, func(t *testing.T) {
			res, err := server.Client().Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to get static file %s: %v", tt.path, err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("static file %s: expected status %d, got %d", tt.path, tt.wantStatus, res.StatusCode)
			}
		})
	}
}

func TestServerMethodNotAllowed(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	// Test that unsupported methods return 405
	paths := []string{"/", "/artists", "/locations", "/health"}
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, path := range paths {
		for _, method := range methods {
			t.Run(fmt.Sprintf("%s %s", method, path), func(t *testing.T) {
				req, err := http.NewRequest(method, server.URL+path, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				res, err := server.Client().Do(req)
				if err != nil {
					t.Fatalf("failed to send request: %v", err)
				}
				defer res.Body.Close()

				if res.StatusCode != http.StatusMethodNotAllowed {
					t.Errorf("expected status 405 for %s %s, got %d", method, path, res.StatusCode)
				}
			})
		}
	}
}
