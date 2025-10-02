package web

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/data"
)

// Artists handles the artists listing page.
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
	// Validate path for both GET and POST using centralized utility
	if !s.validateExactPath(w, r, "/artists") {
		return
	}

	artists := s.svc.Artists()
	filterOptions := s.svc.GetArtistFilterOptions()
	suggestions := s.svc.GenerateAllSearchSuggestions()
	var appliedFilters data.ArtistFilterParams
	totalArtists := len(artists)

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !s.parseFormOrError(w, r) {
			return
		}

		appliedFilters = parseArtistFilterParams(r)
		artists = s.svc.FilterArtists(appliedFilters)
	}

	// Sort artists by concert count (descending) for main display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].ConcertCount > artists[j].ConcertCount
	})

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Suggestions    []data.SearchSuggestion
		Artists        []data.Artist
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsFiltered     bool
		TotalArtists   int
	}{
		Title:          "Artists",
		ExtraCSS:       "artists.css",
		ExtraJS:        "",
		Suggestions:    suggestions,
		Artists:        artists,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsFiltered:     r.Method == http.MethodPost,
		TotalArtists:   totalArtists,
	}

	s.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (s *Server) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	// Validate path
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		s.NotFoundError(w, r, "")
		return
	}

	// Try slug first, then ID
	artist, found := s.svc.ArtistBySlug(path)
	if !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = s.svc.ArtistByID(id)
		}
		if !found {
			s.NotFoundError(w, r, "Artist not found")
			return
		}
	}

	// Get navigation artists using on-demand lookup
	prevArtist, nextArtist := s.svc.GetAdjacentArtists(artist.ID)
	suggestions := s.svc.GenerateAllSearchSuggestions()

	data := struct {
		Title       string
		ExtraCSS    string
		ExtraJS     string
		Suggestions []data.SearchSuggestion
		Artist      data.Artist
		PrevArtist  *data.Artist
		NextArtist  *data.Artist
	}{
		Title:       artist.Name,
		ExtraCSS:    "artist_detail.css",
		ExtraJS:     "",
		Suggestions: suggestions,
		Artist:      artist,
		PrevArtist:  prevArtist,
		NextArtist:  nextArtist,
	}

	s.render(w, r, "artist_detail.tmpl", data)
}
