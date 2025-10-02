package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client manages HTTP communication with the external Groupie Trackers API,
// handling request/response cycles with configurable timeout protection.
type Client struct {
	baseURL    string       // Base URL for all API endpoints (e.g., "https://groupietrackers.herokuapp.com")
	httpClient *http.Client // HTTP client with configured timeout to prevent hanging requests
}

// NewClient initializes an API client with a base URL and request timeout.
// The timeout applies to all HTTP requests made through this client to prevent indefinite blocking.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout}, // Timeout protects against slow/hanging responses
	}
}

// FetchArtists retrieves the complete artist list from the API's /api/artists endpoint.
// Returns raw Artist models that match the API JSON structure exactly (no computed fields yet).
func (c *Client) FetchArtists(ctx context.Context) ([]Artist, error) {
	var artists []Artist
	err := c.fetchJSON(ctx, "/api/artists", &artists) // Fetch and decode in one operation
	return artists, err
}

// FetchRelations retrieves concert date-location mappings from the API's /api/relation endpoint.
// Returns a Relation struct containing all artists' concert schedules indexed by artist ID.
func (c *Client) FetchRelations(ctx context.Context) (Relation, error) {
	var rel Relation
	err := c.fetchJSON(ctx, "/api/relation", &rel) // Fetch and decode in one operation
	return rel, err
}

// fetchJSON performs an HTTP GET request and decodes the JSON response into the destination type.
// This helper centralizes error handling and ensures consistent timeout/context behavior across all API calls.
func (c *Client) fetchJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil) // Context enables cancellation and timeout
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req) // httpClient.Timeout applies here
	if err != nil {
		return err
	}
	defer resp.Body.Close() // Always close response body to prevent resource leaks

	if resp.StatusCode != http.StatusOK { // Only accept 200 OK responses
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(dest) // Stream decode directly from response body for efficiency
}
