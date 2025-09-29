package server

import (
	"bytes"
	"fmt"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// --- Template Data Structures ---

// BaseTemplateData contains common fields used in all page templates.
// This eliminates duplication of Title, ExtraCSS, ExtraJS across all handlers.
type BaseTemplateData struct {
	Title       string                    // Page title for HTML head and display
	ExtraCSS    string                    // Additional CSS file to include
	ExtraJS     string                    // Additional JavaScript file to include  
	Suggestions []data.SearchSuggestion   // Search suggestions for autocomplete
}

// NewBaseTemplateData creates a new BaseTemplateData with search suggestions.
// This provides a consistent way to initialize template data across handlers.
func NewBaseTemplateData(title, cssFile string) BaseTemplateData {
	return BaseTemplateData{
		Title:       title,
		ExtraCSS:    cssFile,
		ExtraJS:     "",
		Suggestions: repo.GenerateAllSearchSuggestions(),
	}
}

// --- HTTP Request Validation ---

// validateRequestGETPath validates that the incoming request uses GET method and matches expected path.
// This helper ensures proper HTTP method usage and prevents handlers from processing invalid routes.
// Responds with appropriate error status (405 or 404) if validation fails.
//
// Returns true if request is valid, false if error response was sent to client.
func validateRequestGETPath(w http.ResponseWriter, r *http.Request, expectedPath string) bool {
	if r.Method != http.MethodGet {
		Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}

	if r.URL.Path != expectedPath {
		Error(w, r, http.StatusNotFound, "Page not found")
		return false
	}

	return true
}

// --- Template Rendering System ---

// render executes an HTML template with the provided data and sends response to client.
//
// This is the core template rendering function that handles:
//   - Template lookup and validation
//   - Template execution with error recovery
//   - HTTP status code management
//   - Graceful fallback to error pages on template failures
//
// The function uses a two-phase rendering approach: templates are first executed
// into a buffer to catch errors before sending any response to the client.
//
// Parameters:
//   - name: template filename (e.g., "home.tmpl")
//   - data: template data (can be any type)
//   - status: optional HTTP status code (defaults to 200)
func render(w http.ResponseWriter, r *http.Request, name string, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}

	tmpl, ok := templates[name]
	if !ok {
		// Prevent infinite recursion if error template itself is missing
		if name == "error.tmpl" {
			log.Printf("FATAL: error.tmpl is missing")
			http.Error(w, "500 Internal Server Error - Error template not found", http.StatusInternalServerError)
			return
		}
		Error(w, r, http.StatusInternalServerError, fmt.Sprintf("Template %s not found", name))
		return
	}

	// Use buffer to catch template execution errors before sending response
	buf := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(buf, "base", data); err != nil {
		// Handle error template execution failure gracefully
		if name == "error.tmpl" {
			log.Printf("Error executing error template: %v", err)
			http.Error(w, "500 Internal Server Error - Failed to execute error template", http.StatusInternalServerError)
			return
		}
		Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Only send response after successful template execution
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

// loadTemplates discovers, compiles, and caches all HTML templates from the templates directory.
//
// This function performs template initialization at server startup:
//   - Registers custom template functions (add, sub, join, upper, title)
//   - Discovers all .tmpl files in the templates directory
//   - Compiles each template with the base.tmpl wrapper for template inheritance
//   - Stores compiled templates in the global templates map
//
// The template system uses inheritance where each page template defines "title" and "body" blocks
// that are injected into the base.tmpl wrapper. Custom functions provide common operations
// like arithmetic and string manipulation directly in templates.
//
// Panics on any error since templates are required for basic server functionality.
func loadTemplates() {
	templates = make(map[string]*template.Template)

	// Custom template functions for common operations
	funcMap := template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"join":  func(items []string, sep string) string { return strings.Join(items, sep) },
		"upper": func(s string) string { return strings.ToUpper(s) },
		"title": func(s string) string {
			words := strings.Fields(strings.ReplaceAll(s, "-", " "))
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
		"contains": func(slice interface{}, item interface{}) bool {
			switch s := slice.(type) {
			case []int:
				if i, ok := item.(int); ok {
					for _, v := range s {
						if v == i {
							return true
						}
					}
				}
			case []string:
				if str, ok := item.(string); ok {
					for _, v := range s {
						if v == str {
							return true
						}
					}
				}
			}
			return false
		},
		"hasField": func(obj interface{}, fieldName string) bool {
			v := reflect.ValueOf(obj)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			if v.Kind() != reflect.Struct {
				return false
			}
			return v.FieldByName(fieldName).IsValid()
		},
	}

	const templateDir = "templates"
	baseTmplPath := filepath.Join(templateDir, "base.tmpl")

	if _, err := os.Stat(baseTmplPath); err != nil {
		log.Fatalf("Failed to find base template at %s: %v", baseTmplPath, err)
	}

	// Discover all template files
	pages, err := filepath.Glob(filepath.Join(templateDir, "*.tmpl"))
	if err != nil {
		log.Fatalf("Failed to glob templates: %v", err)
	}

	// Compile each template with base template for inheritance
	for _, page := range pages {
		name := filepath.Base(page)
		if name == "base.tmpl" {
			continue // Skip base template as it's included in each page
		}

		ts, err := template.New(name).Funcs(funcMap).ParseFiles(baseTmplPath, page)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}

		templates[name] = ts
	}
}

// getPort determines the HTTP server port from environment or configuration.
//
// Checks the PORT environment variable first (for cloud deployments like Heroku),
// then falls back to the configured default port. Ensures the port has a leading
// colon prefix required by http.Server.
//
// Returns a port string like ":8080" ready for use with http.Server.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return config.DefaultPort
	}

	// Ensure port has colon prefix for http.Server
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}

// --- Form Data Processing ---

// Generic form parsing utilities to eliminate duplication across form handlers

// parseIntPtr parses an integer form field and returns a pointer to the value.
// Returns nil for empty or invalid values, which allows distinguishing between 0 and unset.
func parseIntPtr(r *http.Request, fieldName string) *int {
	if str := r.FormValue(fieldName); str != "" {
		if val, err := strconv.Atoi(str); err == nil {
			return &val
		}
	}
	return nil
}

// parseIntSlice parses multiple checkbox values into an integer slice.
// Used for form fields like member counts where multiple selections are allowed.
func parseIntSlice(r *http.Request, fieldName string) []int {
	var results []int
	if values := r.Form[fieldName]; len(values) > 0 {
		for _, valueStr := range values {
			if value, err := strconv.Atoi(valueStr); err == nil {
				results = append(results, value)
			}
		}
	}
	return results
}

// parseStringSlice parses multiple form values into a string slice.
// Used for form fields like countries where multiple selections are allowed.
func parseStringSlice(r *http.Request, fieldName string) []string {
	if values := r.Form[fieldName]; len(values) > 0 {
		return values
	}
	return nil
}

// parseArtistFilterParams extracts and validates artist filter parameters from HTML form submission.
//
// Converts form values into structured filter parameters with proper type handling:
//   - Year ranges: converted to integers with nil for empty values
//   - Member counts: parsed as integer slice from checkbox selections
//   - Countries: used directly as string slice from checkbox selections
//
// This function handles the common pattern of form data being submitted as strings
// that need conversion to appropriate Go types for the business logic layer.
//
// Returns a populated ArtistFilterParams struct ready for use with repository filtering.
func parseArtistFilterParams(r *http.Request) data.ArtistFilterParams {
	var params data.ArtistFilterParams

	// Use generic utilities to eliminate duplication
	params.CreationYearFrom = parseIntPtr(r, "creationYearFrom")
	params.CreationYearTo = parseIntPtr(r, "creationYearTo")
	params.FirstAlbumYearFrom = parseIntPtr(r, "firstAlbumYearFrom")
	params.FirstAlbumYearTo = parseIntPtr(r, "firstAlbumYearTo")
	params.MemberCounts = parseIntSlice(r, "memberCounts")
	params.Countries = parseStringSlice(r, "countries")

	return params
}

// parseLocationFilterParams extracts and validates location filter parameters from HTML form submission.
//
// Similar to parseFilterParams but for location-specific filtering criteria:
//   - Concert count range: number of concerts held at the location
//   - Artist count range: number of unique artists who performed there
//   - Concert year range: temporal filtering of concert dates
//   - Countries: geographical filtering by country names
//
// Returns a populated LocationFilterParams struct for location-based queries.
func parseLocationFilterParams(r *http.Request) data.LocationFilterParams {
	var params data.LocationFilterParams

	// Use generic utilities to eliminate duplication
	params.ConcertCountFrom = parseIntPtr(r, "concertCountFrom")
	params.ConcertCountTo = parseIntPtr(r, "concertCountTo")
	params.ArtistCountFrom = parseIntPtr(r, "artistCountFrom")
	params.ArtistCountTo = parseIntPtr(r, "artistCountTo")
	params.ConcertYearFrom = parseIntPtr(r, "concertYearFrom")
	params.ConcertYearTo = parseIntPtr(r, "concertYearTo")
	params.Countries = parseStringSlice(r, "countries")

	return params
}

// extractSearchTerm extracts the actual search term from datalist suggestion format.
// Converts "Artist Name - artist" or "Member Name - member" back to "Artist Name" or "Member Name"
// for proper search processing. Returns the original string if no pattern is found.
func extractSearchTerm(input string) string {
	if input == "" {
		return input
	}

	// Check if input matches datalist format "term - type"
	if lastDash := strings.LastIndex(input, " - "); lastDash != -1 {
		term := strings.TrimSpace(input[:lastDash])
		if term != "" {
			return term
		}
	}

	return input
}

// addSuggestionsToData adds search suggestions to template data if the data struct has a Suggestions field.
// This helper allows pages to optionally include search suggestions for the global navbar without
// requiring all handlers to be modified or causing template errors on pages that don't need suggestions.
func addSuggestionsToData(data interface{}) interface{} {
	// Use reflection to check if the data struct has a Suggestions field
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return data
	}

	suggestionsField := v.FieldByName("Suggestions")
	if !suggestionsField.IsValid() || !suggestionsField.CanSet() {
		// No Suggestions field or not settable, return original data
		return data
	}

	// Only generate suggestions if the field exists and is empty
	if suggestionsField.IsNil() || suggestionsField.Len() == 0 {
		suggestions := repo.GenerateAllSearchSuggestions()
		suggestionsValue := reflect.ValueOf(suggestions)
		if suggestionsValue.Type().AssignableTo(suggestionsField.Type()) {
			suggestionsField.Set(suggestionsValue)
		}
	}

	return data
}
