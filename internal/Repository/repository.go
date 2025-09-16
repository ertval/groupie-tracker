// Package app provides the core functionality for the Groupie Tracker application.
// This consolidates the data layer, API client, and domain models into a single,
// cohesive package following idiomatic Go patterns.
package repository

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

// Artist represents a musical artist.
type Artist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Image        string   `json:"image"`
	Slug         string   `json:"-"` // SEO-friendly URL slug
}

// Concert represents concert information for an artist.
type Concert struct {
	ID        int                 `json:"id"`
	Locations map[string][]string `json:"datesLocations"`
}

// Response represents the combined API response (for testing).
type Response struct {
	Artists   []Artist  `json:"artists,omitempty"`
	Relations []Concert `json:"relations,omitempty"`
}

// LocationStats holds statistics for a location.
type LocationStats struct {
	Name         string
	DisplayName  string
	Slug         string
	Artists      []Artist
	ArtistCount  int
	TotalShows   int
	ConcertCount int // Total number of concerts at this location
}

// Store manages all application data and API interactions.
type Repository struct {
	artists   map[int]Artist
	concerts  map[int]Concert
	slugToID  map[string]int
	locations []string
	stats     map[string]int
	baseURL   string
	client    *http.Client
}

// NewRepository creates a new data repository with the given API URL and timeout.
func NewRepository(baseURL string, timeout time.Duration) *Repository {
	return &Repository{
		artists:  make(map[int]Artist),
		concerts: make(map[int]Concert),
		slugToID: make(map[string]int),
		stats:    make(map[string]int),
		baseURL:  baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// LoadData fetches and loads all data from the API.
func (r *Repository) LoadData(ctx context.Context) error {
	// Fetch artists
	var artists []Artist
	if err := r.fetchJSON(ctx, "/api/artists", &artists); err != nil {
		return fmt.Errorf("failed to fetch artists: %w", err)
	}

	// Fetch concert data
	var relationResponse struct {
		Index []Concert `json:"index"`
	}
	if err := r.fetchJSON(ctx, "/api/relation", &relationResponse); err != nil {
		return fmt.Errorf("failed to fetch concerts: %w", err)
	}

	// Process and store data
	r.processArtists(artists)
	r.processConcerts(relationResponse.Index)
	r.computeStats()

	return nil
}

// GetArtists returns all artists sorted by name.
func (r *Repository) GetArtists() []Artist {
	artists := make([]Artist, 0, len(r.artists))
	for _, artist := range r.artists {
		artists = append(artists, artist)
	}
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})
	return artists
}

// GetArtist returns an artist by ID.
func (r *Repository) GetArtist(id int) (Artist, bool) {
	artist, exists := r.artists[id]
	return artist, exists
}

// GetArtistBySlug returns an artist by slug.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	id, exists := r.slugToID[slug]
	if !exists {
		return Artist{}, false
	}
	return r.GetArtist(id)
}

// GetConcert returns concert data for an artist.
func (r *Repository) GetConcert(artistID int) (Concert, bool) {
	concert, exists := r.concerts[artistID]
	return concert, exists
}

// GetLocations returns all unique locations.
func (r *Repository) GetLocations() []string {
	return r.locations
}

// GetLocationStats returns statistics for all locations.
func (r *Repository) GetLocationStats() []LocationStats {
	locationMap := make(map[string]LocationStats)

	// Process all concerts to build location stats
	for _, concert := range r.concerts {
		for location, dates := range concert.Locations {
			normalizedLocation := normalizeLocation(location)

			stat, exists := locationMap[normalizedLocation]
			if !exists {
				stat = LocationStats{
					Name:        normalizedLocation,
					DisplayName: location,
					Slug:        createSlug(normalizedLocation),
					Artists:     make([]Artist, 0),
				}
			}

			// Add artist to location if not already there
			artist, exists := r.artists[concert.ID]
			if exists && !containsArtist(stat.Artists, artist) {
				stat.Artists = append(stat.Artists, artist)
			}

			stat.ArtistCount = len(stat.Artists)
			stat.TotalShows += len(dates)
			stat.ConcertCount = stat.TotalShows // Set ConcertCount to same as TotalShows
			locationMap[normalizedLocation] = stat
		}
	}

	// Convert map to slice and sort
	stats := make([]LocationStats, 0, len(locationMap))
	for _, stat := range locationMap {
		stats = append(stats, stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalShows > stats[j].TotalShows
	})

	return stats
}

// GetLocationBySlug returns location details by slug.
func (r *Repository) GetLocationBySlug(slug string) (LocationStats, bool) {
	stats := r.GetLocationStats()
	for _, stat := range stats {
		if stat.Slug == slug {
			return stat, true
		}
	}
	return LocationStats{}, false
}

// GetStats returns computed statistics.
func (r *Repository) GetStats() map[string]int {
	return r.stats
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

// CountShows returns the total number of shows for a concert.
func (r *Repository) CountShows(concert Concert) int {
	total := 0
	for _, dates := range concert.Locations {
		total += len(dates)
	}
	return total
}

// GetCountries extracts unique countries from concert locations.
func (r *Repository) GetCountries(concert Concert) []string {
	countryMap := make(map[string]bool)
	for location := range concert.Locations {
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

// Private helper methods

func (r *Repository) fetchJSON(ctx context.Context, path string, dest interface{}) error {
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
		artist.Slug = createSlug(artist.Name)
		r.artists[artist.ID] = artist
		r.slugToID[artist.Slug] = artist.ID
	}
}

func (r *Repository) processConcerts(concerts []Concert) {
	for _, concert := range concerts {
		r.concerts[concert.ID] = concert
	}
}

func (r *Repository) computeStats() {
	// Compute unique locations
	locationSet := make(map[string]bool)
	totalShows := 0
	totalMembers := 0
	countrySet := make(map[string]bool)

	for _, artist := range r.artists {
		totalMembers += len(artist.Members)
	}

	for _, concert := range r.concerts {
		for location, dates := range concert.Locations {
			locationSet[normalizeLocation(location)] = true
			totalShows += len(dates)

			// Extract country
			parts := strings.Split(location, "-")
			if len(parts) > 1 {
				country := strings.TrimSpace(parts[len(parts)-1])
				countrySet[country] = true
			}
		}
	}

	// Convert set to slice for locations
	r.locations = make([]string, 0, len(locationSet))
	for location := range locationSet {
		r.locations = append(r.locations, location)
	}
	sort.Strings(r.locations)

	// Store computed stats
	r.stats["total_artists"] = len(r.artists)
	r.stats["total_members"] = totalMembers
	r.stats["total_locations"] = len(locationSet)
	r.stats["total_shows"] = totalShows
	r.stats["total_countries"] = len(countrySet)
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

func containsArtist(artists []Artist, target Artist) bool {
	for _, artist := range artists {
		if artist.ID == target.ID {
			return true
		}
	}
	return false
}
