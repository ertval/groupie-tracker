// Package api provides functionality to fetch data from the Groupie Tracker API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"groupie-tracker/internal/data"
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
func (c *Client) FetchArtists(ctx context.Context) ([]data.Artist, error) {
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

	var artists []data.Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return artists, nil
}

// FetchLocations retrieves all locations from the API.
func (c *Client) FetchLocations(ctx context.Context) ([]data.Location, error) {
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
		Index []data.Location `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchDates retrieves all dates from the API.
func (c *Client) FetchDates(ctx context.Context) ([]data.Date, error) {
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
		Index []data.Date `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchRelations retrieves all relations from the API.
func (c *Client) FetchRelations(ctx context.Context) ([]data.Relation, error) {
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
		Index []data.Relation `json:"index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response.Index, nil
}

// FetchAllData retrieves all data from the API endpoints.
func (c *Client) FetchAllData(ctx context.Context) (*data.APIResponse, error) {
	response := &data.APIResponse{}

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
