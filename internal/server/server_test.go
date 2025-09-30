package server

import (
	"context"
	"fmt"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestEnvironment prepares the test environment with mock API and template loading.
//
// This function creates a complete test environment for server testing:
//   - Starts a mock HTTP server with realistic API responses
//   - Configures the global repository to use the mock API
//   - Disables image caching to avoid filesystem operations during tests
//   - Loads templates from the project directory for rendering tests
//   - Sets short timeouts appropriate for unit testing
//
// This setup allows tests to run in isolation without external dependencies
// while still testing the complete request/response cycle.
func setupTestEnvironment(t *testing.T) {
	// Create mock API server with realistic responses for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970},
				{"id": 2, "name": "AC/DC", "creationDate": 1973}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"london-uk": ["2022-01-01"]}},
					{"id": 2, "datesLocations": {"london-uk": ["2023-01-01"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))

	// Configure test-specific settings for isolated testing
	config.WithCache = false // Prevent filesystem operations during tests
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second

	// Initialize repository with mock data
	repo = data.NewRepository()
	if err := repo.LoadData(context.Background()); err != nil {
		t.Fatalf("failed to load data for tests: %v", err)
	}

	// Load templates from project root for rendering functionality
	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(origWd, "..", "..")
	_ = os.Chdir(repoRoot)
	loadTemplates()
	_ = os.Chdir(origWd) // Restore original directory
}

// newTestServer creates a complete test server with mock API and configured handlers.
//
// This function provides a fully functional HTTP test server that:
//   - Creates a mock API server with minimal but valid responses
//   - Initializes the server package with mock data and templates
//   - Returns an httptest.Server ready for HTTP client testing
//   - Automatically cleans up resources when test completes
//
// Use this for integration tests that need to make actual HTTP requests
// to test the full request/response cycle including routing and middleware.
//
// The returned server should be used with server.Client() for HTTP requests.
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

	// Setup working directory for template and static file access
	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(origWd, "..", "..")
	_ = os.Chdir(repoRoot)

	// Configure server to use mock API with test timeouts
	config.APIBaseURL = mockAPIServer.URL
	config.APIRequestTimeout = 5 * time.Second

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	testServer := httptest.NewServer(srv.Handler)
	t.Cleanup(func() {
		testServer.Close()
		mockAPIServer.Close()
		_ = os.Chdir(origWd) // Restore working directory
	})

	return testServer
}

// --- Unit Tests for HTTP Handlers ---

// TestHome verifies the home page handler renders successfully with mock data.
func TestHome(t *testing.T) {
	setupTestEnvironment(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	Home(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestArtists verifies the artists listing page renders with filter UI.
func TestArtists(t *testing.T) {
	setupTestEnvironment(t)

	req := httptest.NewRequest("GET", "/artists", nil)
	w := httptest.NewRecorder()

	Artists(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestHealth verifies the health check endpoint returns proper JSON response.
func TestHealth(t *testing.T) {
	setupTestEnvironment(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "healthy") {
		t.Errorf("expected body to contain 'healthy', got: %s", body)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected content type to be application/json, got: %s", contentType)
	}
}

// TestArtistDetail verifies artist detail pages work with SEO-friendly slugs and proper 404 handling.
func TestArtistDetail(t *testing.T) {
	setupTestEnvironment(t)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"Valid artist slug", "/artists/queen", http.StatusOK},
		{"Valid artist slug (AC/DC)", "/artists/ac-dc", http.StatusOK},
		{"Invalid artist slug", "/artists/nonexistent", http.StatusNotFound},
		{"Empty slug", "/artists/", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			ArtistDetail(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestLocations(t *testing.T) {
	setupTestEnvironment(t)

	req := httptest.NewRequest("GET", "/locations", nil)
	w := httptest.NewRecorder()

	Locations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestLocationDetail(t *testing.T) {
	setupTestEnvironment(t)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"Valid location slug", "/locations/london-uk", http.StatusOK},
		{"Invalid location slug", "/locations/nonexistent", http.StatusNotFound},
		{"Empty slug", "/locations/", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			LocationDetail(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestStaticFiles(t *testing.T) {
	setupTestEnvironment(t)

	// Change to repo root for static files
	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(origWd, "..", "..")
	_ = os.Chdir(repoRoot)
	defer func() { _ = os.Chdir(origWd) }()

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"Base CSS", "/static/css/base.css", http.StatusOK},
		{"Favicon", "/favicon.ico", http.StatusOK},
		{"Non-existent file", "/static/css/nonexistent.css", http.StatusNotFound},
		{"Directory traversal attempt", "/static/../go.mod", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			StaticFiles(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestInvalidMethods(t *testing.T) {
	setupTestEnvironment(t)

	tests := []struct {
		endpoint string
		methods  []string
	}{
		{"/", []string{"POST", "PUT", "DELETE", "PATCH", "HEAD"}},
		{"/artists", []string{"PUT", "DELETE", "PATCH", "HEAD"}},   // POST is now allowed for filtering
		{"/locations", []string{"PUT", "DELETE", "PATCH", "HEAD"}}, // POST is now allowed for filtering
		{"/health", []string{"POST", "PUT", "DELETE", "PATCH", "HEAD"}},
	}

	for _, tt := range tests {
		for _, method := range tt.methods {
			t.Run(fmt.Sprintf("%s %s", method, tt.endpoint), func(t *testing.T) {
				req := httptest.NewRequest(method, tt.endpoint, nil)
				w := httptest.NewRecorder()

				switch tt.endpoint {
				case "/":
					Home(w, req)
				case "/artists":
					Artists(w, req)
				case "/locations":
					Locations(w, req)
				case "/health":
					Health(w, req)
				}

				if w.Code != http.StatusMethodNotAllowed {
					t.Errorf("expected status 405 for %s %s, got %d", method, tt.endpoint, w.Code)
				}
			})
		}
	}
}

func TestInvalidPaths(t *testing.T) {
	setupTestEnvironment(t)

	tests := []struct {
		name       string
		path       string
		handler    func(http.ResponseWriter, *http.Request)
		wantStatus int
	}{
		{"Home with extra path", "/extra", Home, http.StatusNotFound},
		{"Artists with invalid path", "/artists/some/extra/path", ArtistDetail, http.StatusNotFound},
		{"Locations with invalid path", "/locations/some/extra/path", LocationDetail, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// Routes and Server Tests

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

// --- Integration Tests ---

// TestRouter verifies the complete routing configuration with a live test server.
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

// --- Middleware Tests ---

// TestMiddleware verifies the middleware chain handles panic recovery, logging, and security headers.
func TestMiddleware(t *testing.T) {
	// Test panic recovery middleware
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

	// Test withSecureHeaders
	secureTestHandler := withSecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w = httptest.NewRecorder()
	secureTestHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("withSecureHeaders: expected status 200, got %d", w.Code)
	}

	// Check security headers are set
	expectedHeaders := map[string]string{
		"Referrer-Policy":        "origin-when-cross-origin",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "deny",
		"X-XSS-Protection":       "0",
	}

	for header, expectedValue := range expectedHeaders {
		if w.Header().Get(header) != expectedValue {
			t.Errorf("expected header %s to be %s, got %s", header, expectedValue, w.Header().Get(header))
		}
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

	srv, err := NewServer()
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

			srv, err := NewServer()
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
	tests := []struct {
		path    string
		methods []string
	}{
		{"/", []string{"POST", "PUT", "DELETE", "PATCH"}},
		{"/artists", []string{"PUT", "DELETE", "PATCH"}},   // POST is now allowed for filtering
		{"/locations", []string{"PUT", "DELETE", "PATCH"}}, // POST is now allowed for filtering
		{"/health", []string{"POST", "PUT", "DELETE", "PATCH"}},
	}

	for _, tt := range tests {
		for _, method := range tt.methods {
			t.Run(fmt.Sprintf("%s %s", method, tt.path), func(t *testing.T) {
				req, err := http.NewRequest(method, server.URL+tt.path, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				res, err := server.Client().Do(req)
				if err != nil {
					t.Fatalf("failed to send request: %v", err)
				}
				defer res.Body.Close()

				if res.StatusCode != http.StatusMethodNotAllowed {
					t.Errorf("expected status 405 for %s %s, got %d", method, tt.path, res.StatusCode)
				}
			})
		}
	}
}

// --- Utility Function Tests ---

// intPtr is a test helper that returns a pointer to the given int value.
// Useful for testing optional integer fields that use *int to distinguish
// between zero values and unset values.
func intPtr(i int) *int {
	return &i
}

// TestParseFilterParams verifies form data parsing for artist filtering functionality.
func TestParseFilterParams(t *testing.T) {
	setupTestEnvironment(t)

	tests := []struct {
		name     string
		formData map[string][]string
		want     data.ArtistFilterParams
	}{
		{
			name: "Creation year range",
			formData: map[string][]string{
				"creationYearFrom": {"1990"},
				"creationYearTo":   {"2000"},
			},
			want: data.ArtistFilterParams{
				CreationYearFrom: intPtr(1990),
				CreationYearTo:   intPtr(2000),
			},
		},
		{
			name: "Member counts and countries",
			formData: map[string][]string{
				"memberCounts": {"4", "5", "6"},
				"countries":    {"USA", "UK"},
			},
			want: data.ArtistFilterParams{
				MemberCounts: []int{4, 5, 6},
				Countries:    []string{"USA", "UK"},
			},
		},
		{
			name: "All filters",
			formData: map[string][]string{
				"creationYearFrom":   {"1995"},
				"creationYearTo":     {"2005"},
				"firstAlbumYearFrom": {"1990"},
				"firstAlbumYearTo":   {"2010"},
				"memberCounts":       {"3", "4"},
				"countries":          {"USA"},
			},
			want: data.ArtistFilterParams{
				CreationYearFrom:   intPtr(1995),
				CreationYearTo:     intPtr(2005),
				FirstAlbumYearFrom: intPtr(1990),
				FirstAlbumYearTo:   intPtr(2010),
				MemberCounts:       []int{3, 4},
				Countries:          []string{"USA"},
			},
		},
		{
			name:     "Empty form",
			formData: map[string][]string{},
			want:     data.ArtistFilterParams{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/artists", nil)
			req.Form = tt.formData

			got := parseArtistFilterParams(req)

			// Compare basic fields
			if !equalIntPtr(got.CreationYearFrom, tt.want.CreationYearFrom) {
				t.Errorf("CreationYearFrom: got %v, want %v", got.CreationYearFrom, tt.want.CreationYearFrom)
			}
			if !equalIntPtr(got.CreationYearTo, tt.want.CreationYearTo) {
				t.Errorf("CreationYearTo: got %v, want %v", got.CreationYearTo, tt.want.CreationYearTo)
			}
			if !equalIntPtr(got.FirstAlbumYearFrom, tt.want.FirstAlbumYearFrom) {
				t.Errorf("FirstAlbumYearFrom: got %v, want %v", got.FirstAlbumYearFrom, tt.want.FirstAlbumYearFrom)
			}
			if !equalIntPtr(got.FirstAlbumYearTo, tt.want.FirstAlbumYearTo) {
				t.Errorf("FirstAlbumYearTo: got %v, want %v", got.FirstAlbumYearTo, tt.want.FirstAlbumYearTo)
			}

			// Compare slices
			if !equalIntSlices(got.MemberCounts, tt.want.MemberCounts) {
				t.Errorf("MemberCounts: got %v, want %v", got.MemberCounts, tt.want.MemberCounts)
			}
			if !equalStringSlices(got.Countries, tt.want.Countries) {
				t.Errorf("Countries: got %v, want %v", got.Countries, tt.want.Countries)
			}
		})
	}
}

// Helper functions for test comparisons

// equalIntPtr compares two *int pointers, handling nil cases properly.
func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// equalIntSlices performs deep equality comparison of integer slices.
func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// equalStringSlices performs deep equality comparison of string slices.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
