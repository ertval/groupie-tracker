package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"groupie-tracker/internal/config"
)

// Repository manages all application data and provides thread-safe access to it.
type Repository struct {
	// Configuration
	baseURL string
	client  *http.Client
	// Controls whether image caching is enabled
	withCache bool

	// Pre-computed and sorted data collections
	artists         []Artist
	artistsByID     map[int]Artist
	artistsBySlug   map[string]Artist
	locations       []Location
	locationsBySlug map[string]Location
	globalStats     map[string]int
}

// NewRepository creates a new repository instance with the given API URL, timeout,
// and a flag indicating whether to enable local image caching.
func NewRepository(baseURL string, timeout time.Duration) *Repository {
	return &Repository{
		baseURL:   baseURL,
		withCache: config.WithCache,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// LoadData fetches, processes, and pre-computes all data from the API endpoints.
func (r *Repository) LoadData(ctx context.Context) (int, int, int, error) {
	// 1. EXTRACT: Fetch raw data from API endpoints
	var apiArtists []APIArtist
	if err := r.fetchJSON(ctx, "/api/artists", &apiArtists); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to fetch artists: %w", err)
	}

	var apiRelations APIRelation
	if err := r.fetchJSON(ctx, "/api/relation", &apiRelations); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to fetch relations: %w", err)
	}

	// 2. TRANSFORM: Process raw data into rich domain models

	// Create a temporary map for efficient artist lookup during processing
	tempArtists := make(map[int]*Artist)
	for _, apiArtist := range apiArtists {
		tempArtists[apiArtist.ID] = &Artist{
			ID:              apiArtist.ID,
			Name:            apiArtist.Name,
			Slug:            createSlug(apiArtist.Name),
			Members:         apiArtist.Members,
			CreationYear:    apiArtist.CreationYear,
			FirstAlbum:      apiArtist.FirstAlbum,
			Image:           apiArtist.Image,
			Concerts:        []Concert{},
			DatesAtLocation: make(map[string][]string),
		}
	}

	// Process relations and concert data
	for _, relation := range apiRelations.Index {
		if artist, found := tempArtists[relation.ID]; found {
			for location, dates := range relation.DatesLocations {
				normalizedLoc := normalizeLocation(location)
				locationSlug := createSlug(normalizedLoc)
				artist.DatesAtLocation[locationSlug] = append(artist.DatesAtLocation[locationSlug], dates...)

				for _, date := range dates {
					artist.Concerts = append(artist.Concerts, Concert{
						Date:     date,
						Location: normalizedLoc,
					})
				}
			}
		}
	}

	// Create final, sorted slice of artists and compute derived fields
	finalArtists := make([]Artist, 0, len(tempArtists))
	countrySet := make(map[string]bool)
	totalConcerts := 0

	for _, artist := range tempArtists {
		// Sort concerts by date (optional, but good for consistency)
		sort.Slice(artist.Concerts, func(i, j int) bool {
			return artist.Concerts[i].Date < artist.Concerts[j].Date
		})

		// Compute concert count
		artist.ConcertCount = len(artist.Concerts)
		totalConcerts += artist.ConcertCount

		// Compute unique, sorted countries
		artistCountries := make(map[string]bool)
		for _, concert := range artist.Concerts {
			parts := strings.Split(concert.Location, "-")
			if len(parts) > 1 {
				country := strings.TrimSpace(parts[len(parts)-1])
				artistCountries[country] = true
				countrySet[country] = true // Also add to global set
			}
		}
		artist.Countries = make([]string, 0, len(artistCountries))
		for country := range artistCountries {
			artist.Countries = append(artist.Countries, country)
		}
		sort.Strings(artist.Countries)

		finalArtists = append(finalArtists, *artist)
	}

	// Cache images locally. Collect summary counts to avoid noisy per-image logs.
	cachedCount := 0
	downloadedCount := 0
	failedCount := 0
	for i := range finalArtists {
		cached, err := r.cacheImage(&finalArtists[i])
		if err != nil {
			// Count failures but don't log per-artist to avoid noisy output
			failedCount++
			continue
		}
		if cached {
			cachedCount++
		} else {
			downloadedCount++
		}
	}

	// Sort artists by name for consistent ordering
	sort.Slice(finalArtists, func(i, j int) bool {
		return finalArtists[i].Name < finalArtists[j].Name
	})

	// Compute Next/Prev artist IDs for navigation
	for i := range finalArtists {
		if i > 0 {
			finalArtists[i].PrevArtistID = finalArtists[i-1].ID
		}
		if i < len(finalArtists)-1 {
			finalArtists[i].NextArtistID = finalArtists[i+1].ID
		}
	}

	// Process locations
	tempLocations := make(map[string]*Location)
	for i := range finalArtists {
		artist := &finalArtists[i]
		for _, concert := range artist.Concerts {
			loc, found := tempLocations[concert.Location]
			if !found {
				loc = &Location{
					Name:    concert.Location,
					Slug:    createSlug(concert.Location),
					Artists: []Artist{},
				}
				tempLocations[concert.Location] = loc
			}

			// Add artist to location if not already present
			artistFound := false
			for _, locArtist := range loc.Artists {
				if locArtist.ID == artist.ID {
					artistFound = true
					break
				}
			}
			if !artistFound {
				loc.Artists = append(loc.Artists, *artist)
			}
			loc.TotalConcerts++
		}
	}

	finalLocations := make([]Location, 0, len(tempLocations))
	for _, loc := range tempLocations {
		loc.ArtistCount = len(loc.Artists)
		finalLocations = append(finalLocations, *loc)
	}

	sort.Slice(finalLocations, func(i, j int) bool {
		return finalLocations[i].TotalConcerts > finalLocations[j].TotalConcerts
	})

	// 3. LOAD: Populate repository with final, computed data
	r.artists = finalArtists
	r.artistsByID = make(map[int]Artist)
	r.artistsBySlug = make(map[string]Artist)
	for _, artist := range finalArtists {
		r.artistsByID[artist.ID] = artist
		r.artistsBySlug[artist.Slug] = artist
	}

	r.locations = finalLocations
	r.locationsBySlug = make(map[string]Location)
	for _, location := range finalLocations {
		r.locationsBySlug[location.Slug] = location
	}

	totalMembers := 0
	for _, artist := range r.artists {
		totalMembers += len(artist.Members)
	}
	r.globalStats = map[string]int{
		"total_artists":   len(r.artists),
		"total_members":   totalMembers,
		"total_locations": len(r.locations),
		"total_concerts":  totalConcerts,
		"total_countries": len(countrySet),
	}

	return cachedCount, downloadedCount, failedCount, nil
}

// --- Getters ---

// GetArtists returns a pre-sorted slice of all artists.
func (r *Repository) GetArtists() []Artist {
	return r.artists
}

// GetArtistByID returns an artist by their ID.
func (r *Repository) GetArtistByID(id int) (Artist, bool) {
	artist, found := r.artistsByID[id]
	return artist, found
}

// GetArtistBySlug returns an artist by their slug.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	artist, found := r.artistsBySlug[slug]
	return artist, found
}

// GetLocations returns a pre-sorted slice of all locations.
func (r *Repository) GetLocations() []Location {
	return r.locations
}

// GetLocationBySlug returns a location by its slug.
func (r *Repository) GetLocationBySlug(slug string) (Location, bool) {
	location, found := r.locationsBySlug[slug]
	return location, found
}

// GetStats returns pre-computed global statistics.
func (r *Repository) GetStats() map[string]int {
	return r.globalStats
}

// --- Private Helper Methods ---

func (r *Repository) cacheImage(artist *Artist) (bool, error) {
	// If caching is disabled, leave the artist.Image as-is and return false
	// indicating the image was not cached locally.
	if !r.withCache {
		return false, nil
	}
	originalImageURL := artist.Image
	cacheDir := "static/img/artists"
	fileName := fmt.Sprintf("%s.jpg", artist.Slug)
	filePath := filepath.Join(cacheDir, fileName)
	localImagePath := "/" + filepath.ToSlash(filePath)

	// Check if the file already exists
	if _, err := os.Stat(filePath); err == nil {
		artist.Image = localImagePath // File exists, just update path
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err // A different error occurred
	}

	// File does not exist, so download it
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return false, err
	}

	// If originalImageURL is empty, nothing to download
	if strings.TrimSpace(originalImageURL) == "" {
		return false, fmt.Errorf("empty image URL for artist %s", artist.Name)
	}

	resp, err := http.Get(originalImageURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to download image for %s: status %d", artist.Name, resp.StatusCode)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return false, err
	}

	artist.Image = localImagePath // Update to local path after successful download
	return false, nil
}

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

func createSlug(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

func normalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), "_", "-"))
}
