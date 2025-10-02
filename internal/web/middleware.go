package web

import (
	"log"
	"net/http"
	"slices"
	"time"
)

// ============================================================================
// MIDDLEWARE CHAIN
// ============================================================================
//
// Middleware execution order (from outermost to innermost):
//   1. withLogging       - Logs request method, path, and duration
//   2. withRecovery      - Catches panics and converts to 500 errors
//   3. withSecureHeaders - Sets security headers on all responses
//   4. handler           - Actual request handler
//
// This ordering ensures:
//   - All requests are logged, even if they panic
//   - Panics are caught and don't crash the server
//   - Security headers are always set
//
// To add new middleware, insert it into the withMiddleware function chain.
// ============================================================================

// withMiddleware assembles the complete middleware chain for all HTTP requests.
// Chain order (innermost to outermost): secureHeaders → recovery → logging
// This order ensures security headers are set first, panics are caught, and all requests are logged.
func withMiddleware(next http.Handler) http.Handler {
	return withLogging(withRecovery(withSecureHeaders(next)))
}

// withRecovery catches panics in handlers and converts them to 500 errors instead of crashing the server.
// Logs panic details for debugging while returning a generic error to the client for security.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil { // Catch any panic from downstream handlers
				log.Printf("Panic recovered: %v", err) // Log for debugging (includes stack trace in production logs)
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error")) // Generic message to avoid leaking internal details
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging logs each HTTP request with method, path, and response time for monitoring and debugging.
// Time measurement starts before handler execution and completes after response is sent.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()                                             // Record request start time
		next.ServeHTTP(w, r)                                            // Execute the actual handler
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start)) // Log after response: "GET /artists 15.2ms"
	})
}

// withSecureHeaders injects standard security headers into every HTTP response to mitigate common web vulnerabilities.
// Headers protect against: content sniffing, clickjacking, XSS, and referrer leakage.
// CSP is intentionally omitted to allow flexibility with external resources (images, fonts, CDNs).
func withSecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin") // Send only origin in cross-origin referrers (privacy)
		w.Header().Set("X-Content-Type-Options", "nosniff")           // Prevent MIME type sniffing (security)
		w.Header().Set("X-Frame-Options", "deny")                     // Prevent clickjacking by blocking iframe embedding
		w.Header().Set("X-XSS-Protection", "0")                       // Disable legacy XSS filter (modern CSP is better, but not set here)
		// Content-Security-Policy intentionally not set - allows external images/fonts/APIs without restriction
		// w.Header().Set("Content-Security-Policy",
		// "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:;
		// font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';")
		next.ServeHTTP(w, r)
	})
}

// restrictMethod validates that the incoming request uses one of the allowed HTTP methods.
// Returns 405 Method Not Allowed with proper Allow header if method is not permitted.
// This is a method on App to allow access to App.Error for consistent error responses.
func (app *App) restrictMethod(handler http.HandlerFunc, allowedMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := slices.Contains(allowedMethods, r.Method) // Check if request method is in allowed list

		if !allowed { // Request method not allowed for this endpoint
			// Build Allow header with comma-separated list of valid methods (required by HTTP spec)
			allowHeader := ""
			for i, method := range allowedMethods {
				if i > 0 {
					allowHeader += ", "
				}
				allowHeader += method
			}
			w.Header().Set("Allow", allowHeader) // Set Allow header for 405 response
			app.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		handler(w, r) // Method is allowed, proceed with handler execution
	}
}
