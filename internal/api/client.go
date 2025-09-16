// Package api provides functionality to fetch data from the Groupie Tracker API.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// -----------------------------
// API Data Models (1:1 with External API)
// -----------------------------

// Artist represents a musical artist or band as returned by the external API.
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

// Relation represents the relationship between artists with their concert locations and dates from the API.
type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// Location represents a location data structure from the API
type Location struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
}

// Date represents a date data structure from the API
type Date struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

// APIResponse represents the main API response structure.
type APIResponse struct {
	Artists   []Artist   `json:"artists,omitempty"`
	Locations []Location `json:"locations,omitempty"`
	Dates     []Date     `json:"dates,omitempty"`
	Relations []Relation `json:"relations,omitempty"`
}

// -----------------------------
// API Validation Methods
// -----------------------------

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

// GetFirstAlbumDate parses the FirstAlbum string and returns a time.Time.
// Expected format is "DD-MM-YYYY".
func (a *Artist) GetFirstAlbumDate() (time.Time, error) {
	if a.FirstAlbum == "" {
		return time.Time{}, errors.New("first album date is empty")
	}

	// Parse the date in DD-MM-YYYY format
	parsedTime, err := time.Parse("02-01-2006", a.FirstAlbum)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format '%s': expected DD-MM-YYYY", a.FirstAlbum)
	}

	return parsedTime, nil
}

// -----------------------------
// HTTP Client
// -----------------------------

// Client represents an HTTP client for the Groupie Tracker API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client with the specified base URL and timeout.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchArtists retrieves all artists from the API.
func (c *Client) FetchArtists(ctx context.Context) ([]Artist, error) {
	url := c.baseURL + "/api/artists"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var artists []Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return artists, nil
}

// FetchLocations retrieves all locations from the API.
func (c *Client) FetchLocations(ctx context.Context) ([]Location, error) {
	url := c.baseURL + "/api/locations"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var response struct {
		Index []Location `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchDates retrieves all dates from the API.
func (c *Client) FetchDates(ctx context.Context) ([]Date, error) {
	url := c.baseURL + "/api/dates"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var response struct {
		Index []Date `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchRelations retrieves all relations from the API.
func (c *Client) FetchRelations(ctx context.Context) ([]Relation, error) {
	url := c.baseURL + "/api/relation"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var response struct {
		Index []Relation `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchAllData retrieves all data from the API endpoints.
func (c *Client) FetchAllData(ctx context.Context) (*APIResponse, error) {
	response := &APIResponse{}

	// Fetch artists
	artists, err := c.FetchArtists(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching artists: %w", err)
	}
	response.Artists = artists

	// Fetch locations
	locations, err := c.FetchLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching locations: %w", err)
	}
	response.Locations = locations

	// Fetch dates
	dates, err := c.FetchDates(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching dates: %w", err)
	}
	response.Dates = dates

	// Fetch relations
	relations, err := c.FetchRelations(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching relations: %w", err)
	}
	response.Relations = relations

	return response, nil
}
