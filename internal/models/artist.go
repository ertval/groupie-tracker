// Package models contains the core domain models for the Groupie Tracker application.
// This package defines the data structures used throughout the application without
// any business logic or external dependencies.
package models

import "strings"

// Concert represents a single concert performance by an artist.
// This is the basic unit of performance data with location and temporal information.
type Concert struct {
	// Location is the formatted venue/city name (e.g., "New York, NY")
	Location string `json:"location"`

	// Country is the extracted country name from the location
	Country string `json:"country"`

	// Date is the original date string from the API (e.g., "25-12-2019")
	Date string `json:"date"`

	// Year is the extracted year from the date for easier filtering and aggregation
	Year int `json:"year"`
}

// Artist represents the complete internal model of a music artist/band.
// This aggregates data from multiple API endpoints and includes computed fields.
type Artist struct {
	// ID is the unique identifier from the external API
	ID int `json:"id"`

	// Name is the display name of the artist/band
	Name string `json:"name"`

	// Slug is a URL-friendly version of the name for routing
	Slug string `json:"slug"`

	// Members contains all band member names
	Members []string `json:"members"`

	// CreationYear is when the artist/band was formed
	CreationYear int `json:"creation_year"`

	// FirstAlbum is the date of their first album release
	FirstAlbum string `json:"first_album"`

	// Image is the URL to the artist's image
	Image string `json:"image"`

	// Concerts contains all concert performances for this artist
	Concerts []Concert `json:"concerts"`

	// Countries is a computed list of unique countries where they've performed
	Countries []string `json:"countries"`

	// ConcertCount is the total number of concerts (computed field)
	ConcertCount int `json:"concert_count"`
}

// GetMemberCount returns the number of members in the artist/band.
// This is a convenience method to avoid accessing len(Members) directly.
func (a Artist) GetMemberCount() int {
	return len(a.Members)
}

// HasPerformedInCountry checks if the artist has performed in the specified country.
// The comparison is case-insensitive.
func (a Artist) HasPerformedInCountry(country string) bool {
	for _, c := range a.Countries {
		if strings.EqualFold(c, country) {
			return true
		}
	}
	return false
}
