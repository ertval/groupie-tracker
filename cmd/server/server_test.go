package main

import (
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
