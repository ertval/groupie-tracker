package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

func TestNewServerWithEnvironmentVariables(t *testing.T) {
	// Test environment variable handling
	originalPort := os.Getenv("PORT")
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	// Test with custom port
	os.Setenv("PORT", "9090")
	port := getPort()
	expected := ":9090"
	if port != expected {
		t.Errorf("Expected port %s, got %s", expected, port)
	}

	// Test with port that already has colon
	os.Setenv("PORT", ":7070")
	port = getPort()
	expected = ":7070"
	if port != expected {
		t.Errorf("Expected port %s, got %s", expected, port)
	}
}

func TestNewServerConfigConstants(t *testing.T) {
	// Test all the configuration constants are reasonable
	if DefaultPort == "" {
		t.Error("DefaultPort should not be empty")
	}
	if DefaultAPIURL == "" {
		t.Error("DefaultAPIURL should not be empty")
	}
	if RequestTimeout <= 0 {
		t.Error("RequestTimeout should be positive")
	}
	if ShutdownTimeout <= 0 {
		t.Error("ShutdownTimeout should be positive")
	}
	if ReadTimeout <= 0 {
		t.Error("ReadTimeout should be positive")
	}
	if WriteTimeout <= 0 {
		t.Error("WriteTimeout should be positive")
	}
	if IdleTimeout <= 0 {
		t.Error("IdleTimeout should be positive")
	}

	// Test timeout relationships make sense
	if ReadTimeout > RequestTimeout {
		t.Error("ReadTimeout should not exceed RequestTimeout")
	}
	if WriteTimeout > RequestTimeout {
		t.Error("WriteTimeout should not exceed RequestTimeout")
	}
}

func TestApiClientDataConversion(t *testing.T) {
	// Create a mock client with the real API structure
	client := api.NewClient("http://localhost:99999", 100*time.Millisecond) // Will fail, but that's OK
	adapter := client

	// Test that the adapter interface is properly implemented
	var _ storage.APIClient = adapter

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// This will fail due to connection, but we're testing the interface
	_, err := adapter.FetchAllData(ctx)
	if err == nil {
		t.Error("Expected connection error")
	}

	// Test that error contains connection-related text
	if !strings.Contains(strings.ToLower(err.Error()), "connect") &&
		!strings.Contains(strings.ToLower(err.Error()), "dial") &&
		!strings.Contains(strings.ToLower(err.Error()), "port") {
		t.Logf("Got error (which is expected): %v", err)
	}
}

func TestColorConstantsAreValid(t *testing.T) {
	colors := []struct {
		name  string
		value string
	}{
		{"reset", colorReset},
		{"red", colorRed},
		{"green", colorGreen},
		{"yellow", colorYellow},
		{"cyan", colorCyan},
	}

	for _, color := range colors {
		if color.value == "" {
			t.Errorf("Color %s should not be empty", color.name)
		}

		// Check that it's a valid ANSI escape sequence
		if !strings.HasPrefix(color.value, "\033[") {
			t.Errorf("Color %s should start with ANSI escape sequence", color.name)
		}

		if !strings.HasSuffix(color.value, "m") {
			t.Errorf("Color %s should end with 'm'", color.name)
		}
	}
}

func TestWaitForDataLoadVariousScenarios(t *testing.T) {
	tests := []struct {
		name           string
		setupStore     func() *storage.Store
		timeout        time.Duration
		expectError    bool
		errorSubstring string
	}{
		{
			name: "store_with_data",
			setupStore: func() *storage.Store {
				store := storage.NewStore()
				// Load some test data
				store.LoadData(models.APIResponse{
					Artists: []models.Artist{{ID: 1, Name: "Test"}},
				})
				return store
			},
			timeout:     1 * time.Second,
			expectError: false,
		},
		{
			name: "empty_store_short_timeout",
			setupStore: func() *storage.Store {
				return storage.NewStore()
			},
			timeout:        100 * time.Millisecond,
			expectError:    true,
			errorSubstring: "timeout",
		},
		{
			name: "empty_store_cancelled_context",
			setupStore: func() *storage.Store {
				return storage.NewStore()
			},
			timeout:        0, // Will be cancelled immediately
			expectError:    true,
			errorSubstring: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			service := service.NewService(store)

			var ctx context.Context
			var cancel context.CancelFunc

			if tt.timeout == 0 {
				ctx, cancel = context.WithCancel(context.Background())
				cancel() // Cancel immediately
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), tt.timeout)
				defer cancel()
			}

			err := waitForDataLoad(service, ctx)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectError && err != nil && tt.errorSubstring != "" {
				if !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorSubstring, err)
				}
			}
		})
	}
}

func TestServerStructFieldTypes(t *testing.T) {
	// Test that Server struct has the expected field types
	var s Server

	// Test that fields can be set to their expected types
	s.store = storage.NewStore()
	s.apiClient = api.NewClient("http://test.com", 5*time.Second)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	// Verify non-nil assignments worked
	if s.store == nil {
		t.Error("Store assignment failed")
	}
	if s.apiClient == nil {
		t.Error("API client assignment failed")
	}
	if s.ctx == nil {
		t.Error("Context assignment failed")
	}
	if s.cancel == nil {
		t.Error("Cancel function assignment failed")
	}
}

func TestCreateRouter(t *testing.T) {
	// Setup test store
	store := storage.NewStore()
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Test Artist", CreationYear: 2000, Members: []string{"Member 1"}},
		},
	}
	store.LoadData(testData)

	// Create handlers
	apiClient := api.NewClient("https://groupietrackers.herokuapp.com", 10*time.Second)
	service := service.NewService(store)
	h := handlers.NewHandlers(store, service, apiClient)

	// Test router creation
	mux := createRouter(h)
	if mux == nil {
		t.Fatal("createRouter returned nil")
	}

	// Test a basic route
	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Health endpoint returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestMiddleware(t *testing.T) {
	// Test that middleware functions don't panic
	store := storage.NewStore()
	apiClient := api.NewClient("https://groupietrackers.herokuapp.com", 10*time.Second)
	service := service.NewService(store)
	h := handlers.NewHandlers(store, service, apiClient)

	// Create a simple handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Test with middleware
	wrapped := wrapWithMiddleware(testHandler, h)

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Middleware wrapped handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
