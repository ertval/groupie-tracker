package web

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// helper to execute handler and return status code
func executeRequest(t *testing.T, handler http.Handler, r *http.Request) int {
	t.Helper()
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, r)
	return rr.Code
}

func TestRateLimiterSameIP(t *testing.T) {
	var store sync.Map
	limiter := withRateLimitStore(&store, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), 0, 2) // no refill, burst 2

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:12345"

	if code := executeRequest(t, limiter, req.Clone(req.Context())); code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", code)
	}
	if code := executeRequest(t, limiter, req.Clone(req.Context())); code != http.StatusOK {
		t.Fatalf("expected second request to succeed, got %d", code)
	}
	if code := executeRequest(t, limiter, req.Clone(req.Context())); code != http.StatusTooManyRequests {
		t.Fatalf("expected third request to be rate limited, got %d", code)
	}
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	var store sync.Map
	limiter := withRateLimitStore(&store, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), 0, 1) // burst 1 per IP

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.1.1.1:1111"

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "2.2.2.2:2222"

	if code := executeRequest(t, limiter, req1.Clone(req1.Context())); code != http.StatusOK {
		t.Fatalf("expected first IP request to succeed, got %d", code)
	}
	if code := executeRequest(t, limiter, req2.Clone(req2.Context())); code != http.StatusOK {
		t.Fatalf("expected second IP request to succeed, got %d", code)
	}
	if code := executeRequest(t, limiter, req1.Clone(req1.Context())); code != http.StatusTooManyRequests {
		t.Fatalf("expected second request from same IP to be limited, got %d", code)
	}
}
