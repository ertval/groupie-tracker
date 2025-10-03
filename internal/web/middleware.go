package web

import (
	"log"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"groupie-tracker/internal/conf"
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
	return withLogging(
		withRecovery(
			withSecureHeaders(
				withDefaultRateLimit(next),
			),
		),
	)
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

// ============================================================================
// RATE LIMITER
// ============================================================================
// Simple per-client token-bucket rate limiter. Keyed by client IP (X-Forwarded-For or RemoteAddr).
// Not distributed: suitable for a single-process deployment. Uses per-client mutex and a sync.Map
// to avoid global locks.

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time
	rate     float64 // tokens per second
	capacity float64
}

// allow attempts to consume a token; returns true if allowed.
func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	// Refill tokens
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.rate
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.last = now
	}
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	return false
}

var globalLimiterStore sync.Map // map[string]*tokenBucket

// getClientIP returns the client's IP using X-Forwarded-For or RemoteAddr.
func getClientIP(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-For"); h != "" {
		// X-Forwarded-For may be a comma-separated list; take the first
		parts := strings.Split(h, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// withRateLimit wraps handlers with a per-client token-bucket limiter.
// If limit exceeded, returns 429 Too Many Requests with Retry-After header.
func withRateLimit(next http.Handler, rate float64, burst float64) http.Handler {
	return withRateLimitStore(&globalLimiterStore, next, rate, burst)
}

func withDefaultRateLimit(next http.Handler) http.Handler {
	return withRateLimitStore(&globalLimiterStore, next, float64(conf.RateLimitRequestsPerSecond), float64(conf.RateLimitBurst))
}

func withRateLimitStore(store *sync.Map, next http.Handler, rate float64, burst float64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := getClientIP(r)
		v, _ := store.LoadOrStore(key, &tokenBucket{
			tokens:   burst,
			last:     time.Now(),
			rate:     rate,
			capacity: burst,
		})
		b := v.(*tokenBucket)
		if !b.allow() {
			w.Header().Set("Retry-After", "1")
			log.Printf("rate limit exceeded: ip=%s method=%s path=%s", key, r.Method, r.URL.Path)
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// restrictMethod validates that the incoming request uses one of the allowed HTTP methods.
// Returns 405 Method Not Allowed with proper Allow header if method is not permitted.
// This is a method on App to allow access to App.Error for consistent error responses.
func (a *App) restrictMethod(handler http.HandlerFunc, allowedMethods ...string) http.HandlerFunc {
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
			a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		handler(w, r) // Method is allowed, proceed with handler execution
	}
}

// create a function post that allows only post requests
func (a *App) post(handler http.HandlerFunc) http.HandlerFunc {
	return a.restrictMethod(handler, http.MethodPost)
}

// create a function get that allows only get requests
func (a *App) get(handler http.HandlerFunc) http.HandlerFunc {
	return a.restrictMethod(handler, http.MethodGet)
}

// create a function getAndPost that allows only get and post requests
func (a *App) getPost(handler http.HandlerFunc) http.HandlerFunc {
	return a.restrictMethod(handler, http.MethodGet, http.MethodPost)
}

// create a function any that allows any request method
func (a *App) any(handler http.HandlerFunc) http.HandlerFunc {
	return handler // No method restriction, allow any HTTP method
}

// create a function getAndHead that allows only get and head requests
func (a *App) getHead(handler http.HandlerFunc) http.HandlerFunc {
	return a.restrictMethod(handler, http.MethodGet, http.MethodHead)
}
