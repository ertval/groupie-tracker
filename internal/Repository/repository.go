// Package repository provides the core data management functionality for the Groupie Tracker application.
// This package follows idiomatic Go patterns with clear separation between API responses,
// domain models, and the repository that manages all data operations.
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

// API Response Structs - Direct mappings from the 4 API endpoints

// ArtistAPIResponse represents the response from /api/artists
type ArtistAPIResponse struct {
ID            int      `json:"id"`
Name          string   `json:"name"`
Members       []string `json:"members"`
CreationYear  int      `json:"creationDate"`
FirstAlbum    string   `json:"firstAlbum"`
Image         string   `json:"image"`
LocationsURL  string   `json:"locations"`
DatesURL      string   `json:"concertDates"`
RelationsURL  string   `json:"relations"`
}

// LocationAPIResponse represents the response from /api/locations
type LocationAPIResponse struct {
Index []struct {
ID        int      `json:"id"`
Locations []string `json:"locations"`
DatesURL  string   `json:"dates"`
} `json:"index"`
}

// DateAPIResponse represents the response from /api/dates  
type DateAPIResponse struct {
Index []struct {
ID    int      `json:"id"`
Dates []string `json:"dates"`
} `json:"index"`
}

// RelationAPIResponse represents the response from /api/relation
type RelationAPIResponse struct {
Index []struct {
ID              int                 `json:"id"`
DatesLocations  map[string][]string `json:"datesLocations"`
} `json:"index"`
}

// Domain Models - Processed data structures for business logic

// Artist represents a musical artist with computed SEO slug.
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
ID             int                 `json:"id"`
DatesLocations map[string][]string `json:"datesLocations"`
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
// ConcertDates maps artist ID to their concert dates at this location
ConcertDates map[int][]string
}

// ComputedData holds all processed and computed data needed for templates.
type ComputedData struct {
artists       map[int]Artist
concerts      map[int]Concert
slugToID      map[string]int
locationStats map[string]*LocationStats
globalStats   map[string]int
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
artists:       make(map[int]Artist),
concerts:      make(map[int]Concert),
slugToID:      make(map[string]int),
locationStats: make(map[string]*LocationStats),
globalStats:   make(map[string]int),
},
}
}

// LoadData fetches and processes all data from the API endpoints.
func (r *Repository) LoadData(ctx context.Context) error {
// Fetch from all 4 API endpoints
var artistsResp []ArtistAPIResponse
if err := r.fetchJSON(ctx, "/api/artists", &artistsResp); err != nil {
return fmt.Errorf("failed to fetch artists: %w", err)
}

var relationsResp RelationAPIResponse
if err := r.fetchJSON(ctx, "/api/relation", &relationsResp); err != nil {
return fmt.Errorf("failed to fetch relations: %w", err)
}

// Process and compute data
r.processArtists(artistsResp)
r.processRelations(relationsResp.Index)
r.computeLocationStats()
r.computeGlobalStats()

return nil
}

// GetArtists returns all artists sorted by name.
func (r *Repository) GetArtists() []Artist {
artists := make([]Artist, 0, len(r.data.artists))
for _, artist := range r.data.artists {
artists = append(artists, artist)
}
sort.Slice(artists, func(i, j int) bool {
return artists[i].Name < artists[j].Name
})
return artists
}

// GetArtist returns an artist by ID.
func (r *Repository) GetArtist(id int) (Artist, bool) {
artist, exists := r.data.artists[id]
return artist, exists
}

// GetArtistBySlug returns an artist by SEO slug.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
id, exists := r.data.slugToID[slug]
if !exists {
return Artist{}, false
}
return r.GetArtist(id)
}

// GetConcert returns concert data for an artist.
func (r *Repository) GetConcert(artistID int) (Concert, bool) {
concert, exists := r.data.concerts[artistID]
return concert, exists
}

// GetLocationStats returns statistics for all locations sorted by total shows.
func (r *Repository) GetLocationStats() []LocationStats {
stats := make([]LocationStats, 0, len(r.data.locationStats))
for _, stat := range r.data.locationStats {
stats = append(stats, *stat)
}
sort.Slice(stats, func(i, j int) bool {
return stats[i].TotalShows > stats[j].TotalShows
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

// GetStats returns computed global statistics.
func (r *Repository) GetStats() map[string]int {
return r.data.globalStats
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

// CountShows returns the total number of shows for a concert.
func (r *Repository) CountShows(concert Concert) int {
total := 0
for _, dates := range concert.DatesLocations {
total += len(dates)
}
return total
}

// GetCountries extracts unique countries from concert locations.
func (r *Repository) GetCountries(concert Concert) []string {
countryMap := make(map[string]bool)
for location := range concert.DatesLocations {
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

func (r *Repository) processArtists(apiArtists []ArtistAPIResponse) {
for _, apiArtist := range apiArtists {
artist := Artist{
ID:           apiArtist.ID,
Name:         apiArtist.Name,
Members:      apiArtist.Members,
CreationYear: apiArtist.CreationYear,
FirstAlbum:   apiArtist.FirstAlbum,
Image:        apiArtist.Image,
Slug:         createSlug(apiArtist.Name),
}
r.data.artists[artist.ID] = artist
r.data.slugToID[artist.Slug] = artist.ID
}
}

func (r *Repository) processRelations(apiRelations []struct {
ID              int                 `json:"id"`
DatesLocations  map[string][]string `json:"datesLocations"`
}) {
for _, apiRelation := range apiRelations {
concert := Concert{
ID:             apiRelation.ID,
DatesLocations: apiRelation.DatesLocations,
}
r.data.concerts[concert.ID] = concert
}
}

func (r *Repository) computeLocationStats() {
// Process all concerts to build location stats
for _, concert := range r.data.concerts {
artist, artistExists := r.data.artists[concert.ID]
if !artistExists {
continue
}

for location, dates := range concert.DatesLocations {
normalizedLocation := normalizeLocation(location)

// Get or create location stats
locationStat, exists := r.data.locationStats[normalizedLocation]
if !exists {
locationStat = &LocationStats{
Name:         normalizedLocation,
DisplayName:  location,
Slug:         createSlug(normalizedLocation),
Artists:      make([]Artist, 0),
ConcertDates: make(map[int][]string),
}
r.data.locationStats[normalizedLocation] = locationStat
}

// Add artist if not already present
if !containsArtist(locationStat.Artists, artist) {
locationStat.Artists = append(locationStat.Artists, artist)
}

// Store concert dates for this artist at this location
locationStat.ConcertDates[artist.ID] = dates

// Update counters
locationStat.ArtistCount = len(locationStat.Artists)
locationStat.TotalShows += len(dates)
locationStat.ConcertCount = locationStat.TotalShows
}
}
}

func (r *Repository) computeGlobalStats() {
totalMembers := 0
totalShows := 0
countrySet := make(map[string]bool)

for _, artist := range r.data.artists {
totalMembers += len(artist.Members)
}

for _, concert := range r.data.concerts {
for location, dates := range concert.DatesLocations {
totalShows += len(dates)

// Extract country
parts := strings.Split(location, "-")
if len(parts) > 1 {
country := strings.TrimSpace(parts[len(parts)-1])
countrySet[country] = true
}
}
}

r.data.globalStats["total_artists"] = len(r.data.artists)
r.data.globalStats["total_members"] = totalMembers
r.data.globalStats["total_locations"] = len(r.data.locationStats)
r.data.globalStats["total_shows"] = totalShows
r.data.globalStats["total_countries"] = len(countrySet)
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
