package handlers

import (
	"context"
	"groupie-tracker/internal/data"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestApplication creates a new application instance for testing.
func newTestApplication(t *testing.T) *Handler {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970},
				{"id": 2, "name": "AC/DC", "creationDate": 1973}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"london-uk": ["2022-01-01"]}},
					{"id": 2, "datesLocations": {"london-uk": ["2023-01-01"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))

	// Create a new repository with the mock server's URL (disable caching for tests)
	repo := data.NewRepository(server.URL, 5*time.Second, false)
	_, _, _, err := repo.LoadData(context.Background())
	if err != nil {
		t.Fatalf("failed to load data for tests: %v", err)
	}

	// Create a new handler with the repository
	h := NewHandler(repo)

	t.Cleanup(func() {
		server.Close()
	})

	return h
}

func TestHandler_Routes(t *testing.T) {
	h := newTestApplication(t)

	tests := []struct {
		name       string
		path       string
		method     string
		wantStatus int
		body       string
	}{
		{"Home", "/", "GET", http.StatusOK, "Welcome to Groupie Tracker"},
		{"Home Invalid Method", "/", "POST", http.StatusMethodNotAllowed, "Method not allowed"},
		{"Artists", "/artists", "GET", http.StatusOK, "AC/DC"},
		{"Artist Detail", "/artists/queen", "GET", http.StatusOK, "Band Members"},
		{"Artist Detail Not Found", "/artists/not-found", "GET", http.StatusNotFound, "404 - Page Not Found"},
		{"Artist Detail Not Found by ID", "/artists/999", "GET", http.StatusNotFound, "404 - Page Not Found"},
		{"Locations", "/locations", "GET", http.StatusOK, "london-uk"},
		{"Location Detail", "/locations/london-uk", "GET", http.StatusOK, "Artists Who Performed Here"},
		{"Health", "/health", "GET", http.StatusOK, "healthy"},
		{"Static Image", "/static/img/artists/queen.jpg", "GET", http.StatusNotFound, ""},
		{"Static Not Found", "/static/not-found.css", "GET", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler http.HandlerFunc
			switch {
			case tt.path == "/":
				handler = h.Home
			case tt.path == "/artists":
				handler = h.Artists
			case strings.HasPrefix(tt.path, "/artists/"):
				handler = h.ArtistDetail
			case tt.path == "/locations":
				handler = h.Locations
			case strings.HasPrefix(tt.path, "/locations/"):
				handler = h.LocationDetail
			case tt.path == "/health":
				handler = h.Health
			case strings.HasPrefix(tt.path, "/static/"):
				handler = h.StaticFiles
			default:
				handler = h.Home // For not found case
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.body != "" {
				body, _ := io.ReadAll(w.Body)
				if !strings.Contains(string(body), tt.body) {
					t.Errorf("expected body to contain %q", tt.body)
				}
			}
		})
	}
}
