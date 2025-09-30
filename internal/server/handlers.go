package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/data"
	"groupie-tracker/internal/search"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if !s.ensureExactPath(w, r, "/") {
		return
	}

	artists := featuredArtists(s.store.Artists(), 8)

	payload := struct {
		Title          string
		ExtraCSS       string
		Suggestions    []search.Suggestion
		Artists        []data.Artist
		TotalArtists   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		Suggestions:    s.search.Suggestions(),
		Artists:        artists,
		TotalArtists:   len(s.store.Artists()),
		TotalLocations: len(s.store.Locations()),
	}

	s.templates.Render(w, "home.tmpl", payload)
}

func (s *Server) handleArtists(w http.ResponseWriter, r *http.Request) {
	if !s.ensureExactPath(w, r, "/artists") {
		return
	}

	artists := s.store.Artists()
	filters := search.Filters{}
	isFiltered := false

	switch r.Method {
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			s.renderError(w, r, http.StatusBadRequest, "Invalid form data")
			return
		}
		filters = search.ParseFilters(r.Form)
	case http.MethodGet:
		filters = search.ParseFilters(r.URL.Query())
	}

	if !filters.IsEmpty() {
		result := s.search.Search(search.Params{Filters: filters})
		artists = result.Artists
		isFiltered = true
	}

	payload := struct {
		Title          string
		ExtraCSS       string
		Suggestions    []search.Suggestion
		Artists        []data.Artist
		FilterOptions  search.FilterOptions
		AppliedFilters search.Filters
		IsFiltered     bool
		TotalArtists   int
	}{
		Title:          "Artists",
		ExtraCSS:       "artists.css",
		Suggestions:    s.search.Suggestions(),
		Artists:        artists,
		FilterOptions:  s.search.FilterOptions(),
		AppliedFilters: filters,
		IsFiltered:     isFiltered,
		TotalArtists:   len(s.store.Artists()),
	}

	s.templates.Render(w, "artists.tmpl", payload)
}

func (s *Server) handleArtistDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/artists/")
	if slug == "" {
		s.notFound(w, r, "Artist not found")
		return
	}

	artist, ok := s.store.ArtistBySlug(slug)
	if !ok {
		if id, err := strconv.Atoi(slug); err == nil {
			artist, ok = s.store.ArtistByID(id)
		}
	}

	if !ok {
		s.notFound(w, r, "Artist not found")
		return
	}

	prev, next := adjacentArtists(s.store.Artists(), artist.ID)

	payload := struct {
		Title       string
		ExtraCSS    string
		Suggestions []search.Suggestion
		Artist      data.Artist
		PrevArtist  *data.Artist
		NextArtist  *data.Artist
	}{
		Title:       artist.Name,
		ExtraCSS:    "artist_detail.css",
		Suggestions: s.search.Suggestions(),
		Artist:      artist,
		PrevArtist:  prev,
		NextArtist:  next,
	}

	s.templates.Render(w, "artist_detail.tmpl", payload)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !s.ensureExactPath(w, r, "/search") {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	filters := search.ParseFilters(r.URL.Query())

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.renderError(w, r, http.StatusBadRequest, "Invalid form data")
			return
		}
		query = strings.TrimSpace(r.FormValue("q"))
		filters = search.ParseFilters(r.Form)
	}

	result := search.Result{}
	if query != "" || !filters.IsEmpty() {
		result = s.search.Search(search.Params{Query: query, Filters: filters})
	}

	payload := struct {
		Title         string
		ExtraCSS      string
		Suggestions   []search.Suggestion
		Query         string
		Results       search.Result
		FilterOptions search.FilterOptions
		Filters       search.Filters
	}{
		Title:         "Search",
		ExtraCSS:      "search.css",
		Suggestions:   s.search.Suggestions(),
		Query:         query,
		Results:       result,
		FilterOptions: s.search.FilterOptions(),
		Filters:       filters,
	}

	s.templates.Render(w, "search.tmpl", payload)
}

func (s *Server) handleLocations(w http.ResponseWriter, r *http.Request) {
	if !s.ensureExactPath(w, r, "/locations") {
		return
	}

	payload := struct {
		Title       string
		ExtraCSS    string
		Suggestions []search.Suggestion
		Locations   []data.Location
	}{
		Title:       "Locations",
		ExtraCSS:    "locations.css",
		Suggestions: s.search.Suggestions(),
		Locations:   s.store.Locations(),
	}

	s.templates.Render(w, "locations.tmpl", payload)
}

func (s *Server) handleLocationDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		s.notFound(w, r, "Location not found")
		return
	}

	location, ok := s.store.LocationBySlug(slug)
	if !ok {
		s.notFound(w, r, "Location not found")
		return
	}

	payload := struct {
		Title       string
		ExtraCSS    string
		Suggestions []search.Suggestion
		Location    data.Location
	}{
		Title:       location.Name,
		ExtraCSS:    "location_detail.css",
		Suggestions: s.search.Suggestions(),
		Location:    location,
	}

	s.templates.Render(w, "location_detail.tmpl", payload)
}

func (s *Server) handleSuggestions(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	suggestions := s.search.Suggest(query, 20)

	s.respondJSON(w, http.StatusOK, map[string]any{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]any{
		"status": "healthy",
		"stats":  s.store.Stats(),
	})
}

func (s *Server) ensureExactPath(w http.ResponseWriter, r *http.Request, expected string) bool {
	if r.URL.Path != expected {
		s.notFound(w, r, "")
		return false
	}
	return true
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "The requested page was not found."
	}

	payload := struct {
		Title        string
		ExtraCSS     string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        "Page Not Found",
		ExtraCSS:     "errors.css",
		ErrorCode:    http.StatusNotFound,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	w.WriteHeader(http.StatusNotFound)
	s.templates.Render(w, "error.tmpl", payload)
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, code int, message string) {
	payload := struct {
		Title        string
		ExtraCSS     string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        "Error",
		ExtraCSS:     "errors.css",
		ErrorCode:    code,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	w.WriteHeader(code)
	s.templates.Render(w, "error.tmpl", payload)
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func featuredArtists(artists []data.Artist, count int) []data.Artist {
	if count <= 0 || len(artists) <= count {
		return artists
	}

	step := len(artists) / count
	selection := make([]data.Artist, 0, count)
	for i := 0; i < count && i*step < len(artists); i++ {
		selection = append(selection, artists[i*step])
	}
	return selection
}

func adjacentArtists(artists []data.Artist, artistID int) (*data.Artist, *data.Artist) {
	for i := range artists {
		if artists[i].ID != artistID {
			continue
		}

		var prev, next *data.Artist
		if i > 0 {
			prev = &artists[i-1]
		}
		if i < len(artists)-1 {
			next = &artists[i+1]
		}
		return prev, next
	}

	return nil, nil
}
