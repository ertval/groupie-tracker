package http

import (
	"fmt"
	"net/http"
	"time"
)

// WithMiddleware wraps the handler with all middleware layers.
func WithMiddleware(handler http.Handler) http.Handler {
	// Apply middleware in reverse order (outer to inner)
	return withRecovery(
		withLogging(
			withSecurity(
				withRateLimit(handler))))
}

// withRecovery provides panic recovery middleware to prevent crashes.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging provides request logging middleware for monitoring.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		// Simple access log
		fmt.Printf("%s %s %v\n", r.Method, r.URL.Path, duration)
	})
}

// withSecurity provides basic security headers.
func withSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// withRateLimit provides simple rate limiting middleware.
// For now, this is a placeholder that could be enhanced with actual rate limiting.
func withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple rate limiting could be added here
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}
