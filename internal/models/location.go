package models

import (
	"fmt"
	"strings"
)

// Location represents a concert venue with aggregated statistics.
// This is computed from all artist concert data and provides venue-centric views.
type Location struct {
	// Name is the formatted location name (e.g., "New York, NY")
	Name string `json:"name"`

	// Slug is a URL-friendly version of the name for routing
	Slug string `json:"slug"`

	// Artists contains names of all artists who performed at this location
	Artists []string `json:"artists"`

	// ArtistCount is the number of unique artists who performed here
	ArtistCount int `json:"artist_count"`

	// TotalConcerts is the total number of concerts held at this location
	TotalConcerts int `json:"total_concerts"`

	// EarliestYear is the year of the first concert at this location
	EarliestYear int `json:"earliest_year"`

	// LatestYear is the year of the most recent concert at this location
	LatestYear int `json:"latest_year"`

	// Concerts contains all concert performances at this location
	Concerts []Concert `json:"concerts"`
}

// GetCountry extracts the country from the location name.
// Assumes format "City, Country" and returns the part after the last comma.
func (l Location) GetCountry() string {
	parts := strings.Split(l.Name, ", ")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return l.Name
}

// GetYearRange returns a formatted string showing the active years for this location.
// Returns "YYYY" for single year, "YYYY-YYYY" for range.
func (l Location) GetYearRange() string {
	if l.EarliestYear == l.LatestYear {
		return fmt.Sprintf("%d", l.EarliestYear)
	}
	return fmt.Sprintf("%d-%d", l.EarliestYear, l.LatestYear)
}
