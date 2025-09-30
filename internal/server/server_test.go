package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/search"
	"groupie-tracker/internal/testsupport"
)

func TestServerRoutes(t *testing.T) {
	store, err := data.Load(context.Background(), testsupport.MinimalDataset())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	svc := search.NewService(store)

	cfg := config.Config{
		Port:         ":0",
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 5,
		HTTPTimeout:  time.Second * 10,
		MaxBodyBytes: 1 << 20,
	}

	srv, err := New(store, svc, cfg)
	if err != nil {
		t.Fatalf("server construction failed: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []struct {
		name string
		path string
		want int
	}{
		{"home", "/", http.StatusOK},
		{"artists", "/artists", http.StatusOK},
		{"artist detail", "/artists/the-example", http.StatusOK},
		{"locations", "/locations", http.StatusOK},
		{"location detail", "/locations/new-york-usa", http.StatusOK},
		{"search", "/search?q=example", http.StatusOK},
		{"not found", "/unknown", http.StatusNotFound},
	}

	client := &http.Client{Timeout: 2 * time.Second}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(ts.URL + tc.path)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("status mismatch: got %d want %d", resp.StatusCode, tc.want)
			}
		})
	}

	t.Run("suggestions API", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/suggestions?q=exa")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("suggestions status: %d", resp.StatusCode)
		}

		var payload struct {
			Suggestions []search.Suggestion `json:"suggestions"`
			Total       int                 `json:"total"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if payload.Total == 0 {
			t.Fatalf("expected suggestions")
		}
	})

	t.Run("health", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("health status: %d", resp.StatusCode)
		}
	})
}
