// Package handlers provides HTTP request handlers for the Groupie Tracker application.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

// Handlers contains all HTTP handlers and their dependencies.
type Handlers struct {
	store     *storage.Store
	templates *template.Template
	apiClient *api.Client
}

// Response structures for API endpoints
type SearchResponse struct {
	Artists []models.Artist `json:"artists"`
	Query   string          `json:"query"`
	Total   int             `json:"total"`
}

type SuggestResponse struct {
	Suggestions []string `json:"suggestions"`
	Query       string   `json:"query"`
}

type HealthResponse struct {
	Status string         `json:"status"`
	Stats  map[string]int `json:"stats"`
}

// NewHandlers creates a new handlers instance with the given store and API client.
func NewHandlers(store *storage.Store) *Handlers {
	h := &Handlers{
		store: store,
	}
	h.loadTemplates()
	return h
}

// loadTemplates loads all HTML templates
func (h *Handlers) loadTemplates() {
	templateFiles := []string{
		"templates/base.html",
		"templates/home.html",
		"templates/artists.html",
		"templates/artist_detail.html",
		"templates/locations.html",
		"templates/404.html",
		"templates/500.html",
	}

	var err error
	h.templates, err = template.ParseFiles(templateFiles...)
	if err != nil {
		log.Printf("Warning: Could not load templates: %v", err)
		// Create a simple fallback template
		h.templates = template.Must(template.New("fallback").Parse(`
			<!DOCTYPE html>
			<html><head><title>{{.Title}}</title></head>
			<body><h1>{{.Title}}</h1><div>{{.Content}}</div></body></html>
		`))
	}
}

// SetAPIClient sets the API client for the handlers.
func (h *Handlers) SetAPIClient(client *api.Client) {
	h.apiClient = client
}

// SetTemplates sets the template instance for rendering HTML pages.
func (h *Handlers) SetTemplates(tmpl *template.Template) {
	h.templates = tmpl
}

// HomeHandler handles the home page.
func (h *Handlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists := h.store.GetAllArtists()

	data := struct {
		Title   string
		Artists []models.Artist
	}{
		Title:   "Home",
		Artists: artists,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Try to execute the home template, fall back to simple response if it fails
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "base.html", data); err != nil {
			log.Printf("Template execution error: %v", err)
			// Fallback to simple HTML
			h.writeSimpleHTML(w, "Home", fmt.Sprintf("Found %d artists", len(artists)))
		}
	} else {
		h.writeSimpleHTML(w, "Home", fmt.Sprintf("Found %d artists", len(artists)))
	}
}

// writeSimpleHTML writes a simple HTML response as fallback
func (h *Handlers) writeSimpleHTML(w http.ResponseWriter, title, content string) {
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head><title>%s - Groupie Tracker</title></head>
		<body><h1>%s</h1><p>%s</p></body>
		</html>
	`, title, title, content)
	w.Write([]byte(html))
}

// ArtistsHandler handles the artists listing page.
func (h *Handlers) ArtistsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists := h.store.GetAllArtists()

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Artists - Groupie Tracker</title>
</head>
<body>
    <h1>Artists</h1>
    <ul>`

	for _, artist := range artists {
		html += fmt.Sprintf(`
        <li>
            <a href="/artists/%d">%s</a> (%d)
        </li>`, artist.ID, artist.Name, artist.CreationYear)
	}

	html += `
    </ul>
    <p><a href="/">Back to Home</a></p>
</body>
</html>`

	w.Write([]byte(html))
}

// ArtistDetailHandler handles individual artist detail pages.
func (h *Handlers) ArtistDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract artist ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		http.Redirect(w, r, "/artists", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid artist ID", http.StatusBadRequest)
		return
	}

	artist, exists := h.store.GetArtist(id)
	if !exists {
		h.NotFoundHandler(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s - Groupie Tracker</title>
</head>
<body>
    <h1>%s</h1>
    <img src="%s" alt="%s" style="max-width: 300px;">
    <p><strong>Creation Year:</strong> %d</p>
    <p><strong>First Album:</strong> %s</p>
    <p><strong>Members:</strong></p>
    <ul>`, artist.Name, artist.Name, artist.Image, artist.Name, artist.CreationYear, artist.FirstAlbum)

	for _, member := range artist.Members {
		html += fmt.Sprintf("<li>%s</li>", member)
	}

	html += `
    </ul>
    <p><a href="/artists">Back to Artists</a></p>
</body>
</html>`

	w.Write([]byte(html))
}

// SearchHandler handles search API requests.
func (h *Handlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")

	var artists []models.Artist
	if query == "" {
		artists = h.store.GetAllArtists()
	} else {
		artists = h.store.SearchArtists(query)
	}

	response := SearchResponse{
		Artists: artists,
		Query:   query,
		Total:   len(artists),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode response")
		return
	}
}

// SuggestHandler handles autocomplete suggestion requests.
func (h *Handlers) SuggestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")

	var suggestions []string
	if query != "" && len(query) >= 2 {
		artists := h.store.SearchArtists(query)

		// Create suggestions based on matching artists
		suggestionMap := make(map[string]bool)

		for _, artist := range artists {
			// Add artist name if it matches
			if strings.Contains(strings.ToLower(artist.Name), strings.ToLower(query)) {
				suggestionMap[artist.Name] = true
			}

			// Add member names if they match
			for _, member := range artist.Members {
				if strings.Contains(strings.ToLower(member), strings.ToLower(query)) {
					suggestionMap[member] = true
				}
			}
		}

		// Convert map to slice
		for suggestion := range suggestionMap {
			suggestions = append(suggestions, suggestion)
		}
	}

	response := SuggestResponse{
		Suggestions: suggestions,
		Query:       query,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode response")
		return
	}
}

// HealthHandler handles health check requests.
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.store.GetStats()

	response := HealthResponse{
		Status: "healthy",
		Stats:  stats,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// NotFoundHandler handles 404 errors.
func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Page Not Found - Groupie Tracker</title>
</head>
<body>
    <h1>404 - Page Not Found</h1>
    <p>The page you requested could not be found.</p>
    <p><a href="/">Go to Home</a></p>
</body>
</html>`

	w.Write([]byte(html))
}

// InternalErrorHandler handles 500 errors.
func (h *Handlers) InternalErrorHandler(w http.ResponseWriter, r *http.Request, message string) {
	log.Printf("Internal server error: %s", message)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusInternalServerError)

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Internal Server Error - Groupie Tracker</title>
</head>
<body>
    <h1>500 - Internal Server Error</h1>
    <p>Something went wrong on our end. Please try again later.</p>
    <p><a href="/">Go to Home</a></p>
</body>
</html>`

	w.Write([]byte(html))
}

// LocationsHandler handles the locations page.
func (h *Handlers) LocationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	locations := h.store.GetUniqueLocations()

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Locations - Groupie Tracker</title>
</head>
<body>
    <h1>Concert Locations</h1>
    <ul>`

	for _, location := range locations {
		html += fmt.Sprintf(`
        <li>%s</li>`, location)
	}

	html += `
    </ul>
    <p><a href="/">Back to Home</a></p>
</body>
</html>`

	w.Write([]byte(html))
}

// RefreshHandler handles data refresh requests (POST /api/refresh).
func (h *Handlers) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.apiClient == nil {
		http.Error(w, "API client not configured", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch fresh data from API
	data, err := h.apiClient.FetchAllData(ctx)
	if err != nil {
		log.Printf("Failed to refresh data: %v", err)
		http.Error(w, "Failed to refresh data", http.StatusInternalServerError)
		return
	}

	// Update store with new data
	storeData := storage.StoreData{
		Artists:   data.Artists,
		Locations: data.Locations,
		Dates:     data.Dates,
		Relations: data.Relations,
	}
	h.store.LoadData(storeData)

	// Return success response
	response := struct {
		Status  string         `json:"status"`
		Message string         `json:"message"`
		Stats   map[string]int `json:"stats"`
	}{
		Status:  "success",
		Message: "Data refreshed successfully",
		Stats:   h.store.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode refresh response")
		return
	}

	log.Printf("Data refreshed: %d artists, %d locations, %d dates, %d relations",
		len(data.Artists), len(data.Locations), len(data.Dates), len(data.Relations))
}
