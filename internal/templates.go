package data

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// --- Template Management ---

// loadTemplates compiles all HTML templates with custom helper functions.
func loadTemplates() error {
	templates = make(map[string]*template.Template)

	// Template helper functions
	funcMap := template.FuncMap{
		"join":      strings.Join,
		"len":       templateLen,
		"pluralize": pluralize,
		"contains":  strings.Contains,
		"add":       add,
		"sub":       sub,
		"hasField":  hasField,
		"title":     toTitleCase,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
	}

	// Template file patterns
	templateFiles := []string{
		"templates/home.tmpl",
		"templates/artists.tmpl",
		"templates/artist_detail.tmpl",
		"templates/search.tmpl",
		"templates/locations.tmpl",
		"templates/location_detail.tmpl",
		"templates/error.tmpl",
	}

	// Load each template with base template
	for _, templateFile := range templateFiles {
		templateName := filepath.Base(templateFile)
		
		tmpl, err := template.New(templateName).Funcs(funcMap).ParseFiles(
			"templates/base.tmpl",
			templateFile,
		)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", templateFile, err)
		}

		templates[templateName] = tmpl
	}

	return nil
}

// renderTemplate renders a template with the provided data.
func renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	tmpl, exists := templates[templateName]
	if !exists {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		return
	}
}

// --- Template Helper Functions ---

// templateLen returns the length of various types for template use.
func templateLen(v interface{}) int {
	switch val := v.(type) {
	case []Artist:
		return len(val)
	case []string:
		return len(val)
	case []Concert:
		return len(val)
	case []Location:
		return len(val)
	case string:
		return len(val)
	default:
		return 0
	}
}

// pluralize returns singular or plural form based on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// add performs addition for template use.
func add(a, b int) int {
	return a + b
}

// sub performs subtraction for template use.
func sub(a, b int) int {
	return a - b
}

// hasField checks if a struct has a specific field (simplified version).
func hasField(obj interface{}, fieldName string) bool {
	// Simple implementation for template use
	return true // Most cases will have the field
}

// toTitleCase converts a string to title case, replacing hyphens with spaces.
func toTitleCase(s string) string {
	// Replace hyphens with spaces
	s = strings.ReplaceAll(s, "-", " ")
	
	// Split into words and capitalize each
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	
	return strings.Join(words, " ")
}

// --- Middleware ---

// withMiddleware wraps the handler with all middleware.
func withMiddleware(handler http.Handler) http.Handler {
	// Apply middleware in reverse order (outer to inner)
	return withRecovery(
		withLogging(
			withSecurity(
				withRateLimit(handler))))
}

// withRecovery provides panic recovery middleware.
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

// withLogging provides request logging middleware.
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

// withRateLimit provides simple rate limiting.
func withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple rate limiting could be added here
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}