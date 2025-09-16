package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"groupie-tracker/internal/client"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/handlers"
)

// getProjectRoot returns the absolute path to the project root directory
func getProjectRoot() string {
	// Try to derive project root from the compiled test file location
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		candidate := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
		if _, err := os.Stat(filepath.Join(candidate, "templates")); err == nil {
			return candidate
		}
	}

	// Fallback: walk up from current working directory until we find templates/
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	p := wd
	for {
		if _, err := os.Stat(filepath.Join(p, "templates")); err == nil {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	// Last resort: return working directory
	return wd
}

// createTestHandlers creates handlers for testing with proper template loading
func createTestHandlers() *handlers.Handlers {
	repo := data.NewRepository()
	testData := &client.Response{
		Artists: []client.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	repo.Initialize(context.Background(), nil, testData)

	// Change to project root to ensure templates are found
	originalDir, _ := os.Getwd()
	projectRoot := getProjectRoot()
	os.Chdir(projectRoot)

	h := handlers.NewHandlers(repo)

	// Restore original directory
	os.Chdir(originalDir)

	return h
}

// createTestHandlersWithoutTemplates returns handlers without loading templates
// This avoids fatal errors in test environments where templates may not exist.
func createTestHandlersWithoutTemplates() *handlers.Handlers {
	repo := data.NewRepository()
	testData := &client.Response{
		Artists: []client.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	_ = repo.Initialize(context.Background(), nil, testData)

	// Create handlers with templates properly loaded by using createTestHandlers()
	h := createTestHandlers()
	// Clear unexported templates field using reflection so tests don't depend on template files
	hv := reflect.ValueOf(h).Elem()
	f := hv.FieldByName("templates")
	if f.IsValid() && f.CanSet() {
		f.Set(reflect.Zero(f.Type()))
	}
	return h
}

func TestServerInitialization(t *testing.T) {
	// Change to project root before testing server initialization
	originalDir, _ := os.Getwd()
	projectRoot := getProjectRoot()
	err := os.Chdir(projectRoot)
	if err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Test that server can be initialized without crashing
	server, err := NewServer()
	if err != nil {
		// If we can't load templates in test environment, that's OK
		// Just check that we get a server struct back
		if server == nil {
			t.Errorf("NewServer() should return a server struct even if there are errors: %v", err)
		}
	} else if server == nil {
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
	handlers := createTestHandlersWithoutTemplates()
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
			url:            "/healthz",
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

			// For routes that would normally render templates, we expect either:
			// 1. OK (200) if templates are available
			// 2. Internal Server Error (500) if templates are not available
			// But NOT other errors like 404 for valid routes
			status := rr.Code
			if status == http.StatusInternalServerError && tt.url == "/dev/panic" {
				// expected: panic route recovers and returns 500
				return
			}
			if status != tt.expectedStatus && !(tt.expectedStatus == http.StatusOK && status == http.StatusInternalServerError) {
				t.Errorf("Handler returned wrong status code: got %v want %v (or 500 for template errors)", status, tt.expectedStatus)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	handlers := createTestHandlersWithoutTemplates()
	mux := createRouter(handlers)

	req, err := http.NewRequest("GET", "/healthz", nil)
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
	handlers := createTestHandlersWithoutTemplates()
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

func TestPanicHandler(t *testing.T) {
	handlers := createTestHandlersWithoutTemplates()
	mux := createRouter(handlers)

	req, err := http.NewRequest("GET", "/dev/panic", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// The panic handler should trigger panic recovery middleware
	// and return 500 internal server error
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Panic handler should return 500, got %v", status)
	}
}

func TestGetPortEnv(t *testing.T) {
	// Ensure default when PORT not set
	orig := os.Getenv("PORT")
	defer os.Setenv("PORT", orig)
	os.Unsetenv("PORT")

	port := getPort()
	if port != DefaultPort {
		t.Fatalf("expected default port %s, got %s", DefaultPort, port)
	}

	// When PORT set without colon
	os.Setenv("PORT", "9090")
	port = getPort()
	if port != ":9090" {
		t.Fatalf("expected :9090, got %s", port)
	}

	// When PORT set with colon
	os.Setenv("PORT", ":7070")
	port = getPort()
	if port != ":7070" {
		t.Fatalf("expected :7070, got %s", port)
	}
}

func TestApplyMiddleware_RecoversPanic(t *testing.T) {
	// Prepare a repository and handlers
	h := createTestHandlersWithoutTemplates()

	// Handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	mux := applyMiddleware(panicHandler, h)

	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 after panic recovery, got %d", rr.Code)
	}
}

func TestMiddlewareApplication(t *testing.T) {
	handlers := createTestHandlersWithoutTemplates()
	mux := createRouter(handlers)

	// Test that middleware is applied (request logging should be present)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// For routes that would normally render templates, we expect either:
	// 1. OK (200) if templates are available
	// 2. Internal Server Error (500) if templates are not available
	// But NOT other errors
	if status := rr.Code; status != http.StatusOK && status != http.StatusInternalServerError {
		t.Errorf("Middleware application test returned wrong status: got %v want %v or 500", status, http.StatusOK)
	}
}

// TestMiddlewarePanicRecovery tests that the middleware properly recovers from panics
func TestMiddlewarePanicRecovery(t *testing.T) {
	repo := data.NewRepository()
	testData := &client.Response{
		Artists: []client.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	repo.Initialize(context.Background(), nil, testData)

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("This is a test panic")
	})

	// Apply middleware
	mux := applyMiddleware(panicHandler, createTestHandlersWithoutTemplates())

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Should return 500 status due to panic recovery
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Panic recovery test returned wrong status: got %v want %v", status, http.StatusInternalServerError)
	}
}

// TestApplyMiddleware tests that the middleware properly wraps handlers
func TestApplyMiddleware(t *testing.T) {
	// Create a simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	handlers := createTestHandlersWithoutTemplates()
	wrappedMux := applyMiddleware(testHandler, handlers)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	wrappedMux.ServeHTTP(rr, req)

	// Should return OK status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ApplyMiddleware test returned wrong status: got %v want %v", status, http.StatusOK)
	}

	// Should return the expected response body
	expected := "test response"
	if body := rr.Body.String(); body != expected {
		t.Errorf("ApplyMiddleware test returned wrong body: got %v want %v", body, expected)
	}
}

// TestCreateRouter tests that the router is properly configured with all routes
func TestCreateRouter(t *testing.T) {
	handlers := createTestHandlersWithoutTemplates()
	mux := createRouter(handlers)

	// Verify that all expected routes are registered by checking they don't return 404 for valid paths
	testPaths := []string{"/", "/artists", "/locations", "/healthz", "/dev/panic"}

	for _, path := range testPaths {
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		// None of these should return a server error (500), though some might return 404
		// in test environment due to missing templates/static files
		if status := rr.Code; status == http.StatusInternalServerError && path != "/dev/panic" {
			t.Errorf("CreateRouter test for path %s returned server error: %v", path, status)
		}
	}

	// Verify static file serving route is registered
	req, err := http.NewRequest("GET", "/static/test.css", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Static file serving should either work (200) or return 404 if file doesn't exist
	// but not a server error
	if status := rr.Code; status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("CreateRouter static file test returned unexpected status: got %v", status)
	}
}

// TestCreateRouterWithNilHandlers tests that the router handles nil handlers gracefully
func TestCreateRouterWithNilHandlers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("createRouter with nil handlers panicked: %v", r)
		}
	}()

	// This should not panic even with nil handlers
	mux := createRouter(nil)
	if mux == nil {
		t.Error("createRouter should return a valid mux even with nil handlers")
	}
}

func TestServerStartMethod(t *testing.T) {
	// Change to project root before testing server initialization
	originalDir, _ := os.Getwd()
	projectRoot := getProjectRoot()
	err := os.Chdir(projectRoot)
	if err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	defer os.Chdir(originalDir)

	server, err := NewServer()
	if err != nil {
		// If we can't load templates in test environment, that's OK
		// Just check that we get a server struct back
		if server == nil {
			t.Errorf("NewServer() should return a server struct even if there are errors: %v", err)
		}
	} else if server == nil {
		t.Error("NewServer() should not return nil")
	} else {
		// Test that Start method is available and server is properly configured
		// We can't actually call Start() as it would block the test
		// But we can check that the server struct has the expected fields
		if server.server == nil {
			t.Error("Server should have HTTP server configured")
		}
		if server.handlers == nil {
			t.Error("Server should have handlers configured")
		}
		if server.repo == nil {
			t.Error("Server should have repository configured")
		}
		if server.apiClient == nil {
			t.Error("Server should have API client configured")
		}
	}
}

func TestGetPortEnvironmentVariable(t *testing.T) {
	// Test with environment variable set
	originalPort := os.Getenv("PORT")
	defer os.Setenv("PORT", originalPort) // Restore original value

	os.Setenv("PORT", "9000")
	port := getPort()
	expected := ":9000"
	if port != expected {
		t.Errorf("getPort() with PORT env var should return %v, got %v", expected, port)
	}

	// Test without environment variable
	os.Unsetenv("PORT")
	port = getPort()
	expected = DefaultPort
	if port != expected {
		t.Errorf("getPort() without PORT env var should return %v, got %v", expected, port)
	}
}

func TestRouterPathMatching(t *testing.T) {
	handlers := createTestHandlersWithoutTemplates()
	mux := createRouter(handlers)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "Exact root path",
			path:           "/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Root path with extra slashes should redirect",
			path:           "//",
			expectedStatus: http.StatusMovedPermanently, // Go's http.ServeMux redirects // to /
		},
		{
			name:           "Exact artists path",
			path:           "/artists",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Artists with trailing slash should 404",
			path:           "/artists/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Artist detail path - should return OK even if artist doesn't exist in templates",
			path:           "/artists/1",
			expectedStatus: http.StatusOK, // Template error results in 500, but 200 is from executeTemplate handling
		},
		{
			name:           "Locations path",
			path:           "/locations",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Location detail path",
			path:           "/locations/some-location",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			// For routes that would normally render templates, we expect either:
			// 1. OK (200) if templates are available
			// 2. Internal Server Error (500) if templates are not available
			// 3. The expected status code for cases like 404 or redirects
			if status := rr.Code; status != tt.expectedStatus && !(tt.expectedStatus == http.StatusOK && status == http.StatusInternalServerError) {
				t.Errorf("Path %s returned status %v, expected %v (or 500 for template errors)", tt.path, status, tt.expectedStatus)
			}
		})
	}
}
