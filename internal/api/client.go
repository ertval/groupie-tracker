package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL points to the public Groupie Tracker API instance.
const DefaultBaseURL = "https://groupietrackers.herokuapp.com/api"

// Client wraps the Groupie Tracker HTTP API and exposes typed fetch methods.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client instance.
type Option func(*Client)

// WithBaseURL allows overriding the remote API endpoint.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// WithHTTPClient injects a custom *http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithTimeout sets the timeout used by the underlying HTTP client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout <= 0 {
			return
		}
		if c.httpClient == nil {
			c.httpClient = &http.Client{Timeout: timeout}
			return
		}
		clone := *c.httpClient
		clone.Timeout = timeout
		c.httpClient = &clone
	}
}

// NewClient constructs a ready-to-use API client.
func NewClient(opts ...Option) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// FetchArtists retrieves the artist catalog from the remote API.
func (c *Client) FetchArtists(ctx context.Context) ([]Artist, error) {
	var artists []Artist
	if err := c.fetch(ctx, "/artists", &artists); err != nil {
		return nil, err
	}
	return artists, nil
}

// FetchRelations retrieves the concert schedule data from the remote API.
func (c *Client) FetchRelations(ctx context.Context) ([]RelationIndex, error) {
	var relations Relation
	if err := c.fetch(ctx, "/relation", &relations); err != nil {
		return nil, err
	}
	return relations.Index, nil
}

// FetchAll downloads both artists and relations in sequence.
func (c *Client) FetchAll(ctx context.Context) ([]Artist, []RelationIndex, error) {
	artists, err := c.FetchArtists(ctx)
	if err != nil {
		return nil, nil, err
	}

	relations, err := c.FetchRelations(ctx)
	if err != nil {
		return nil, nil, err
	}

	return artists, relations, nil
}

func (c *Client) fetch(ctx context.Context, path string, target interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api: unexpected status %d for %s", resp.StatusCode, path)
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("api: decode %s failed: %w", path, err)
	}

	return nil
}
