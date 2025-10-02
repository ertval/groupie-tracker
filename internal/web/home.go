package web

import (
	"net/http"

	"groupie-tracker/internal/data"
)

// Home handles the home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	// Validate path using centralized utility
	if !s.validateExactPath(w, r, "/") {
		return
	}

	artists := s.svc.Artists()
	stats := s.svc.Stats()
	suggestions := s.svc.GenerateAllSearchSuggestions()

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
