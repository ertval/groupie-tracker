// Package api provides functionality to fetch data from the Groupie Tracker API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"groupie-tracker/internal/models"
)

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
func (c *Client) FetchArtists(ctx context.Context) ([]models.Artist, error) {
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

	var artists []models.Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return artists, nil
}

// FetchLocations retrieves all locations from the API.
func (c *Client) FetchLocations(ctx context.Context) ([]models.Location, error) {
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
		Index []models.Location `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchDates retrieves all dates from the API.
func (c *Client) FetchDates(ctx context.Context) ([]models.Date, error) {
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
		Index []models.Date `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchRelations retrieves all relations from the API.
func (c *Client) FetchRelations(ctx context.Context) ([]models.Relation, error) {
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
		Index []models.Relation `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// AllData represents the complete dataset from the API.
type AllData struct {
	Artists   []models.Artist
	Locations []models.Location
	Dates     []models.Date
	Relations []models.Relation
}

// FetchAllData retrieves all data from the API endpoints.
func (c *Client) FetchAllData(ctx context.Context) (*AllData, error) {
	data := &AllData{}

	// Fetch artists
	artists, err := c.FetchArtists(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching artists: %w", err)
	}
	data.Artists = artists

	// Fetch locations
	locations, err := c.FetchLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching locations: %w", err)
	}
	data.Locations = locations

	// Fetch dates
	dates, err := c.FetchDates(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching dates: %w", err)
	}
	data.Dates = dates

	// Fetch relations
	relations, err := c.FetchRelations(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching relations: %w", err)
	}
	data.Relations = relations

	return data, nil
}
