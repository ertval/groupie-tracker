package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles communication with the external API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client with the specified configuration.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// FetchArtists retrieves all artists from the API.
func (c *Client) FetchArtists(ctx context.Context) ([]Artist, error) {
	var artists []Artist
	err := c.fetchJSON(ctx, "/api/artists", &artists)
	return artists, err
}

// FetchRelations retrieves all relation data from the API.
func (c *Client) FetchRelations(ctx context.Context) (Relation, error) {
	var rel Relation
	err := c.fetchJSON(ctx, "/api/relation", &rel)
	return rel, err
}

// fetchJSON is a helper method to fetch and decode JSON from the API.
func (c *Client) fetchJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}
