package models

import "fmt"

// AppStats provides type-safe application statistics for monitoring and display.
type AppStats struct {
	// TotalArtists is the number of artists in the system
	TotalArtists int `json:"total_artists"`

	// TotalLocations is the number of unique performance locations
	TotalLocations int `json:"total_locations"`

	// TotalConcerts is the total number of concerts across all artists
	TotalConcerts int `json:"total_concerts"`

	// EarliestYear is the year of the oldest concert in the dataset
	EarliestYear int `json:"earliest_year"`

	// LatestYear is the year of the most recent concert in the dataset
	LatestYear int `json:"latest_year"`

	// CachedImages tracks the number of locally cached artist images
	CachedImages int `json:"cached_images"`

	// DownloadedImages tracks the number of images downloaded during startup
	DownloadedImages int `json:"downloaded_images"`
}

// GetYearRange returns a formatted string of the data's year coverage.
func (s AppStats) GetYearRange() string {
	if s.EarliestYear == s.LatestYear {
		return fmt.Sprintf("%d", s.EarliestYear)
	}
	return fmt.Sprintf("%d-%d", s.EarliestYear, s.LatestYear)
}

// GetAverageeConcertsPerArtist calculates the average number of concerts per artist.
func (s AppStats) GetAverageConcertsPerArtist() float64 {
	if s.TotalArtists == 0 {
		return 0
	}
	return float64(s.TotalConcerts) / float64(s.TotalArtists)
}

// GetAverageConcertsPerLocation calculates the average number of concerts per location.
func (s AppStats) GetAverageConcertsPerLocation() float64 {
	if s.TotalLocations == 0 {
		return 0
	}
	return float64(s.TotalConcerts) / float64(s.TotalLocations)
}
