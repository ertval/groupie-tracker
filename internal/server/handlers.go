package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/data"
)

// Home handles the home page.
func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	if !a.validateRequestGETPath(w, r, "/") {
		return
	}

	artists := a.repo.GetArtists()
	stats := a.repo.GetStats()

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Artists        []data.Artist
		TotalMembers   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		ExtraJS:        "",
		Artists:        artists,
		TotalMembers:   stats["total_members"],
		TotalLocations: stats["total_locations"],
	}

	a.render(w, r, "home.tmpl", data)
}

// Artists handles the artists listing page.
func (a *App) Artists(w http.ResponseWriter, r *http.Request) {
	// Allow both GET and POST requests
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate path for both GET and POST
	if r.URL.Path != "/artists" {
		a.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	artists := a.repo.GetArtists()
	filterOptions := a.repo.GetArtistFilterOptions()
	var appliedFilters data.ArtistFilterParams
	totalArtists := len(artists)

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.Error(w, r, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		appliedFilters = a.parseFilterParams(r)
		artists = a.repo.FilterArtists(appliedFilters)
	}

	// Sort artists by concert count (descending) for main display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].ConcertCount > artists[j].ConcertCount
	})

	// Get top 10 artists by concert count for the top section
	topArtists := make([]data.Artist, 0, 10)
	if len(artists) > 0 {
		limit := 10
		if len(artists) < 10 {
			limit = len(artists)
		}
		topArtists = artists[:limit]
	}

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Artists        []data.Artist
		TopArtists     []data.Artist
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsFiltered     bool
		TotalArtists   int
	}{
		Title:          "Artists",
		ExtraCSS:       "artists.css",
		ExtraJS:        "",
		Artists:        artists,
		TopArtists:     topArtists,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsFiltered:     r.Method == http.MethodPost,
		TotalArtists:   totalArtists,
	}

	a.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (a *App) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		a.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	// Try slug first, then ID
	artist, found := a.repo.GetArtistBySlug(path)
	if !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = a.repo.GetArtistByID(id)
		}
		if !found {
			a.Error(w, r, http.StatusNotFound, "Artist not found")
			return
		}
	}

	// Get navigation artists
	var prevArtist, nextArtist *data.Artist
	if artist.PrevArtistID != 0 {
		if p, ok := a.repo.GetArtistByID(artist.PrevArtistID); ok {
			prevArtist = &p
		}
	}
	if artist.NextArtistID != 0 {
		if n, ok := a.repo.GetArtistByID(artist.NextArtistID); ok {
			nextArtist = &n
		}
	}

	data := struct {
		Title      string
		ExtraCSS   string
		ExtraJS    string
		Artist     data.Artist
		PrevArtist *data.Artist
		NextArtist *data.Artist
	}{
		Title:      artist.Name,
		ExtraCSS:   "artist_detail.css",
		ExtraJS:    "",
		Artist:     artist,
		PrevArtist: prevArtist,
		NextArtist: nextArtist,
	}

	a.render(w, r, "artist_detail.tmpl", data)
}

// Locations handles the locations listing page.
func (a *App) Locations(w http.ResponseWriter, r *http.Request) {
	// Allow both GET and POST requests
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate path for both GET and POST
	if r.URL.Path != "/locations" {
		a.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	locations := a.repo.GetLocations()
	filterOptions := a.repo.GetLocationFilterOptions()
	var appliedFilters data.LocationFilterParams
	totalLocations := len(locations)
	stats := a.repo.GetStats()

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.Error(w, r, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		appliedFilters = a.parseLocationFilterParams(r)
		locations = a.repo.FilterLocations(appliedFilters)
	}

	data := struct {
		Title                 string
		ExtraCSS              string
		ExtraJS               string
		Locations             []data.Location
		LocationFilterOptions data.LocationFilterOptions
		AppliedFilters        data.LocationFilterParams
		IsFiltered            bool
		TotalLocations        int
		TotalCountries        int
		TotalConcerts         int
	}{
		Title:                 "Locations",
		ExtraCSS:              "locations.css",
		ExtraJS:               "",
		Locations:             locations,
		LocationFilterOptions: filterOptions,
		AppliedFilters:        appliedFilters,
		IsFiltered:            r.Method == http.MethodPost,
		TotalLocations:        totalLocations,
		TotalCountries:        stats["total_countries"],
		TotalConcerts:         stats["total_concerts"],
	}

	a.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (a *App) LocationDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		a.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	location, found := a.repo.GetLocationBySlug(slug)
	if !found {
		a.Error(w, r, http.StatusNotFound, "Location not found")
		return
	}

	data := struct {
		Title    string
		ExtraCSS string
		ExtraJS  string
		Location data.Location
		Artists  []data.ArtistAtLocation
	}{
		Title:    fmt.Sprintf("%s - Location", location.Name),
		ExtraCSS: "location_detail.css",
		ExtraJS:  "",
		Location: location,
		Artists:  location.Artists,
	}

	a.render(w, r, "location_detail.tmpl", data)
}

// DevIndex renders a small developer page with quick links.
func (a *App) DevIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	links := []struct{ Href, Text string }{
		{"/dev/panic", "Trigger Panic (/dev/panic)"},
		{"/dev/404", "Simulate 404 (/dev/404)"},
		{"/dev/500", "Simulate 500 (/dev/500)"},
		{"/dev/tmpl-error", "Simulate Template Error (/dev/tmpl-error)"},
		{"/health", "Health Check (/health)"},
	}

	data := struct {
		Title    string
		ExtraCSS string
		ExtraJS  string
		Links    []struct{ Href, Text string }
	}{
		Title:    "Developer Tools",
		ExtraCSS: "dev.css",
		ExtraJS:  "",
		Links:    links,
	}

	a.render(w, r, "dev.tmpl", data)
}

// Error handles all errors (4xx and 5xx) in a centralized way.
func (a *App) Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
		ErrorCode:    status,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	a.render(w, r, "error.tmpl", data, status)
}

// Health provides a health check endpoint.
func (a *App) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     a.repo.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DevPanic is a development endpoint to test panic recovery.
func (a *App) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Dev404 is a development endpoint to test 404 error template.
func (a *App) Dev404(w http.ResponseWriter, r *http.Request) {
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
	a.Home(w, nr)
}

// Dev500 is a development endpoint to test 500 error template.
func (a *App) Dev500(w http.ResponseWriter, r *http.Request) {
	a.Error(w, r, http.StatusInternalServerError, "This is a simulated 500 error.")
}

// Dev500Tmpl is a development endpoint to test template failure.
func (a *App) Dev500Tmpl(w http.ResponseWriter, r *http.Request) {
	// To simulate a template error, we can try to render a template that doesn't exist.
	a.render(w, r, "nonexistent.tmpl", nil)
}

func (a *App) StaticFiles(w http.ResponseWriter, r *http.Request) {
	const staticDir = "static"

	// Only allow GET and HEAD methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		a.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Handle favicon.ico requests
	if r.URL.Path == "/favicon.ico" {
		target := filepath.Join(staticDir, "favicon.ico")
		if fi, err := os.Stat(target); err != nil || fi.IsDir() {
			a.Error(w, r, http.StatusNotFound, "Favicon not found")
			return
		}
		http.ServeFile(w, r, target)
		return
	}

	// Only allow /static/ prefix
	if !strings.HasPrefix(r.URL.Path, "/static/") {
		a.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Extract relative path and prevent directory traversal
	rel := strings.TrimPrefix(r.URL.Path, "/static/")
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		a.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Build target path and verify it's a regular file
	target := filepath.Join(staticDir, rel)
	if fi, err := os.Stat(target); err != nil || fi.IsDir() {
		a.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Serve the file (Go's http.ServeFile handles content-type automatically)
	http.ServeFile(w, r, target)
}
