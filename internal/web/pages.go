package web

import (
	"encoding/json"
	"net/http"
	"time"

	"groupie-tracker/internal/data"
)

// ============================================================================
// HOME PAGE
// ============================================================================

// Home handles the home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	// Validate path using centralized utility
	if !s.validateExactPath(w, r, "/") {
		return
	}

	artists := s.store.Artists()
	stats := s.store.Stats()
	suggestions := s.store.GenerateAllSearchSuggestions()

	// Get 8 random artists for homepage display
	artists = getRandomArtists(artists, 8)

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Suggestions    []data.SearchSuggestion
		Artists        []data.Artist
		TotalMembers   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		ExtraJS:        "",
		Suggestions:    suggestions,
		Artists:        artists,
		TotalMembers:   stats.TotalMembers,
		TotalLocations: stats.TotalLocations,
	}

	s.render(w, r, "home.tmpl", data)
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// Health provides a health check endpoint.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     s.store.Stats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// DEVELOPER TOOLS
// ============================================================================

// DevIndex renders a small developer page with quick links.
func (s *Server) DevIndex(w http.ResponseWriter, r *http.Request) {
	links := []struct{ Href, Text string }{
		{"/dev/panic", "Trigger Panic (/dev/panic)"},
		{"/dev/404", "Simulate 404 (/dev/404)"},
		{"/dev/500", "Simulate 500 (/dev/500)"},
		{"/dev/tmpl-error", "Simulate Template Error (/dev/tmpl-error)"},
		{"/health", "Health Check (/health)"},
	}

	suggestions := s.store.GenerateAllSearchSuggestions()

	data := struct {
		Title       string
		ExtraCSS    string
		ExtraJS     string
		Suggestions []data.SearchSuggestion
		Links       []struct{ Href, Text string }
	}{
		Title:       "Developer Tools",
		ExtraCSS:    "dev.css",
		ExtraJS:     "",
		Suggestions: suggestions,
		Links:       links,
	}

	s.render(w, r, "dev.tmpl", data)
}

// DevPanic is a development endpoint to test panic recovery.
func (s *Server) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Dev404 is a development endpoint to test 404 error template.
func (s *Server) Dev404(w http.ResponseWriter, r *http.Request) {
	// Simulate a realistic 404 by mutating a shallow copy of the request
	// so that template rendering sees a non-existent requested URL.
	// We keep the original request untouched and pass the modified copy
	// to the Home handler which will validate the path and trigger a 404.
	nr := new(http.Request)
	*nr = *r
	// Ensure method is GET and set a path that we know doesn't exist in the router
	nr.Method = http.MethodGet
	nr.URL.Path = "/this-page-does-not-exist"

	// Call Home with the modified request so the Error template is rendered
	// using the realistic requested URL stored in nr.URL.Path.
	s.Home(w, nr)
}

// Dev500 is a development endpoint to test 500 error template.
func (s *Server) Dev500(w http.ResponseWriter, r *http.Request) {
	s.Error(w, r, http.StatusInternalServerError, "This is a simulated 500 error.")
}

// Dev500Tmpl is a development endpoint to test template failure.
func (s *Server) Dev500Tmpl(w http.ResponseWriter, r *http.Request) {
	// To simulate a template error, we can try to render a template that doesn't exist.
	s.render(w, r, "nonexistent.tmpl", nil)
}
