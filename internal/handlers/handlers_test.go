package handlers

import (
	"context"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

	// Disable image caching for tests to avoid creating files on disk
	config.WithCache = false
	// Point repository to the mock server and use a short timeout for tests
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := data.NewRepository()
	_, _, _, err := repo.LoadData(context.Background())
	if err != nil {
		t.Fatalf("failed to load data for tests: %v", err)
	}

	// Create in-memory templates to avoid reading from disk during tests
	base := `{{define "base"}}<html><body>{{template "content" .}}</body></html>{{end}}`

	templates := make(map[string]*template.Template)
	// home.tmpl
	home := base + `{{define "content"}}Welcome to Groupie Tracker{{end}}`
	templates["home.tmpl"] = template.Must(template.New("home.tmpl").Parse(home))
	// artists.tmpl
	artists := base + `{{define "content"}}AC/DC{{end}}`
	templates["artists.tmpl"] = template.Must(template.New("artists.tmpl").Parse(artists))
	// artist_detail.tmpl
	artistDetail := base + `{{define "content"}}Band Members{{end}}`
	templates["artist_detail.tmpl"] = template.Must(template.New("artist_detail.tmpl").Parse(artistDetail))
	// locations.tmpl
	locations := base + `{{define "content"}}london-uk{{end}}`
	templates["locations.tmpl"] = template.Must(template.New("locations.tmpl").Parse(locations))
	// location_detail.tmpl
	locationDetail := base + `{{define "content"}}Artists Who Performed Here{{end}}`
	templates["location_detail.tmpl"] = template.Must(template.New("location_detail.tmpl").Parse(locationDetail))
	// error.tmpl (required by Error) - include code and message to match handlers.Error output
	errorTmpl := base + `{{define "content"}}{{.ErrorCode}} - {{.Message}}{{end}}`
	templates["error.tmpl"] = template.Must(template.New("error.tmpl").Parse(errorTmpl))

	// Create a handler instance without calling NewHandler (avoids loadTemplates)
	h := &Handler{repo: repo, templates: templates}

	// Ensure the working directory is the repository root so handlers.StaticFiles
	// can find the top-level `static` directory. Tests run from package dir.
	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(origWd, "..", "..")
	_ = os.Chdir(repoRoot)

	t.Cleanup(func() {
		server.Close()
		_ = os.Chdir(origWd)
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
		{"Artist Detail Not Found", "/artists/not-found", "GET", http.StatusNotFound, "Artist not found"},
		{"Artist Detail Not Found by ID", "/artists/999", "GET", http.StatusNotFound, "Artist not found"},
		{"Locations", "/locations", "GET", http.StatusOK, "london-uk"},
		{"Location Detail", "/locations/london-uk", "GET", http.StatusOK, "Artists Who Performed Here"},
		{"Health", "/health", "GET", http.StatusOK, "healthy"},
		{"Static Image", "/static/img/artists/queen.jpg", "GET", http.StatusOK, ""},
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
