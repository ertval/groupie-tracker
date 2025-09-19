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

	"groupie-tracker/internal/config"
)

// Repository manages all application data and provides thread-safe access to it.
type Repository struct {
	// Configuration
	apiEndpoint string
	apiClient   *http.Client
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

// NewRepository creates a new repository instance using values from the
// `internal/config` package. This keeps configuration centralized and makes
// tests simpler by allowing them to override `config` variables.
func NewRepository() *Repository {
	return &Repository{
		apiEndpoint: config.APIBaseURL,
		apiClient: &http.Client{
			Timeout: config.APIRequestTimeout,
		},
		withCache: config.WithCache,
	}
}

// LoadData fetches, transforms, and loads all data used by the application.
// It orchestrates the full ETL pipeline:
//  1. Extract:   fetch artists and relations from the remote API
//  2. Transform: build rich Artist models, compute stats, cache images, build Locations
//  3. Load:      populate in-memory collections and indexes for fast lookups
func (r *Repository) LoadData(ctx context.Context) error {
	// 1) Extract
	apiArtists, apiRelations, err := r.fetchAPIData(ctx)
	if err != nil {
		return err
	}

	// 2) Transform
	finalArtists, totalConcerts, countrySet := r.buildArtists(apiArtists, apiRelations)

	cachedCount, downloadedCount, failedCount := r.cacheArtistImages(finalArtists)

	finalArtists = r.sortAndLinkArtists(finalArtists)

	finalLocations := r.buildLocations(finalArtists)

	// 3) Load
	r.populateData(finalArtists, finalLocations, totalConcerts, countrySet, cachedCount, downloadedCount, failedCount)
	return nil
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

// fetchAPIData retrieves raw API payloads needed for repository initialization.
// Returns the full set of artists and relations straight from the API.
func (r *Repository) fetchAPIData(ctx context.Context) ([]APIArtist, APIRelation, error) {
	var apiArtists []APIArtist
	if err := r.fetchJSON(ctx, "/api/artists", &apiArtists); err != nil {
		return nil, APIRelation{}, fmt.Errorf("failed to fetch artists: %w", err)
	}

	var apiRelations APIRelation
	if err := r.fetchJSON(ctx, "/api/relation", &apiRelations); err != nil {
		return nil, APIRelation{}, fmt.Errorf("failed to fetch relations: %w", err)
	}
	return apiArtists, apiRelations, nil
}

// buildArtists consolidates API data into enriched Artist models.
// It also computes artist-level derived fields and returns aggregate counters
// required to build global statistics (totalConcerts and distinct countries).
func (r *Repository) buildArtists(apiArtists []APIArtist, apiRelations APIRelation) ([]Artist, int, map[string]bool) {
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
		sort.Slice(artist.Concerts, func(i, j int) bool {
			return artist.Concerts[i].Date < artist.Concerts[j].Date
		})

		artist.ConcertCount = len(artist.Concerts)
		totalConcerts += artist.ConcertCount

		// Compute unique, sorted countries
		artistCountries := make(map[string]bool)
		for _, concert := range artist.Concerts {
			parts := strings.Split(concert.Location, "-")
			if len(parts) > 1 {
				country := strings.TrimSpace(parts[len(parts)-1])
				artistCountries[country] = true
				countrySet[country] = true
			}
		}
		artist.Countries = make([]string, 0, len(artistCountries))
		for country := range artistCountries {
			artist.Countries = append(artist.Countries, country)
		}
		sort.Strings(artist.Countries)

		finalArtists = append(finalArtists, *artist)
	}

	return finalArtists, totalConcerts, countrySet
}

// cacheArtistImages caches remote artist images locally when enabled and returns
// aggregate counters for reporting.
// Behavior:
//   - When caching is disabled, this function is a no-op and returns zeros.
//   - When caching is enabled, for each artist it either:
//   - reuses an existing cached file (counted as cached), or
//   - downloads the image (counted as downloaded), or
//   - records a failure (counted as failed) and leaves the original URL.
func (r *Repository) cacheArtistImages(artists []Artist) (int, int, int) {
	// Fast path when caching is disabled: leave images untouched and return zeros.
	if !r.withCache {
		return 0, 0, 0
	}

	cachedCount := 0
	downloadedCount := 0
	failedCount := 0

	cacheDir := "static/img/artists"
	// Ensure cache directory exists before we start.
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		// If we cannot create the cache directory, treat all as failures but keep running.
		return 0, 0, len(artists)
	}

	for i := range artists {
		artist := &artists[i]
		originalImageURL := artist.Image
		fileName := fmt.Sprintf("%s.jpg", artist.Slug)
		filePath := filepath.Join(cacheDir, fileName)
		localImagePath := "/" + filepath.ToSlash(filePath)

		// If the file is already cached, just point to it.
		if _, err := os.Stat(filePath); err == nil {
			artist.Image = localImagePath
			cachedCount++
			continue
		} else if !os.IsNotExist(err) {
			failedCount++
			continue
		}

		// Skip empty URLs (record as failure to keep counters consistent with earlier behavior)
		if strings.TrimSpace(originalImageURL) == "" {
			failedCount++
			continue
		}

		// Download and save the image.
		resp, err := http.Get(originalImageURL)
		if err != nil {
			failedCount++
			continue
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				failedCount++
				return
			}
			file, err := os.Create(filePath)
			if err != nil {
				failedCount++
				return
			}
			defer file.Close()
			if _, err := io.Copy(file, resp.Body); err != nil {
				failedCount++
				return
			}
			// Success: point the artist image to the local path and count as downloaded.
			artist.Image = localImagePath
			downloadedCount++
		}()
	}
	return cachedCount, downloadedCount, failedCount
}

// sortAndLinkArtists sorts artists by name and sets prev/next navigation IDs.
func (r *Repository) sortAndLinkArtists(artists []Artist) []Artist {
	sort.Slice(artists, func(i, j int) bool { return artists[i].Name < artists[j].Name })
	for i := range artists {
		if i > 0 {
			artists[i].PrevArtistID = artists[i-1].ID
		}
		if i < len(artists)-1 {
			artists[i].NextArtistID = artists[i+1].ID
		}
	}
	return artists
}

// buildLocations aggregates concerts per location, including artist lists and counts.
func (r *Repository) buildLocations(artists []Artist) []Location {
	tempLocations := make(map[string]*Location)
	for i := range artists {
		artist := &artists[i]
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
	return finalLocations
}

// populateData loads computed collections and indexes into the repository
// along with global statistics.
func (r *Repository) populateData(finalArtists []Artist, finalLocations []Location, totalConcerts int, countrySet map[string]bool, cachedCount, downloadedCount, failedCount int) {
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
		"total_artists":     len(r.artists),
		"total_members":     totalMembers,
		"total_locations":   len(r.locations),
		"total_concerts":    totalConcerts,
		"total_countries":   len(countrySet),
		"cached_images":     cachedCount,
		"downloaded_images": downloadedCount,
		"failed_images":     failedCount,
	}
}

// fetchJSON performs a GET request against the repository's baseURL+path and
// decodes the JSON response into dest. It returns a wrapped error with context
// if any network or decoding issues occur.

func (r *Repository) fetchJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", r.apiEndpoint+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := r.apiClient.Do(req)
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
