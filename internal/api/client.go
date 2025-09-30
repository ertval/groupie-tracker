// Package api provides external API client functionality for fetching data
// from the Groupie Tracker API endpoints. This encapsulates all HTTP communication
// and API data structures.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with the Groupie Tracker API.
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

// APIArtist represents the raw artist data structure from the /api/artists endpoint.
type APIArtist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Image        string   `json:"image"`
}

// APIRelationIndex represents a single artist's concert data from the /api/relation endpoint.
type APIRelationIndex struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// APIRelation wraps the complete concert relations dataset from the /api/relation endpoint.
type APIRelation struct {
	Index []APIRelationIndex `json:"index"`
}

// FetchArtists retrieves all artists from the /api/artists endpoint.
func (c *Client) FetchArtists(ctx context.Context) ([]APIArtist, error) {
	url := fmt.Sprintf("%s/artists", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d for artists endpoint", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read artists response: %w", err)
	}

	var artists []APIArtist
	if err := json.Unmarshal(body, &artists); err != nil {
		return nil, fmt.Errorf("failed to parse artists JSON: %w", err)
	}

	return artists, nil
}

// FetchRelations retrieves all concert relations from the /api/relation endpoint.
func (c *Client) FetchRelations(ctx context.Context) ([]APIRelationIndex, error) {
	url := fmt.Sprintf("%s/relation", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch relations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d for relations endpoint", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read relations response: %w", err)
	}

	var relations APIRelation
	if err := json.Unmarshal(body, &relations); err != nil {
		return nil, fmt.Errorf("failed to parse relations JSON: %w", err)
	}

	return relations.Index, nil
}

// FetchAllData retrieves both artists and relations in parallel for efficiency.
// This is the primary method used during application initialization.
func (c *Client) FetchAllData(ctx context.Context) ([]APIArtist, []APIRelationIndex, error) {
	// Create channels for concurrent fetching
	artistsChan := make(chan []APIArtist, 1)
	relationsChan := make(chan []APIRelationIndex, 1)
	errorsChan := make(chan error, 2)

	// Fetch artists concurrently
	go func() {
		artists, err := c.FetchArtists(ctx)
		if err != nil {
			errorsChan <- fmt.Errorf("artists fetch failed: %w", err)
			return
		}
		artistsChan <- artists
	}()

	// Fetch relations concurrently
	go func() {
		relations, err := c.FetchRelations(ctx)
		if err != nil {
			errorsChan <- fmt.Errorf("relations fetch failed: %w", err)
			return
		}
		relationsChan <- relations
	}()

	// Wait for both requests to complete
	var artists []APIArtist
	var relations []APIRelationIndex
	var errors []error

	for i := 0; i < 2; i++ {
		select {
		case a := <-artistsChan:
			artists = a
		case r := <-relationsChan:
			relations = r
		case err := <-errorsChan:
			errors = append(errors, err)
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context cancelled during API fetch: %w", ctx.Err())
		}
	}

	// Check for any errors
	if len(errors) > 0 {
		return nil, nil, fmt.Errorf("API fetch errors: %v", errors)
	}

	return artists, relations, nil
}
