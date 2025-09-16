// Package api provides functionality to fetch data from the Groupie Tracker API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Artist represents a musical artist from the external API.
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

// Relation represents concert relationships from the API.
type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// Response represents the API response structure.
type Response struct {
	Artists   []Artist   `json:"artists,omitempty"`
	Relations []Relation `json:"relations,omitempty"`
}

// Client represents an HTTP client for the Groupie Tracker API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchAll retrieves all data from the API endpoints.
func (c *Client) FetchAll(ctx context.Context) (*Response, error) {
	response := &Response{}

	// Fetch artists
	artists, err := c.fetchArtists(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching artists: %w", err)
	}
	response.Artists = artists

	// Fetch relations
	relations, err := c.fetchRelations(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching relations: %w", err)
	}
	response.Relations = relations

	return response, nil
}

func (c *Client) fetchArtists(ctx context.Context) ([]Artist, error) {
	var artists []Artist
	err := c.fetch(ctx, "/api/artists", &artists)
	return artists, err
}

func (c *Client) fetchRelations(ctx context.Context) ([]Relation, error) {
	var response struct {
		Index []Relation `json:"index"`
	}
	err := c.fetch(ctx, "/api/relation", &response)
	return response.Index, err
}

func (c *Client) fetch(ctx context.Context, path string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}
