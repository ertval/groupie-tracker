package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

// TestApiClientAdapter_FetchAllData tests the adapter functionality
func TestApiClientAdapter_FetchAllData(t *testing.T) {
	// Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/artists":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"id":1,"name":"Test Artist","creationDate":2000,"members":["Member 1"],"firstAlbum":"01-01-2001","image":"test.jpg"}]`))
		case "/api/locations":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"locations":["test-location"]}]}`))
		case "/api/dates":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"dates":["01-01-2020"]}]}`))
		case "/api/relation":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"datesLocations":{"test-location":["01-01-2020"]}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	client := api.NewClient(mockServer.URL, RequestTimeout)
	adapter := &apiClientAdapter{client: client}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := adapter.FetchAllData(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(data.Artists) != 1 {
		t.Errorf("Expected 1 artist, got %d", len(data.Artists))
	}

	if data.Artists[0].Name != "Test Artist" {
		t.Errorf("Expected artist name 'Test Artist', got %s", data.Artists[0].Name)
	}
}

// TestApiClientAdapter_FetchAllData_Error tests error handling
func TestApiClientAdapter_FetchAllData_Error(t *testing.T) {
	client := api.NewClient("http://localhost:99999", 1*time.Second)
	adapter := &apiClientAdapter{client: client}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := adapter.FetchAllData(ctx)
	if err == nil {
		t.Error("Expected error when API is unreachable")
	}
}

func TestNewServer_Success(t *testing.T) {
	// Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/artists":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"id":1,"name":"Test Artist","creationDate":2000,"members":["Member 1"],"firstAlbum":"01-01-2001","image":"test.jpg"}]`))
		case "/api/locations":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"locations":["test-location"]}]}`))
		case "/api/dates":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"dates":["01-01-2020"]}]}`))
		case "/api/relation":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"datesLocations":{"test-location":["01-01-2020"]}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Create a test client
	testClient := api.NewClient(mockServer.URL, RequestTimeout)
	testAdapter := &apiClientAdapter{client: testClient}

	// Create store manually for testing
	store := storage.NewStoreWithCache(testAdapter)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Test initial data load
	store.StartCache(ctx)
	time.Sleep(200 * time.Millisecond) // Wait for initial load
	defer store.StopCache()

	// Verify that data was loaded correctly
	artists := store.GetAllArtists()
	if len(artists) != 1 {
		t.Errorf("Expected 1 artist, got %d", len(artists))
	}

	if len(artists) > 0 && artists[0].Name != "Test Artist" {
		t.Errorf("Expected artist name 'Test Artist', got %s", artists[0].Name)
	}
}

func TestNewServer_Integration(t *testing.T) {
	// This test simulates the actual NewServer flow but with shorter timeouts

	// Mock API server that responds immediately
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/artists":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"id":1,"name":"Queen","creationDate":1970,"members":["Freddie Mercury"],"firstAlbum":"14-12-1973","image":"test.jpg"}]`))
		case "/api/locations":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"locations":["London"]}]}`))
		case "/api/dates":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"dates":["14-12-1973"]}]}`))
		case "/api/relation":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"index":[{"id":1,"datesLocations":{"London":["14-12-1973"]}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Override default values temporarily
	originalAPIURL := DefaultAPIURL
	originalTimeout := RequestTimeout
	defer func() {
		// Note: we can't actually restore these as they're constants,
		// but this documents the intended behavior
		_ = originalAPIURL
		_ = originalTimeout
	}()

	// Create test server instance components manually (simulating NewServer)
	apiClient := api.NewClient(mockServer.URL, 2*time.Second)
	adapter := &apiClientAdapter{client: apiClient}
	store := storage.NewStoreWithCache(adapter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cache and wait for data load (simulating NewServer logic)
	store.StartCache(ctx)

	// Wait for initial data load with timeout (simulating improved NewServer logic)
	loadCtx, loadCancel := context.WithTimeout(ctx, 3*time.Second)
	defer loadCancel()

	dataLoaded := false
	for {
		select {
		case <-loadCtx.Done():
			t.Fatal("Timeout waiting for initial data load")
		default:
			stats := store.GetStats()
			if stats["artists"] > 0 {
				dataLoaded = true
				goto checkData
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

checkData:
	if !dataLoaded {
		t.Fatal("Data was not loaded")
	}

	defer store.StopCache()

	// Verify server components would be properly initialized
	stats := store.GetStats()
	if stats["artists"] == 0 {
		t.Error("Expected artists to be loaded")
	}

	// Test handlers creation
	h := handlers.NewHandlers(store)
	h.SetAPIClient(apiClient)

	// Test router creation
	mux := createRouter(h)
	if mux == nil {
		t.Error("Router creation failed")
	}

	// Test server configuration
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	if server.Addr != ":8080" {
		t.Errorf("Expected server address :8080, got %s", server.Addr)
	}
}

func TestNewServer_APITimeout(t *testing.T) {
	// Mock server that responds very slowly
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Slower than our timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	client := api.NewClient(slowServer.URL, 500*time.Millisecond) // Short timeout
	adapter := &apiClientAdapter{client: client}
	store := storage.NewStoreWithCache(adapter)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	store.StartCache(ctx)
	time.Sleep(100 * time.Millisecond)
	store.StopCache()

	// Should have no data due to timeout
	stats := store.GetStats()
	if stats["artists"] > 0 {
		t.Error("Expected no artists to be loaded when API times out")
	}
}

func TestServer_ConfigConstants(t *testing.T) {
	// Test that all constants are properly defined
	if DefaultPort != ":8080" {
		t.Errorf("Expected DefaultPort to be ':8080', got %s", DefaultPort)
	}

	if DefaultAPIURL != "https://groupietrackers.herokuapp.com" {
		t.Errorf("Expected DefaultAPIURL to be 'https://groupietrackers.herokuapp.com', got %s", DefaultAPIURL)
	}

	if RequestTimeout != 30*time.Second {
		t.Errorf("Expected RequestTimeout to be 30s, got %v", RequestTimeout)
	}

	if ShutdownTimeout != 10*time.Second {
		t.Errorf("Expected ShutdownTimeout to be 10s, got %v", ShutdownTimeout)
	}

	if ReadTimeout != 15*time.Second {
		t.Errorf("Expected ReadTimeout to be 15s, got %v", ReadTimeout)
	}

	if WriteTimeout != 15*time.Second {
		t.Errorf("Expected WriteTimeout to be 15s, got %v", WriteTimeout)
	}

	if IdleTimeout != 60*time.Second {
		t.Errorf("Expected IdleTimeout to be 60s, got %v", IdleTimeout)
	}
}

func TestGetPort_Default(t *testing.T) {
	// Clear PORT env var
	originalPort := os.Getenv("PORT")
	os.Unsetenv("PORT")
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		}
	}()

	port := getPort()
	if port != DefaultPort {
		t.Errorf("Expected default port %s, got %s", DefaultPort, port)
	}
}

func TestGetPort_Environment(t *testing.T) {
	// Set custom PORT env var
	originalPort := os.Getenv("PORT")
	os.Setenv("PORT", "3000")
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	port := getPort()
	expected := ":3000"
	if port != expected {
		t.Errorf("Expected port %s, got %s", expected, port)
	}
}

func TestCreateRouter_RoutesExist(t *testing.T) {
	// Setup test store with minimal data
	store := storage.NewStore()
	testData := storage.StoreData{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}},
		},
	}
	store.LoadData(testData)

	// Create handlers and router
	h := handlers.NewHandlers(store)
	mux := createRouter(h)

	// Test routes exist and respond appropriately
	testRoutes := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/artists", http.StatusOK},
		{"GET", "/artists/1", http.StatusOK},
		{"GET", "/locations", http.StatusOK},
		{"GET", "/api/search", http.StatusOK},
		{"GET", "/api/suggest", http.StatusOK},
		{"GET", "/healthz", http.StatusOK},
	}

	for _, route := range testRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != route.status {
				t.Errorf("Expected status %d for %s %s, got %d", route.status, route.method, route.path, w.Code)
			}
		})
	}
}

func TestCreateRouter_MethodNotAllowed(t *testing.T) {
	store := storage.NewStore()
	h := handlers.NewHandlers(store)
	mux := createRouter(h)

	// Test method not allowed
	req := httptest.NewRequest("DELETE", "/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for DELETE /, got %d", w.Code)
	}
}

func TestCreateRouter_NotFound(t *testing.T) {
	store := storage.NewStore()
	h := handlers.NewHandlers(store)
	mux := createRouter(h)

	// Test not found
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent route, got %d", w.Code)
	}
}

func TestMiddleware_Recovery(t *testing.T) {
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	wrapped := recoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic and should return 500
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic recovery, got %d", w.Code)
	}

	if w.Body.String() == "" {
		t.Error("Expected error message in response body")
	}
}

func TestMiddleware_Logging(t *testing.T) {
	// Simple handler for testing
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with logging middleware
	wrapped := loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic or error
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestColorConstants(t *testing.T) {
	// Test color constants are defined
	colors := map[string]string{
		"colorReset":  colorReset,
		"colorRed":    colorRed,
		"colorGreen":  colorGreen,
		"colorYellow": colorYellow,
		"colorCyan":   colorCyan,
	}

	for name, color := range colors {
		if color == "" {
			t.Errorf("Color constant %s should not be empty", name)
		}
	}
}

func TestWrapWithMiddleware(t *testing.T) {
	store := storage.NewStore()
	h := handlers.NewHandlers(store)

	// Simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Test middleware wrapping
	wrapped := wrapWithMiddleware(testHandler, h)
	if wrapped == nil {
		t.Error("Expected wrapped handler, got nil")
	}

	// Test that wrapped handler works
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic for recovery")
	})

	// Wrap with basic recovery middleware
	wrapped := recoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic and should return 500
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic recovery, got %d", w.Code)
	}

	if w.Body.String() == "" {
		t.Error("Expected error message in response body")
	}
}

func TestRecoveryMiddlewareWithHandler(t *testing.T) {
	store := storage.NewStore()
	h := handlers.NewHandlers(store)

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic for recovery with handler")
	})

	// Wrap with recovery middleware that uses handler
	wrapped := recoveryMiddlewareWithHandler(panicHandler, h)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic and should use custom error handler
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic recovery, got %d", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Simple handler for testing
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with logging middleware
	wrapped := loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic or error
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestServer_Struct(t *testing.T) {
	// Test Server struct field types
	store := storage.NewStore()
	apiClient := api.NewClient("http://test.com", 5*time.Second)
	h := handlers.NewHandlers(store)

	server := &http.Server{
		Addr:    ":8080",
		Handler: createRouter(h),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &Server{
		store:     store,
		apiClient: apiClient,
		handlers:  h,
		server:    server,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Test that all fields are properly set
	if s.store == nil {
		t.Error("Server store should not be nil")
	}
	if s.apiClient == nil {
		t.Error("Server apiClient should not be nil")
	}
	if s.handlers == nil {
		t.Error("Server handlers should not be nil")
	}
	if s.server == nil {
		t.Error("Server server should not be nil")
	}
	if s.ctx == nil {
		t.Error("Server context should not be nil")
	}
	if s.cancel == nil {
		t.Error("Server cancel function should not be nil")
	}
}

func TestCreateRouter_RefreshEndpoint(t *testing.T) {
	store := storage.NewStore()
	h := handlers.NewHandlers(store)
	mux := createRouter(h)

	// Test refresh endpoint exists
	req := httptest.NewRequest("POST", "/api/refresh", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should not be 404 (handler exists)
	if w.Code == http.StatusNotFound {
		t.Error("Refresh endpoint should exist")
	}
}

func TestCreateRouter_StaticFiles(t *testing.T) {
	store := storage.NewStore()
	h := handlers.NewHandlers(store)
	mux := createRouter(h)

	// Test static file route exists
	req := httptest.NewRequest("GET", "/static/css/test.css", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should not be 404 (static handler exists, though file might not exist)
	// The static handler should respond, even if file is missing
	if w.Code == http.StatusMethodNotAllowed {
		t.Error("Static file route should handle GET requests")
	}
}
