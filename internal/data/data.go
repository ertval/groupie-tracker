// Package data provides the core data management functionality for the Groupie Tracker application.
// This package follows idiomatic Go patterns with clear separation between API responses,
// domain models, and the repository that manages all data operations.
package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Artist represents a musical artist with all their concert information.
// This unifies both artist data and concert relations in a single struct.
type Artist struct {
	ID           int                 `json:"id"`
	Name         string              `json:"name"`
	Members      []string            `json:"members"`
	CreationYear int                 `json:"creationDate"`
	FirstAlbum   string              `json:"firstAlbum"`
	Image        string              `json:"image"`
	Slug         string              `json:"-"`                        // SEO-friendly URL slug, computed at runtime
	Concerts     map[string][]string `json:"datesLocations,omitempty"` // location -> dates
}

// Relation represents the API response from /api/relation endpoint.
type Relation struct {
	Index []struct {
		ID             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
}

// LocationStats holds minimal statistics for a location.
type LocationStats struct {
	Name    string   `json:"name"`
	Slug    string   `json:"slug"`
	Artists []Artist `json:"artists"`
	// Computed fields for templates and stats
	ArtistCount   int `json:"artist_count"`
	TotalConcerts int `json:"total_concerts"`
	// ConcertDates maps artist ID to the dates they played at this location
	ConcertDates map[int][]string `json:"concert_dates,omitempty"`
	// ConcertYears maps artist ID to the unique years they played at this location

}

// ComputedData holds the core application data.
type ComputedData struct {
	artists       map[int]*Artist
	slugToID      map[string]int
	locationStats map[string]*LocationStats
}

// Repository manages all application data and provides thread-safe access to it.
type Repository struct {
	baseURL string
	client  *http.Client
	data    *ComputedData
}

// NewRepository creates a new repository instance with the given API URL and timeout.
func NewRepository(baseURL string, timeout time.Duration) *Repository {
	return &Repository{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
		data: &ComputedData{
			artists:       make(map[int]*Artist),
			slugToID:      make(map[string]int),
			locationStats: make(map[string]*LocationStats),
		},
	}
}

// LoadData fetches and processes all data from the API endpoints.
func (r *Repository) LoadData(ctx context.Context) error {
	// Fetch artists from API
	var artists []Artist
	if err := r.fetchJSON(ctx, "/api/artists", &artists); err != nil {
		return fmt.Errorf("failed to fetch artists: %w", err)
	}

	// Fetch relations from API
	var relationsResp Relation
	if err := r.fetchJSON(ctx, "/api/relation", &relationsResp); err != nil {
		return fmt.Errorf("failed to fetch relations: %w", err)
	}

	// Process and compute data
	r.processArtists(artists)
	r.processRelations(relationsResp.Index)
	r.computeLocationStats()

	return nil
}

// GetArtists returns all artists sorted by name.
func (r *Repository) GetArtists() []Artist {
	artists := make([]Artist, 0, len(r.data.artists))
	for _, artist := range r.data.artists {
		artists = append(artists, *artist)
	}
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})
	return artists
}

// GetArtist returns an artist by ID.
func (r *Repository) GetArtist(id int) (Artist, bool) {
	artist, exists := r.data.artists[id]
	if !exists {
		return Artist{}, false
	}
	return *artist, true
}

// GetArtistBySlug returns an artist by SEO slug.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	id, exists := r.data.slugToID[slug]
	if !exists {
		return Artist{}, false
	}
	return r.GetArtist(id)
}

// GetLocationStats returns statistics for all locations sorted by artist count.
func (r *Repository) GetLocationStats() []LocationStats {
	stats := make([]LocationStats, 0, len(r.data.locationStats))
	for _, stat := range r.data.locationStats {
		stats = append(stats, *stat)
	}
	// Sort locations by total concerts (descending)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalConcerts > stats[j].TotalConcerts
	})
	return stats
}

// GetLocationBySlug returns location details by SEO slug.
func (r *Repository) GetLocationBySlug(slug string) (LocationStats, bool) {
	for _, location := range r.data.locationStats {
		if location.Slug == slug {
			return *location, true
		}
	}
	return LocationStats{}, false
}

// GetLocations returns all unique location names.
func (r *Repository) GetLocations() []string {
	locations := make([]string, 0, len(r.data.locationStats))
	for _, location := range r.data.locationStats {
		locations = append(locations, location.Name)
	}
	sort.Strings(locations)
	return locations
}

// GetNextPrevArtist returns navigation info for an artist.
func (r *Repository) GetNextPrevArtist(current Artist) (prev, next *Artist) {
	artists := r.GetArtists()
	for i, artist := range artists {
		if artist.ID == current.ID {
			if i > 0 {
				prev = &artists[i-1]
			}
			if i < len(artists)-1 {
				next = &artists[i+1]
			}
			break
		}
	}
	return prev, next
}

// CountConcerts returns the total number of concerts for an artist.
func (r *Repository) CountConcerts(artist Artist) int {
	total := 0
	for _, dates := range artist.Concerts {
		total += len(dates)
	}
	return total
}

// GetCountries extracts unique countries from an artist's concert locations.
func (r *Repository) GetCountries(artist Artist) []string {
	countryMap := make(map[string]bool)
	for location := range artist.Concerts {
		parts := strings.Split(location, "-")
		if len(parts) > 1 {
			country := strings.TrimSpace(parts[len(parts)-1])
			countryMap[country] = true
		}
	}

	countries := make([]string, 0, len(countryMap))
	for country := range countryMap {
		countries = append(countries, country)
	}
	sort.Strings(countries)
	return countries
}

// GetStats returns computed global statistics on demand.
func (r *Repository) GetStats() map[string]int {
	totalMembers := 0
	totalConcerts := 0
	countrySet := make(map[string]bool)

	for _, artist := range r.data.artists {
		totalMembers += len(artist.Members)
		for location, dates := range artist.Concerts {
			totalConcerts += len(dates)
			// Extract country
			parts := strings.Split(location, "-")
			if len(parts) > 1 {
				country := strings.TrimSpace(parts[len(parts)-1])
				countrySet[country] = true
			}
		}
	}

	return map[string]int{
		"total_artists":   len(r.data.artists),
		"total_members":   totalMembers,
		"total_locations": len(r.data.locationStats),
		"total_concerts":  totalConcerts,
		"total_countries": len(countrySet),
	}
}

// Private helper methods

func (r *Repository) fetchJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", r.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

func (r *Repository) processArtists(artists []Artist) {
	for _, artist := range artists {
		// Generate SEO slug
		artist.Slug = createSlug(artist.Name)

		// Store artist as pointer for efficiency
		r.data.artists[artist.ID] = &artist
		r.data.slugToID[artist.Slug] = artist.ID
	}
}

func (r *Repository) processRelations(apiRelations []struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}) {
	for _, apiRelation := range apiRelations {
		if artist, exists := r.data.artists[apiRelation.ID]; exists {
			artist.Concerts = apiRelation.DatesLocations
		}
	}
}

func (r *Repository) computeLocationStats() {
	// computeLocationStats computes per-location aggregates (artists, dates, totals)
	// Process all artists to build location stats
	for _, artist := range r.data.artists {
		for location := range artist.Concerts {
			normalizedLocation := normalizeLocation(location)

			// Get or create location stats
			locationStat, exists := r.data.locationStats[normalizedLocation]
			if !exists {
				locationStat = &LocationStats{
					Name:    normalizedLocation,
					Slug:    createSlug(normalizedLocation),
					Artists: make([]Artist, 0),
				}
				r.data.locationStats[normalizedLocation] = locationStat
			}

			// Add artist if not already present
			if !containsArtistPtr(locationStat.Artists, artist) {
				locationStat.Artists = append(locationStat.Artists, *artist)
			}
		}
	}

	// After collecting artists per location, compute the derived stats
	for _, loc := range r.data.locationStats {
		loc.ArtistCount = len(loc.Artists)
		loc.ConcertDates = make(map[int][]string)

		totalConcerts := 0

		// For each artist in this location, collect dates and counts
		for _, artist := range loc.Artists {
			// Artist.Concerts keys are the raw location names; find matching entries
			for rawLocation, dates := range artist.Concerts {
				if normalizeLocation(rawLocation) == loc.Name {
					// map artist ID to dates for template consumption
					loc.ConcertDates[artist.ID] = append(loc.ConcertDates[artist.ID], dates...)
					totalConcerts += len(dates)
				}
			}
		}

		// TotalConcerts is the sum of all concert dates at this location
		loc.TotalConcerts = totalConcerts
	}
}

func containsArtistPtr(artists []Artist, target *Artist) bool {
	for _, artist := range artists {
		if artist.ID == target.ID {
			return true
		}
	}
	return false
}

// Utility functions

func createSlug(name string) string {
	// Convert to lowercase and replace non-alphanumeric with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

func normalizeLocation(location string) string {
	return strings.ToLower(strings.TrimSpace(location))
}
