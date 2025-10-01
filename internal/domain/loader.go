package domain

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"groupie-tracker/internal/api"
)

// processArtists transforms raw API data into enriched Artist domain models.
func (s *Store) processArtists(apiArtists []api.Artist, apiRelations api.Relation) []Artist {
	artists := s.transformAPIArtists(apiArtists)
	artists = s.addConcertData(artists, apiRelations)
	return artists
}

// transformAPIArtists converts raw API artist data to domain Artist objects.
func (s *Store) transformAPIArtists(apiArtists []api.Artist) []Artist {
	artists := make([]Artist, 0, len(apiArtists))

	for _, apiArtist := range apiArtists {
		artist := Artist{
			ID:              apiArtist.ID,
			Name:            apiArtist.Name,
			Slug:            createSlug(apiArtist.Name),
			Members:         apiArtist.Members,
			CreationYear:    apiArtist.CreationYear,
			FirstAlbum:      apiArtist.FirstAlbum,
			Image:           apiArtist.Image,
			DatesAtLocation: make(map[string][]string),
		}
		artists = append(artists, artist)
	}

	return artists
}

// addConcertData enriches artists with concert information from API relations.
func (s *Store) addConcertData(artists []Artist, apiRelations api.Relation) []Artist {
	// Index relations by artist ID for efficient lookup
	relationMap := make(map[int]api.RelationIndex)
	for _, rel := range apiRelations.Index {
		relationMap[rel.ID] = rel
	}

	// Add concert data to each artist
	for i := range artists {
		artist := &artists[i]

		if rel, exists := relationMap[artist.ID]; exists {
			countries := make(map[string]bool)

			// Process each location and its dates
			for location, dates := range rel.DatesLocations {
				normalizedLoc := normalizeLocation(location)
				locationSlug := createSlug(normalizedLoc)
				artist.DatesAtLocation[locationSlug] = dates

				// Create concert objects
				for _, date := range dates {
					artist.Concerts = append(artist.Concerts, Concert{
						Date:     date,
						Location: normalizedLoc,
					})
				}

				// Extract country from location
				countries[s.extractCountryFromLocation(normalizedLoc)] = true
			}

			// Sort concerts chronologically
			sort.Slice(artist.Concerts, func(i, j int) bool {
				return artist.Concerts[i].Date < artist.Concerts[j].Date
			})

			// Set derived fields
			artist.ConcertCount = len(artist.Concerts)
			artist.Countries = s.convertCountriesMapToSlice(countries)
		}
	}

	// Sort artists by name for consistent display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

// buildIndexes creates fast lookup maps and stores the processed artists.
func (s *Store) buildIndexes(artists []Artist) {
	s.artists = artists
	s.artistsByID = make(map[int]Artist, len(artists))
	s.artistsBySlug = make(map[string]Artist, len(artists))

	for _, artist := range artists {
		s.artistsByID[artist.ID] = artist
		s.artistsBySlug[artist.Slug] = artist
	}
}

// buildLocations creates location aggregates from artist concert data.
func (s *Store) buildLocations() {
	locations := s.createLocations(s.artists)
	s.locations = locations
	s.locationsBySlug = make(map[string]Location, len(locations))
	for _, location := range locations {
		s.locationsBySlug[location.Slug] = location
	}
}

// createLocations builds location models from artist concert data.
func (s *Store) createLocations(artists []Artist) []Location {
	// Build lookup map once - O(n) instead of O(n²)
	artistMap := make(map[int]Artist, len(artists))
	for _, artist := range artists {
		artistMap[artist.ID] = artist
	}

	locationMap := make(map[string]*Location)
	// Track concert count per artist per location
	artistConcertCount := make(map[string]map[int]int)

	for i := range artists {
		artist := &artists[i]
		for _, concert := range artist.Concerts {
			// Initialize location if not exists
			if _, exists := locationMap[concert.Location]; !exists {
				locationMap[concert.Location] = &Location{
					Name:         concert.Location,
					Slug:         createSlug(concert.Location),
					Artists:      make([]ArtistAtLocation, 0),
					EarliestYear: 9999, // Initialize with high value
					LatestYear:   0,    // Initialize with low value
				}
				artistConcertCount[concert.Location] = make(map[int]int)
			}

			// Count concerts per artist per location
			artistConcertCount[concert.Location][artist.ID]++
			locationMap[concert.Location].TotalConcerts++

			// Update year range for this location
			year := s.extractYearFromDate(concert.Date)
			if year > 0 {
				if year < locationMap[concert.Location].EarliestYear {
					locationMap[concert.Location].EarliestYear = year
				}
				if year > locationMap[concert.Location].LatestYear {
					locationMap[concert.Location].LatestYear = year
				}
			}
		}
	}

	// Convert concert count map to ArtistAtLocation structs
	for locationName, location := range locationMap {
		artistCounts := artistConcertCount[locationName]
		artistsAtLocation := make([]ArtistAtLocation, 0, len(artistCounts))

		for artistID, concertCount := range artistCounts {
			// Use O(1) map lookup instead of O(n) linear search
			if artist, found := artistMap[artistID]; found {
				artistsAtLocation = append(artistsAtLocation, ArtistAtLocation{
					Artist:       artist,
					ConcertCount: concertCount,
				})
			}
		}

		// Sort artists by concert count (descending), then by name
		sort.Slice(artistsAtLocation, func(i, j int) bool {
			if artistsAtLocation[i].ConcertCount != artistsAtLocation[j].ConcertCount {
				return artistsAtLocation[i].ConcertCount > artistsAtLocation[j].ConcertCount
			}
			return artistsAtLocation[i].Artist.Name < artistsAtLocation[j].Artist.Name
		})

		location.Artists = artistsAtLocation
		location.ArtistCount = len(artistsAtLocation)
	}

	// Convert to slice and sort by concert count
	locations := make([]Location, 0, len(locationMap))
	for _, loc := range locationMap {
		locations = append(locations, *loc)
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].TotalConcerts > locations[j].TotalConcerts
	})

	return locations
}

// computeStats calculates application statistics from loaded data.
func (s *Store) computeStats() {
	totalMembers := 0
	totalConcerts := 0
	countries := make(map[string]bool)

	for _, artist := range s.artists {
		totalMembers += len(artist.Members)
		totalConcerts += artist.ConcertCount

		for _, country := range artist.Countries {
			countries[country] = true
		}
	}

	s.appStats = AppStats{
		TotalArtists:   len(s.artists),
		TotalMembers:   totalMembers,
		TotalLocations: len(s.locations),
		TotalConcerts:  totalConcerts,
		TotalCountries: len(countries),
		// Cache stats will be set by cacheImages if enabled
	}
}

// cacheImages downloads and caches artist images locally when caching is enabled.
// Returns true if caching is successfully enabled.
// Uses a worker pool for concurrent downloads.
func (s *Store) cacheImages(artists []Artist) bool {
	if !s.withCache {
		return false
	}

	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return false
	}

	// Use worker pool for concurrent downloads
	numWorkers := 4 // Fixed number of workers for consistent behavior

	// Job represents a download task
	type job struct {
		index     int
		artist    *Artist
		fileName  string
		filePath  string
		localPath string
		exists    bool
	}

	// Create job queue
	jobs := make(chan job, len(artists))

	// Prepare all jobs
	for i := range artists {
		artist := &artists[i]
		fileName := fmt.Sprintf("%s.jpg", artist.Slug)
		filePath := filepath.Join(cacheDir, fileName)
		localPath := "/" + filepath.ToSlash(filePath)
		exists := false

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			exists = true
		}

		jobs <- job{
			index:     i,
			artist:    artist,
			fileName:  fileName,
			filePath:  filePath,
			localPath: localPath,
			exists:    exists,
		}
	}
	close(jobs)

	// Atomic counters for thread-safe counting
	var cached, downloaded int32
	var mu sync.Mutex // Mutex for updating artist images

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if j.exists {
					// Use cached file
					mu.Lock()
					j.artist.Image = j.localPath
					mu.Unlock()
					cached++
				} else {
					// Download image
					if downloadImage(j.artist.Image, j.filePath) {
						mu.Lock()
						j.artist.Image = j.localPath
						mu.Unlock()
						downloaded++
					}
				}
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()

	// Update stats with cache information
	s.appStats.CachedImages = int(cached)
	s.appStats.DownloadedImages = int(downloaded)

	return true
}

// Helper functions

// downloadImage downloads and saves a single image from a URL to local filesystem.
func downloadImage(url, path string) bool {
	if strings.TrimSpace(url) == "" {
		return false
	}

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return false
	}
	defer resp.Body.Close()

	file, err := os.Create(path)
	if err != nil {
		return false
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err == nil
}

// createSlug converts display names into URL-friendly slugs.
func createSlug(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

// normalizeLocation converts raw API location strings to consistent format.
func normalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), "_", "-"))
}

// convertCountriesMapToSlice converts a map[string]bool to sorted string slice.
func (s *Store) convertCountriesMapToSlice(countriesMap map[string]bool) []string {
	countries := make([]string, 0, len(countriesMap))
	for country := range countriesMap {
		if country != "" { // Skip empty countries
			countries = append(countries, country)
		}
	}
	sort.Strings(countries)
	return countries
}

// extractCountryFromLocation extracts the country name from a location string.
func (s *Store) extractCountryFromLocation(location string) string {
	parts := strings.Split(location, "-")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// extractYearFromDate parses a date string and extracts the year.
func (s *Store) extractYearFromDate(date string) int {
	parts := strings.Split(date, "-")
	if len(parts) >= 3 {
		var year int
		fmt.Sscanf(parts[2], "%d", &year)
		return year
	}
	return 0
}
