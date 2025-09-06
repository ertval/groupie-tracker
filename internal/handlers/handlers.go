// Package handlers provides HTTP request handlers for the Groupie Tracker application.
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

// Handlers contains all HTTP handlers and their dependencies.
type Handlers struct {
	store     *storage.Store
	templates *template.Template
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

// NewHandlers creates a new handlers instance with the given store.
func NewHandlers(store *storage.Store) *Handlers {
	return &Handlers{
		store: store,
	}
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

	// For now, return a simple response since we don't have templates yet
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	
	stats := h.store.GetStats()
	
	response := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Groupie Tracker</title>
</head>
<body>
    <h1>Welcome to Groupie Tracker</h1>
    <p>Total Artists: %d</p>
    <p>Total Locations: %d</p>
    <p>Total Dates: %d</p>
    <p>Total Relations: %d</p>
    <p><a href="/artists">View All Artists</a></p>
</body>
</html>`, stats["artists"], stats["locations"], stats["dates"], stats["relations"])
	
	w.Write([]byte(response))
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
