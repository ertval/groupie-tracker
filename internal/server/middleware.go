package server

import (
	"log"
	"net/http"
	"time"
)

// server configuration is now provided by the internal/config package
// withMiddleware applies all middleware to a handler.
func withMiddleware(next http.Handler) http.Handler {
	return withLogging(withRecovery(withSecureHeaders(next)))
}

// withRecovery wraps a handler with panic recovery middleware.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging wraps a handler with request logging middleware.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// withSecureHeaders wraps a handler with security headers middleware.
func withSecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")
		// Content-Security-Policy is intentionally not set to allow flexibility with external resources
		// w.Header().Set("Content-Security-Policy",
		// "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:;
		// font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';")
		next.ServeHTTP(w, r)
	})
}

// onlyMethod wraps a handler to only allow specific HTTP methods.
// This eliminates duplication of method validation across handlers.
func onlyMethod(handler http.HandlerFunc, methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := false
		for _, method := range methods {
			if r.Method == method {
				allowed = true
				break
			}
		}

		if !allowed {
			// Set Allow header with all allowed methods
			allowHeader := ""
			for i, method := range methods {
				if i > 0 {
					allowHeader += ", "
				}
				allowHeader += method
			}
			w.Header().Set("Allow", allowHeader)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}
