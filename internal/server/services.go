package server

import (
	"groupie-tracker/internal/data"
)

// Service interfaces following Interface Segregation Principle
// Each interface has a focused responsibility instead of one monolithic repository

// ArtistService handles all artist-related operations
type ArtistService interface {
	GetArtists() []data.Artist
	GetArtistByID(id int) (data.Artist, bool)
	GetArtistBySlug(slug string) (data.Artist, bool)
	GetAdjacentArtists(currentID int) (prev, next *data.Artist)
	FilterArtists(params data.ArtistFilterParams) []data.Artist
	GetArtistFilterOptions() data.ArtistFilterOptions
}

// SearchService handles all search-related operations
type SearchService interface {
	SearchArtists(params data.SearchParams) data.SearchResult
	GenerateAllSearchSuggestions() []data.SearchSuggestion
}

// LocationService handles all location-related operations
type LocationService interface {
	GetLocations() []data.Location
	GetLocationBySlug(slug string) (data.Location, bool)
}

// StatsService handles statistical data operations
type StatsService interface {
	GetStats() map[string]int
}

// CacheService handles image caching operations
type CacheService interface {
	IsCacheEnabled() bool
}

// Service implementations that wrap the repository

// artistService implements ArtistService using the repository
type artistService struct {
	repo *data.Repository
}

func (a *artistService) GetArtists() []data.Artist {
	return a.repo.GetArtists()
}

func (a *artistService) GetArtistByID(id int) (data.Artist, bool) {
	return a.repo.GetArtistByID(id)
}

func (a *artistService) GetArtistBySlug(slug string) (data.Artist, bool) {
	return a.repo.GetArtistBySlug(slug)
}

func (a *artistService) GetAdjacentArtists(currentID int) (prev, next *data.Artist) {
	return a.repo.GetAdjacentArtists(currentID)
}

func (a *artistService) FilterArtists(params data.ArtistFilterParams) []data.Artist {
	return a.repo.FilterArtists(params)
}

func (a *artistService) GetArtistFilterOptions() data.ArtistFilterOptions {
	return a.repo.GetArtistFilterOptions()
}

// searchService implements SearchService using the repository
type searchService struct {
	repo *data.Repository
}

func (s *searchService) SearchArtists(params data.SearchParams) data.SearchResult {
	return s.repo.SearchArtists(params)
}

func (s *searchService) GenerateAllSearchSuggestions() []data.SearchSuggestion {
	return s.repo.GenerateAllSearchSuggestions()
}

// locationService implements LocationService using the repository
type locationService struct {
	repo *data.Repository
}

func (l *locationService) GetLocations() []data.Location {
	return l.repo.GetLocations()
}

func (l *locationService) GetLocationBySlug(slug string) (data.Location, bool) {
	return l.repo.GetLocationBySlug(slug)
}

// statsService implements StatsService using the repository
type statsService struct {
	repo *data.Repository
}

func (s *statsService) GetStats() map[string]int {
	return s.repo.GetStats()
}

// cacheService implements CacheService using the repository
type cacheService struct {
	repo *data.Repository
}

func (c *cacheService) IsCacheEnabled() bool {
	return c.repo.IsCacheEnabled()
}

// Service factory functions

func newArtistService(repo *data.Repository) ArtistService {
	return &artistService{repo: repo}
}

func newSearchService(repo *data.Repository) SearchService {
	return &searchService{repo: repo}
}

func newLocationService(repo *data.Repository) LocationService {
	return &locationService{repo: repo}
}

func newStatsService(repo *data.Repository) StatsService {
	return &statsService{repo: repo}
}

func newCacheService(repo *data.Repository) CacheService {
	return &cacheService{repo: repo}
}
