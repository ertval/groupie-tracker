package data

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

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
			MemberCount:     len(apiArtist.Members),
			FirstAlbumYear:  extractYearFromDate(apiArtist.FirstAlbum),
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
				countries[extractCountryFromLocation(normalizedLoc)] = true
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

// createArtistIndexes builds lookup maps for artists by ID and slug.
func (s *Store) createArtistIndexes(artists []Artist) (map[int]Artist, map[string]Artist, map[int]int) {
	artistsByID := make(map[int]Artist, len(artists))
	artistsBySlug := make(map[string]Artist, len(artists))
	positions := make(map[int]int, len(artists))

	for idx, artist := range artists {
		artistsByID[artist.ID] = artist
		artistsBySlug[artist.Slug] = artist
		positions[artist.ID] = idx
	}

	return artistsByID, artistsBySlug, positions
}

// createLocationsData builds location aggregates and lookup maps.
func (s *Store) createLocationsData(artists []Artist) ([]Location, map[string]Location) {
	locations := s.createLocations(artists)
	locationsBySlug := make(map[string]Location, len(locations))
	for _, location := range locations {
		locationsBySlug[location.Slug] = location
	}
	return locations, locationsBySlug
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
					Country:      extractCountryFromLocation(concert.Location),
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
			year := extractYearFromDate(concert.Date)
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

// calculateArtistFilterOptions derives available artist filter metadata from the dataset.
func (s *Store) calculateArtistFilterOptions(artists []Artist) ArtistFilterOptions {
	if len(artists) == 0 {
		return ArtistFilterOptions{}
	}

	minCreationYear, maxCreationYear := artists[0].CreationYear, artists[0].CreationYear
	minFirstAlbumYear, maxFirstAlbumYear := 0, 0
	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	for _, artist := range artists {
		if artist.CreationYear < minCreationYear {
			minCreationYear = artist.CreationYear
		}
		if artist.CreationYear > maxCreationYear {
			maxCreationYear = artist.CreationYear
		}

		albumYear := artist.FirstAlbumYear
		if albumYear == 0 {
			albumYear = extractYearFromDate(artist.FirstAlbum)
		}
		if albumYear > 0 {
			if minFirstAlbumYear == 0 || albumYear < minFirstAlbumYear {
				minFirstAlbumYear = albumYear
			}
			if albumYear > maxFirstAlbumYear {
				maxFirstAlbumYear = albumYear
			}
		}

		memberCount := artist.MemberCount
		if memberCount == 0 {
			memberCount = len(artist.Members)
		}
		memberCountSet[memberCount] = true

		for _, country := range artist.Countries {
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	memberCounts := make([]int, 0, len(memberCountSet))
	for count := range memberCountSet {
		memberCounts = append(memberCounts, count)
	}
	sort.Ints(memberCounts)

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	if minFirstAlbumYear == 0 {
		minFirstAlbumYear = minCreationYear
	}
	if maxFirstAlbumYear == 0 {
		maxFirstAlbumYear = maxCreationYear
	}

	return ArtistFilterOptions{
		CreationYearMin:   minCreationYear,
		CreationYearMax:   maxCreationYear,
		FirstAlbumYearMin: minFirstAlbumYear,
		FirstAlbumYearMax: maxFirstAlbumYear,
		MemberCounts:      memberCounts,
		Countries:         countries,
	}
}

// calculateLocationFilterOptions derives available location filter metadata.
func (s *Store) calculateLocationFilterOptions(locations []Location) LocationFilterOptions {
	if len(locations) == 0 {
		return LocationFilterOptions{}
	}

	minConcerts, maxConcerts := locations[0].TotalConcerts, locations[0].TotalConcerts
	minArtists, maxArtists := locations[0].ArtistCount, locations[0].ArtistCount
	minYear, maxYear := locations[0].EarliestYear, locations[0].LatestYear
	countrySet := make(map[string]bool)

	for _, location := range locations {
		if location.TotalConcerts < minConcerts {
			minConcerts = location.TotalConcerts
		}
		if location.TotalConcerts > maxConcerts {
			maxConcerts = location.TotalConcerts
		}

		if location.ArtistCount < minArtists {
			minArtists = location.ArtistCount
		}
		if location.ArtistCount > maxArtists {
			maxArtists = location.ArtistCount
		}

		if location.EarliestYear > 0 && location.EarliestYear < minYear {
			minYear = location.EarliestYear
		}
		if location.LatestYear > maxYear {
			maxYear = location.LatestYear
		}

		country := location.Country
		if country == "" {
			country = extractCountryFromLocation(location.Name)
		}
		if country != "" {
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return LocationFilterOptions{
		ConcertCountMin: minConcerts,
		ConcertCountMax: maxConcerts,
		ArtistCountMin:  minArtists,
		ArtistCountMax:  maxArtists,
		ConcertYearMin:  minYear,
		ConcertYearMax:  maxYear,
		Countries:       countries,
	}
}

// generateSearchSuggestions pre-computes autocomplete suggestions from the dataset.
func (s *Store) generateSearchSuggestions(artists []Artist) []SearchSuggestion {
	var suggestions []SearchSuggestion
	seen := make(map[string]bool)

	for _, artist := range artists {
		artistKey := "artist:" + artist.Name
		if !seen[artistKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.Name+" - artist",
				string(SuggestionTypeArtist),
				artist.Name+" - artist",
				"/artists/"+artist.Slug,
				artist.ID,
			))
			seen[artistKey] = true
		}

		for _, member := range artist.Members {
			memberKey := "member:" + member
			if !seen[memberKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					member+" - member",
					string(SuggestionTypeMember),
					member+" - member of "+artist.Name,
					"/artists/"+artist.Slug,
					artist.ID,
				))
				seen[memberKey] = true
			}
		}

		for location := range artist.DatesAtLocation {
			locationKey := "location:" + location
			if !seen[locationKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					location+" - location",
					string(SuggestionTypeLocation),
					location+" - concert location",
					"/search?q="+location,
					0,
				))
				seen[locationKey] = true
			}
		}

		creationYear := strconv.Itoa(artist.CreationYear)
		yearKey := "creation:" + creationYear
		if !seen[yearKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				creationYear+" - creation year",
				string(SuggestionTypeCreation),
				"Artists created in "+creationYear,
				"/search?q="+creationYear,
				0,
			))
			seen[yearKey] = true
		}

		albumKey := "album:" + artist.FirstAlbum
		if !seen[albumKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.FirstAlbum+" - first album",
				string(SuggestionTypeFirstAlbum),
				"Albums released on "+artist.FirstAlbum,
				"/search?q="+artist.FirstAlbum,
				0,
			))
			seen[albumKey] = true
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

// calculateStats computes application statistics from derived data.
func (s *Store) calculateStats(artists []Artist, locations []Location, cachedImages, downloadedImages int) AppStats {
	totalMembers := 0
	totalConcerts := 0
	countries := make(map[string]bool)

	for _, artist := range artists {
		totalMembers += len(artist.Members)
		totalConcerts += artist.ConcertCount

		for _, country := range artist.Countries {
			countries[country] = true
		}
	}

	return AppStats{
		TotalArtists:     len(artists),
		TotalMembers:     totalMembers,
		TotalLocations:   len(locations),
		TotalConcerts:    totalConcerts,
		TotalCountries:   len(countries),
		CachedImages:     cachedImages,
		DownloadedImages: downloadedImages,
	}
}

// cacheImages downloads and caches artist images locally when caching is enabled.
// Returns whether caching was enabled along with cached/downloaded image counts.
func (s *Store) cacheImages(artists []Artist) (bool, int, int) {
	if !s.withCache {
		return false, 0, 0
	}

	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return false, 0, 0
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
					atomic.AddInt32(&cached, 1)
				} else {
					// Download image
					if downloadImage(j.artist.Image, j.filePath) {
						mu.Lock()
						j.artist.Image = j.localPath
						mu.Unlock()
						atomic.AddInt32(&downloaded, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	return true, int(atomic.LoadInt32(&cached)), int(atomic.LoadInt32(&downloaded))
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
