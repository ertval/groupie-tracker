package web

import (
	"bytes"
	"fmt"
	"groupie-tracker/internal/conf"
	"groupie-tracker/internal/data"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// --- Template Data Structures ---

// --- Template Rendering System ---

// render executes a template and sends the response.
func (app *App) render(w http.ResponseWriter, r *http.Request, name string, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}

	tmpl, ok := app.templates[name]
	if !ok {
		// Prevent infinite recursion if error template itself is missing
		if name == "error.tmpl" {
			log.Printf("FATAL: error.tmpl is missing")
			http.Error(w, "500 Internal Server Error - Error template not found", http.StatusInternalServerError)
			return
		}
		app.Error(w, r, http.StatusInternalServerError, fmt.Sprintf("Template %s not found", name))
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
		app.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Only send response after successful template execution
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

// loadTemplates compiles and caches all HTML templates.
func (app *App) loadTemplates() {
	app.templates = make(map[string]*template.Template)

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

		app.templates[name] = ts
	}
}

// getPort determines HTTP server port from environment or config.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return conf.DefaultPort
	}

	// Ensure port has colon prefix for http.Server
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}

// --- Form Data Processing ---

// parseIntPtr parses integer form field and returns pointer.
func parseIntPtr(r *http.Request, fieldName string) *int {
	if str := r.FormValue(fieldName); str != "" {
		if val, err := strconv.Atoi(str); err == nil {
			return &val
		}
	}
	return nil
}

// parseIntSlice parses multiple checkbox values into integer slice.
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

// parseStringSlice parses multiple form values into string slice.
func parseStringSlice(r *http.Request, fieldName string) []string {
	if values := r.Form[fieldName]; len(values) > 0 {
		return values
	}
	return nil
}

// parseArtistFilterParams extracts artist filter parameters from form data.
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

// parseLocationFilterParams extracts location filter parameters from form data.
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

// extractSearchTerm extracts search term from datalist suggestion format.
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

// --- Data Manipulation Utilities ---

// getRandomArtists shuffles the provided artists slice and returns up to maxCount random artists.
// This utility function encapsulates the randomization logic to keep handlers clean.
//
// Parameters:
//   - artists: slice of artists to shuffle
//   - maxCount: maximum number of artists to return after shuffling
//
// Returns a new slice containing up to maxCount randomly selected artists.
// The original slice is not modified.
func getRandomArtists(artists []data.Artist, maxCount int) []data.Artist {
	if len(artists) == 0 {
		return artists
	}

	// Create a copy to avoid modifying the original slice
	shuffled := make([]data.Artist, len(artists))
	copy(shuffled, artists)

	// Shuffle the copy
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Limit to maxCount
	if len(shuffled) > maxCount {
		shuffled = shuffled[:maxCount]
	}

	return shuffled
}
