package data

import (
	"context"
	"encoding/json"
	"fmt"
	"groupie-tracker/internal/config"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// --- Server Infrastructure ---

// Server encapsulates all server dependencies with direct repository access.
type Server struct {
	repo               *Repository                // Direct repository access
	templates          map[string]*template.Template // Pre-compiled templates
	suggestions        []SearchSuggestion         // Cached search suggestions
	artistFilterOpts   ArtistFilterOptions        // Cached artist filter options
	locationFilterOpts LocationFilterOptions      // Cached location filter options
	searchCache        map[string][]*Artist       // Query cache for performance
	cacheSize          int                        // Maximum cached queries
	httpServer         *http.Server               // HTTP server instance
	Handler            http.Handler               // Exported handler for tests
}

// NewServer creates and fully initializes a Server.
func NewServer() (*Server, error) {
	start := time.Now()

	server := &Server{}

	// Initialize repository
	server.repo = NewRepository()

	// Load all data from external API
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	err := server.repo.LoadData(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Initialize caches
	server.initializeCaches()

	// Compile templates
	server.loadTemplates()

	// Log startup summary
	stats := server.repo.GetStats()
	if !server.repo.IsCacheEnabled() {
		log.Printf("Data loaded successfully - %d artists (Image caching is disabled, serving from API)", stats["artists"])
	} else {
		log.Printf("Data loaded successfully with cache - %d artists", stats["artists"])
	}

	// Setup routing and middleware
	serveMux := withMiddleware(server.createServeMux())
	server.Handler = serveMux
	port := getPort()

	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:         port,
		Handler:      serveMux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	log.Printf("🚀 Server Initialized in %v and Ready to Open - http://localhost%s", time.Since(start), port)

	return server, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// --- Route Configuration ---

// createServeMux initializes and configures the HTTP router.
func (s *Server) createServeMux() *http.ServeMux {
	router := http.NewServeMux()

	// Static assets
	router.HandleFunc("/static/", s.restrictMethod(s.StaticFiles, "GET", "HEAD"))
	router.HandleFunc("/favicon.ico", s.restrictMethod(s.StaticFiles, "GET", "HEAD"))

	// Health check
	router.HandleFunc("/health", s.restrictMethod(s.Health, "GET"))

	// API endpoints
	router.HandleFunc("/api/suggestions", s.restrictMethod(s.SuggestionsAPI, "GET"))

	// Search endpoints
	router.HandleFunc("/search", s.restrictMethod(s.Search, "GET", "POST"))

	// Development tools
	router.HandleFunc("/dev", s.restrictMethod(s.DevIndex, "GET"))
	router.HandleFunc("/dev/panic", s.DevPanic)
	router.HandleFunc("/dev/404", s.Dev404)
	router.HandleFunc("/dev/500", s.Dev500)
	router.HandleFunc("/dev/tmpl-error", s.Dev500Tmpl)

	// Main application pages
	router.HandleFunc("/artists", s.restrictMethod(s.Artists, "GET", "POST"))
	router.HandleFunc("/artists/", s.restrictMethod(s.ArtistDetail, "GET"))
	router.HandleFunc("/locations", s.restrictMethod(s.Locations, "GET", "POST"))
	router.HandleFunc("/locations/", s.restrictMethod(s.LocationDetail, "GET"))

	// Home page
	router.HandleFunc("/", s.restrictMethod(s.Home, "GET"))

	return router
}

// --- Middleware ---

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
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

// --- HTTP Handlers ---

// Home handles the home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	if !s.validateExactPath(w, r, "/") {
		return
	}

	artists := s.repo.GetArtists()
	stats := s.repo.GetStats()

	// Get 8 random artists for homepage display
	randomArtists := getRandomArtists(artists, 8)

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Suggestions    []SearchSuggestion
		Artists        []Artist
		TotalMembers   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		ExtraJS:        "",
		Suggestions:    s.suggestions,
		Artists:        convertArtistPointersToValues(randomArtists),
		TotalMembers:   stats["total_members"],
		TotalLocations: stats["total_locations"],
	}

	s.render(w, r, "home.tmpl", data)
}

// Artists handles the artists listing page with filtering.
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
	if !s.validateExactPath(w, r, "/artists") {
		return
	}

	var artists []*Artist
	var filterParams ArtistFilterParams

	if r.Method == http.MethodPost {
		// Parse filter parameters from form
		filterParams = s.parseArtistFilters(r)
		artists = s.repo.FilterArtists(filterParams)
	} else {
		// GET request - show all artists
		artists = s.repo.GetArtists()
	}

	data := struct {
		Title          string
		ExtraCSS       string
		Artists        []Artist
		FilterOptions  ArtistFilterOptions
		FilterParams   ArtistFilterParams
		ResultCount    int
		TotalCount     int
		IsFiltered     bool
	}{
		Title:          "Artists",
		ExtraCSS:       "artists.css",
		Artists:        convertArtistPointersToValues(artists),
		FilterOptions:  s.artistFilterOpts,
		FilterParams:   filterParams,
		ResultCount:    len(artists),
		TotalCount:     len(s.repo.GetArtists()),
		IsFiltered:     r.Method == http.MethodPost,
	}

	s.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist detail pages.
func (s *Server) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/artists/")
	if slug == "" {
		s.renderError(w, r, http.StatusNotFound, "Artist not found", "Invalid artist URL")
		return
	}

	artist, found := s.repo.GetArtistBySlug(slug)
	if !found {
		s.renderError(w, r, http.StatusNotFound, "Artist not found", fmt.Sprintf("No artist found with slug: %s", slug))
		return
	}

	// Get adjacent artists for navigation
	prev, next := s.repo.GetAdjacentArtists(artist.ID)

	data := struct {
		Title    string
		ExtraCSS string
		Artist   Artist
		Previous *Artist
		Next     *Artist
	}{
		Title:    artist.Name,
		ExtraCSS: "artist_detail.css",
		Artist:   *artist,
		Previous: prev,
		Next:     next,
	}

	s.render(w, r, "artist_detail.tmpl", data)
}

// Locations handles the locations listing page with filtering.
func (s *Server) Locations(w http.ResponseWriter, r *http.Request) {
	if !s.validateExactPath(w, r, "/locations") {
		return
	}

	var locations []*Location
	var filterParams LocationFilterParams

	if r.Method == http.MethodPost {
		// Parse filter parameters from form
		filterParams = s.parseLocationFilters(r)
		locations = s.repo.FilterLocations(filterParams)
	} else {
		// GET request - show all locations
		locations = s.repo.GetLocations()
	}

	data := struct {
		Title          string
		ExtraCSS       string
		Locations      []Location
		FilterOptions  LocationFilterOptions
		FilterParams   LocationFilterParams
		ResultCount    int
		TotalCount     int
		IsFiltered     bool
	}{
		Title:          "Locations",
		ExtraCSS:       "locations.css",
		Locations:      convertLocationPointersToValues(locations),
		FilterOptions:  s.locationFilterOpts,
		FilterParams:   filterParams,
		ResultCount:    len(locations),
		TotalCount:     len(s.repo.GetLocations()),
		IsFiltered:     r.Method == http.MethodPost,
	}

	s.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location detail pages.
func (s *Server) LocationDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		s.renderError(w, r, http.StatusNotFound, "Location not found", "Invalid location URL")
		return
	}

	location, found := s.repo.GetLocationBySlug(slug)
	if !found {
		s.renderError(w, r, http.StatusNotFound, "Location not found", fmt.Sprintf("No location found with slug: %s", slug))
		return
	}

	data := struct {
		Title    string
		ExtraCSS string
		Location Location
	}{
		Title:    location.Name,
		ExtraCSS: "location_detail.css",
		Location: *location,
	}

	s.render(w, r, "location_detail.tmpl", data)
}

// Search handles search functionality.
func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	if !s.validateExactPath(w, r, "/search") {
		return
	}

	var results []*Artist
	var query string
	var filterParams ArtistFilterParams

	if r.Method == http.MethodPost {
		query = strings.TrimSpace(r.FormValue("query"))
		filterParams = s.parseArtistFilters(r)

		// Perform search with filters
		searchParams := SearchParams{
			Query:   query,
			Filters: filterParams,
		}
		searchResult := s.repo.SearchArtists(searchParams)
		results = searchResult.Artists
	}

	data := struct {
		Title         string
		ExtraCSS      string
		Query         string
		Results       []Artist
		FilterOptions ArtistFilterOptions
		FilterParams  ArtistFilterParams
		ResultCount   int
		HasSearched   bool
	}{
		Title:         "Search",
		ExtraCSS:      "search.css",
		Query:         query,
		Results:       convertArtistPointersToValues(results),
		FilterOptions: s.artistFilterOpts,
		FilterParams:  filterParams,
		ResultCount:   len(results),
		HasSearched:   r.Method == http.MethodPost,
	}

	s.render(w, r, "search.tmpl", data)
}

// SuggestionsAPI handles the search suggestions API endpoint.
func (s *Server) SuggestionsAPI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	maxResults := 10

	if maxResultsStr := r.URL.Query().Get("max"); maxResultsStr != "" {
		if parsed, err := strconv.Atoi(maxResultsStr); err == nil && parsed > 0 {
			maxResults = parsed
		}
	}

	filteredSuggestions := FilterSuggestionsOptimized(s.suggestions, query, maxResults)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filteredSuggestions); err != nil {
		log.Printf("Failed to encode suggestions: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Health handles health check endpoint.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	stats := s.repo.GetAppStats()
	
	healthData := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     stats,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(healthData); err != nil {
		log.Printf("Failed to encode health data: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// StaticFiles handles static file serving.
func (s *Server) StaticFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle favicon specifically
	if path == "/favicon.ico" {
		http.ServeFile(w, r, "static/favicon.ico")
		return
	}

	// Handle static files
	if strings.HasPrefix(path, "/static/") {
		// Security: prevent directory traversal
		if strings.Contains(path, "..") {
			s.renderError(w, r, http.StatusBadRequest, "Invalid path", "Path contains invalid characters")
			return
		}

		// Remove /static/ prefix and serve from static directory
		filePath := path[8:] // Remove "/static/"
		fullPath := filepath.Join("static", filePath)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			s.renderError(w, r, http.StatusNotFound, "File not found", fmt.Sprintf("Static file not found: %s", filePath))
			return
		}

		http.ServeFile(w, r, fullPath)
		return
	}

	s.renderError(w, r, http.StatusNotFound, "Not found", "The requested resource was not found")
}

// --- Development Tools ---

// DevIndex shows development tools page.
func (s *Server) DevIndex(w http.ResponseWriter, r *http.Request) {
	if !s.validateExactPath(w, r, "/dev") {
		return
	}

	data := struct {
		Title    string
		ExtraCSS string
		Tools    []DevTool
	}{
		Title:    "Development Tools",
		ExtraCSS: "dev.css",
		Tools: []DevTool{
			{Name: "Panic Test", URL: "/dev/panic", Description: "Test panic recovery middleware"},
			{Name: "404 Test", URL: "/dev/404", Description: "Test 404 error handling"},
			{Name: "500 Test", URL: "/dev/500", Description: "Test 500 error handling"},
			{Name: "Template Error", URL: "/dev/tmpl-error", Description: "Test template error handling"},
		},
	}

	s.render(w, r, "dev.tmpl", data)
}

// DevPanic deliberately panics to test recovery middleware.
func (s *Server) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("This is a test panic for middleware testing")
}

// Dev404 returns a 404 error for testing.
func (s *Server) Dev404(w http.ResponseWriter, r *http.Request) {
	s.renderError(w, r, http.StatusNotFound, "Test 404", "This is a test 404 error")
}

// Dev500 returns a 500 error for testing.
func (s *Server) Dev500(w http.ResponseWriter, r *http.Request) {
	s.renderError(w, r, http.StatusInternalServerError, "Test 500", "This is a test 500 error")
}

// Dev500Tmpl tests template error handling.
func (s *Server) Dev500Tmpl(w http.ResponseWriter, r *http.Request) {
	// This should cause a template error
	s.render(w, r, "nonexistent.tmpl", nil)
}

// DevTool represents a development tool.
type DevTool struct {
	Name        string
	URL         string
	Description string
}

// --- Helper Methods ---

// initializeCaches initializes server caches.
func (s *Server) initializeCaches() {
	s.suggestions = s.repo.GenerateAllSearchSuggestions()
	s.artistFilterOpts = s.repo.GetArtistFilterOptions()
	s.locationFilterOpts = s.repo.GetLocationFilterOptions()
	s.searchCache = make(map[string][]*Artist)
	s.cacheSize = 100
}

// loadTemplates compiles all HTML templates.
func (s *Server) loadTemplates() {
	s.templates = make(map[string]*template.Template)

	// Template helper functions
	funcMap := template.FuncMap{
		"hasField": func(obj any, field string) bool {
			// Simplified version - would need reflection for full implementation
			return true
		},
		"contains": func(slice []string, item string) bool {
			return slices.Contains(slice, item)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"join": func(slice []string, sep string) string {
			return strings.Join(slice, sep)
		},
	}

	// Load all template files
	templateFiles := []string{
		"base.tmpl", "home.tmpl", "artists.tmpl", "artist_detail.tmpl",
		"locations.tmpl", "location_detail.tmpl", "search.tmpl",
		"error.tmpl", "dev.tmpl",
	}

	for _, filename := range templateFiles {
		tmpl := template.New(filename).Funcs(funcMap)
		tmpl = template.Must(tmpl.ParseFiles(
			filepath.Join("templates", "base.tmpl"),
			filepath.Join("templates", filename),
		))
		s.templates[filename] = tmpl
	}
}

// render executes a template with data.
func (s *Server) render(w http.ResponseWriter, r *http.Request, templateName string, data any) {
	tmpl, exists := s.templates[templateName]
	if !exists {
		s.renderError(w, r, http.StatusInternalServerError, "Template error", fmt.Sprintf("Template %s not found", templateName))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Template execution error: %v", err)
		s.renderError(w, r, http.StatusInternalServerError, "Template error", "Failed to render page")
	}
}

// renderError renders an error page.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, statusCode int, title, message string) {
	w.WriteHeader(statusCode)

	data := struct {
		Title        string
		ExtraCSS     string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        title,
		ExtraCSS:     "errors.css",
		ErrorCode:    statusCode,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05 UTC"),
	}

	tmpl, exists := s.templates["error.tmpl"]
	if !exists {
		http.Error(w, message, statusCode)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Error template execution failed: %v", err)
		http.Error(w, message, statusCode)
	}
}

// restrictMethod restricts HTTP methods for a handler.
func (s *Server) restrictMethod(handler http.HandlerFunc, allowedMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, method := range allowedMethods {
			if r.Method == method {
				handler(w, r)
				return
			}
		}
		
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		s.renderError(w, r, http.StatusMethodNotAllowed, "Method not allowed", 
			fmt.Sprintf("Method %s not allowed for this endpoint", r.Method))
	}
}

// validateExactPath validates that the request path matches exactly.
func (s *Server) validateExactPath(w http.ResponseWriter, r *http.Request, expectedPath string) bool {
	if r.URL.Path != expectedPath {
		s.renderError(w, r, http.StatusNotFound, "Not found", "The requested resource was not found")
		return false
	}
	return true
}

// parseArtistFilters parses artist filter parameters from form data.
func (s *Server) parseArtistFilters(r *http.Request) ArtistFilterParams {
	var params ArtistFilterParams

	if err := r.ParseForm(); err != nil {
		return params
	}

	// Parse creation year range
	if yearFromStr := r.FormValue("creation_year_from"); yearFromStr != "" {
		if yearFrom, err := strconv.Atoi(yearFromStr); err == nil {
			params.CreationYearFrom = &yearFrom
		}
	}
	if yearToStr := r.FormValue("creation_year_to"); yearToStr != "" {
		if yearTo, err := strconv.Atoi(yearToStr); err == nil {
			params.CreationYearTo = &yearTo
		}
	}

	// Parse member counts
	if memberCountsStr := r.Form["member_counts"]; len(memberCountsStr) > 0 {
		for _, countStr := range memberCountsStr {
			if count, err := strconv.Atoi(countStr); err == nil {
				params.MemberCounts = append(params.MemberCounts, count)
			}
		}
	}

	// Parse countries
	params.Countries = r.Form["countries"]

	return params
}

// parseLocationFilters parses location filter parameters from form data.
func (s *Server) parseLocationFilters(r *http.Request) LocationFilterParams {
	var params LocationFilterParams

	if err := r.ParseForm(); err != nil {
		return params
	}

	// Parse concert count range
	if countFromStr := r.FormValue("concert_count_from"); countFromStr != "" {
		if countFrom, err := strconv.Atoi(countFromStr); err == nil {
			params.ConcertCountFrom = &countFrom
		}
	}
	if countToStr := r.FormValue("concert_count_to"); countToStr != "" {
		if countTo, err := strconv.Atoi(countToStr); err == nil {
			params.ConcertCountTo = &countTo
		}
	}

	// Parse countries
	params.Countries = r.Form["countries"]

	return params
}

// getRandomArtists returns a random subset of artists.
func getRandomArtists(artists []*Artist, count int) []*Artist {
	if len(artists) <= count {
		return artists
	}

	// Create a copy and shuffle
	shuffled := make([]*Artist, len(artists))
	copy(shuffled, artists)

	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

// convertArtistPointersToValues converts artist pointers to values for templates.
func convertArtistPointersToValues(artists []*Artist) []Artist {
	result := make([]Artist, len(artists))
	for i, artist := range artists {
		result[i] = *artist
	}
	return result
}

// convertLocationPointersToValues converts location pointers to values for templates.
func convertLocationPointersToValues(locations []*Location) []Location {
	result := make([]Location, len(locations))
	for i, location := range locations {
		result[i] = *location
	}
	return result
}

// getPort returns the port to listen on.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = config.DefaultPort
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	return port
}