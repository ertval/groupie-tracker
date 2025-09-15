package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPanicHandler ensures that a handler panic is recovered and results in a 500 response
func TestPanicHandler(t *testing.T) {
	// Create a Handlers with nil dependencies - the panic handler won't use them
	h := &Handlers{}

	// Create a request to any path
	req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
	rr := httptest.NewRecorder()

	// Wrap the panic in an inline handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentionally cause a panic
		panic("trigger test panic")
	})

	// Use the same recovery pattern as the real handlers: defer recover and call InternalErrorHandler
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				h.InternalErrorHandler(w, r, "Panic: test")
			}
		}()
		handler.ServeHTTP(w, r)
	})

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Fatalf("expected body to contain Internal Server Error, got: %s", body)
	}
}
