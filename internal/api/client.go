package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles all external API communications.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client with configured timeout.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchArtists retrieves all artists from the external API.
func (c *Client) FetchArtists(ctx context.Context) ([]APIArtist, error) {
	url := c.baseURL + "/api/artists"
	var artists []APIArtist

	if err := c.fetchJSON(ctx, url, &artists); err != nil {
		return nil, fmt.Errorf("failed to fetch artists: %w", err)
	}

	return artists, nil
}

// FetchRelations retrieves all concert relations from the external API.
func (c *Client) FetchRelations(ctx context.Context) (*APIRelation, error) {
	url := c.baseURL + "/api/relation"
	var relations APIRelation

	if err := c.fetchJSON(ctx, url, &relations); err != nil {
		return nil, fmt.Errorf("failed to fetch relations: %w", err)
	}

	return &relations, nil
}

// fetchJSON is a helper method for making HTTP requests and decoding JSON responses.
func (c *Client) fetchJSON(ctx context.Context, url string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return nil
}
