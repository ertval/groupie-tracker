// Package models defines the data structures for the Groupie Tracker application.
// It contains the core entities: Artist, Location, Date, and Relation.
package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Artist represents a musical artist or band with all their information.
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Slug         string   `json:"slug,omitempty"`
}

// Location represents concert locations for an artist.
type Location struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
	Dates     string   `json:"dates"`
}

// Date represents concert dates for an artist.
type Date struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

// Relation links artists with their concert locations and dates.
type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// APIResponse represents the main API response structure.
type APIResponse struct {
	Artists   []Artist   `json:"artists,omitempty"`
	Locations []Location `json:"locations,omitempty"`
	Dates     []Date     `json:"dates,omitempty"`
	Relations []Relation `json:"relations,omitempty"`
}

// Validate checks if the Artist struct has valid data.
func (a *Artist) Validate() error {
	if a.Name == "" {
		return errors.New("artist name cannot be empty")
	}

	if a.CreationYear <= 0 {
		return errors.New("creation year must be greater than 0")
	}

	if len(a.Members) == 0 {
		return errors.New("artist must have at least one member")
	}

	return nil
}

// GetFirstAlbumDate parses the FirstAlbum string and returns a time.Time.
// Expected format is "DD-MM-YYYY".
func (a *Artist) GetFirstAlbumDate() (time.Time, error) {
	if a.FirstAlbum == "" {
		return time.Time{}, errors.New("first album date is empty")
	}

	// Parse the date in DD-MM-YYYY format
	parsedTime, err := time.Parse("02-01-2006", a.FirstAlbum)
	if err != nil {
		return time.Time{}, errors.New("invalid date format, expected DD-MM-YYYY")
	}

	return parsedTime, nil
}

// GenerateSlug creates a URL-friendly slug from the artist name.
func (a *Artist) GenerateSlug() string {
	if a.Name == "" {
		return ""
	}

	// Convert to lowercase
	slug := strings.ToLower(a.Name)

	// Replace spaces and special characters with hyphens
	// Keep only alphanumeric characters and hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	return slug
}

// SetSlug generates and sets the slug for the artist.
func (a *Artist) SetSlug() {
	a.Slug = a.GenerateSlug()
}

// GetSlug returns the artist's slug, generating it if it doesn't exist.
func (a *Artist) GetSlug() string {
	if a.Slug == "" {
		a.Slug = a.GenerateSlug()
	}
	return a.Slug
}

// Validate checks if the Location struct has valid data.
func (l *Location) Validate() error {
	if l.ID <= 0 {
		return errors.New("location ID must be greater than 0")
	}

	if len(l.Locations) == 0 {
		return errors.New("location must have at least one location")
	}

	return nil
}

// Validate checks if the Date struct has valid data.
func (d *Date) Validate() error {
	if d.ID <= 0 {
		return errors.New("date ID must be greater than 0")
	}

	if len(d.Dates) == 0 {
		return errors.New("date must have at least one date")
	}

	return nil
}

// Validate checks if the Relation struct has valid data.
func (r *Relation) Validate() error {
	if r.ID <= 0 {
		return errors.New("relation ID must be greater than 0")
	}

	if len(r.DatesLocations) == 0 {
		return errors.New("relation must have at least one dates-location mapping")
	}

	return nil
}

// GenerateLocationSlug creates a URL-friendly slug from a location name.
// Location names come in formats like "new_york-usa" or "paris-france"
func GenerateLocationSlug(locationName string) string {
	if locationName == "" {
		return ""
	}

	// Convert to lowercase
	slug := strings.ToLower(locationName)

	// Replace underscores with hyphens for consistency
	slug = strings.ReplaceAll(slug, "_", "-")

	// Keep only alphanumeric characters and hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	return slug
}

// NormalizeLocationName formats location names for display
// Converts "new_york-usa" to "New York, USA"
func NormalizeLocationName(locationName string) string {
	if locationName == "" {
		return ""
	}

	// Split by the last hyphen to separate city from country
	parts := strings.Split(locationName, "-")
	if len(parts) < 2 {
		return toTitleCase(strings.ReplaceAll(locationName, "_", " "))
	}

	// Get city and country parts
	countryIndex := len(parts) - 1
	country := parts[countryIndex]
	city := strings.Join(parts[:countryIndex], "-")

	// Format city: replace underscores with spaces and title case
	formattedCity := toTitleCase(strings.ReplaceAll(city, "_", " "))

	// Format country: uppercase
	formattedCountry := strings.ToUpper(country)

	return fmt.Sprintf("%s, %s", formattedCity, formattedCountry)
}

// toTitleCase converts a string to title case (first letter of each word capitalized)
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
