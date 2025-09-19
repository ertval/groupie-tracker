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
	if err := repo.LoadData(context.Background()); err != nil {
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
		{"Static CSS File", "/static/css/base.css", "GET", http.StatusOK, ""},
		{"Static Not Found", "/static/not-found.css", "GET", http.StatusNotFound, ""},
		// Extra static handler tests
		{"Favicon", "/favicon.ico", "GET", http.StatusOK, ""},
		{"Favicon Invalid Method", "/favicon.ico", "POST", http.StatusMethodNotAllowed, "Method not allowed"},
		{"Static Directory Browse", "/static/css/", "GET", http.StatusNotFound, ""},
		{"Static Path Traversal", "/static/../go.mod", "GET", http.StatusNotFound, ""},
		{"Static HEAD", "/static/css/base.css", "HEAD", http.StatusOK, ""},
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
			case tt.path == "/favicon.ico":
				handler = h.StaticFiles
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

func TestStaticFilesAdvanced(t *testing.T) {
	h := newTestApplication(t)

	// Create temporary test files for more comprehensive testing
	tempDir := t.TempDir()
	testStaticDir := filepath.Join(tempDir, "static")
	err := os.MkdirAll(filepath.Join(testStaticDir, "css"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test static directory: %v", err)
	}

	// Create test files with specific content and modification times
	testCSS := filepath.Join(testStaticDir, "css", "test.css")
	cssContent := "body { color: red; }"
	err = os.WriteFile(testCSS, []byte(cssContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CSS file: %v", err)
	}

	testJS := filepath.Join(testStaticDir, "js", "test.js")
	err = os.MkdirAll(filepath.Join(testStaticDir, "js"), 0755)
	if err != nil {
		t.Fatalf("Failed to create js directory: %v", err)
	}
	jsContent := "console.log('test');"
	err = os.WriteFile(testJS, []byte(jsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test JS file: %v", err)
	}

	testFavicon := filepath.Join(testStaticDir, "favicon.ico")
	faviconContent := "fake-favicon-data"
	err = os.WriteFile(testFavicon, []byte(faviconContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create favicon: %v", err)
	}

	// Change working directory to temp dir for this test
	originalWD, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWD)

	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		expectedStatus int
		expectedBody   string
		checkHeaders   map[string]string
		headerContains map[string]string
	}{
		{
			name:           "CSS file with proper headers",
			method:         "GET",
			path:           "/static/css/test.css",
			expectedStatus: http.StatusOK,
			expectedBody:   cssContent,
			checkHeaders: map[string]string{
				"Content-Type":           "text/css; charset=utf-8",
				"Cache-Control":          "public, max-age=31536000",
				"Vary":                   "Accept-Encoding",
				"X-Content-Type-Options": "nosniff",
			},
			headerContains: map[string]string{
				"ETag":          "\"",
				"Last-Modified": "GMT",
			},
		},
		{
			name:           "JS file with proper headers",
			method:         "GET",
			path:           "/static/js/test.js",
			expectedStatus: http.StatusOK,
			expectedBody:   jsContent,
			checkHeaders: map[string]string{
				"Content-Type":  "application/javascript; charset=utf-8",
				"Cache-Control": "public, max-age=31536000",
			},
		},
		{
			name:           "Favicon with caching",
			method:         "GET",
			path:           "/favicon.ico",
			expectedStatus: http.StatusOK,
			expectedBody:   faviconContent,
			checkHeaders: map[string]string{
				"Cache-Control":          "public, max-age=86400",
				"Vary":                   "Accept-Encoding",
				"X-Content-Type-Options": "nosniff",
			},
		},
		{
			name:           "HEAD request for CSS",
			method:         "HEAD",
			path:           "/static/css/test.css",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Content-Type":  "text/css; charset=utf-8",
				"Cache-Control": "public, max-age=31536000",
			},
		},
		{
			name:           "Method not allowed",
			method:         "POST",
			path:           "/static/css/test.css",
			expectedStatus: http.StatusMethodNotAllowed,
			checkHeaders: map[string]string{
				"Allow": "GET, HEAD",
			},
		},
		{
			name:           "Path traversal attempt 1",
			method:         "GET",
			path:           "/static/../favicon.ico",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Path traversal attempt 2",
			method:         "GET",
			path:           "/static/css/../../go.mod",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Directory listing attempt",
			method:         "GET",
			path:           "/static/css/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Non-existent file",
			method:         "GET",
			path:           "/static/nonexistent.css",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid path prefix",
			method:         "GET",
			path:           "/assets/test.css",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Empty relative path",
			method:         "GET",
			path:           "/static/",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)

			// Add any custom headers for the request
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}

			w := httptest.NewRecorder()
			h.StaticFiles(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" {
				body := w.Body.String()
				if body != tt.expectedBody {
					t.Errorf("Expected body %q, got %q", tt.expectedBody, body)
				}
			}

			// Check exact header matches
			for header, expectedValue := range tt.checkHeaders {
				if got := w.Header().Get(header); got != expectedValue {
					t.Errorf("Expected header %s: %q, got %q", header, expectedValue, got)
				}
			}

			// Check header contains
			for header, expectedSubstring := range tt.headerContains {
				if got := w.Header().Get(header); !strings.Contains(got, expectedSubstring) {
					t.Errorf("Expected header %s to contain %q, got %q", header, expectedSubstring, got)
				}
			}
		})
	}
}

func TestStaticFilesCaching(t *testing.T) {
	h := newTestApplication(t)

	// Create temporary static directory and file
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test static directory: %v", err)
	}

	testFile := filepath.Join(staticDir, "test.txt")
	testContent := "test content for caching"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	originalWD, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWD)

	// First request to get ETag and Last-Modified
	req1 := httptest.NewRequest("GET", "/static/test.txt", nil)
	w1 := httptest.NewRecorder()
	h.StaticFiles(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w1.Code)
	}

	etag := w1.Header().Get("ETag")
	lastModified := w1.Header().Get("Last-Modified")

	if etag == "" {
		t.Error("Expected ETag header to be set")
	}
	if lastModified == "" {
		t.Error("Expected Last-Modified header to be set")
	}

	// Test If-None-Match with matching ETag (should return 304)
	t.Run("If-None-Match with matching ETag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/test.txt", nil)
		req.Header.Set("If-None-Match", etag)
		w := httptest.NewRecorder()
		h.StaticFiles(w, req)

		if w.Code != http.StatusNotModified {
			t.Errorf("Expected status 304, got %d", w.Code)
		}

		// Body should be empty for 304
		if body := w.Body.String(); body != "" {
			t.Errorf("Expected empty body for 304, got %q", body)
		}
	})

	// Test If-None-Match with non-matching ETag (should return 200)
	t.Run("If-None-Match with non-matching ETag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/test.txt", nil)
		req.Header.Set("If-None-Match", `"different-etag"`)
		w := httptest.NewRecorder()
		h.StaticFiles(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if body := w.Body.String(); body != testContent {
			t.Errorf("Expected body %q, got %q", testContent, body)
		}
	})

	// Test If-Modified-Since with same time (should return 304)
	t.Run("If-Modified-Since with same time", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/test.txt", nil)
		req.Header.Set("If-Modified-Since", lastModified)
		w := httptest.NewRecorder()
		h.StaticFiles(w, req)

		if w.Code != http.StatusNotModified {
			t.Errorf("Expected status 304, got %d", w.Code)
		}
	})

	// Test If-Modified-Since with older time (should return 200)
	t.Run("If-Modified-Since with older time", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/test.txt", nil)
		req.Header.Set("If-Modified-Since", "Mon, 01 Jan 2000 00:00:00 GMT")
		w := httptest.NewRecorder()
		h.StaticFiles(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test wildcard If-None-Match (should return 304)
	t.Run("If-None-Match with wildcard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/test.txt", nil)
		req.Header.Set("If-None-Match", "*")
		w := httptest.NewRecorder()
		h.StaticFiles(w, req)

		if w.Code != http.StatusNotModified {
			t.Errorf("Expected status 304, got %d", w.Code)
		}
	})
}

func TestStaticFilesContentTypes(t *testing.T) {
	h := newTestApplication(t)

	// Create temporary static directory with different file types
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test static directory: %v", err)
	}

	// Create test files of different types
	testFiles := map[string]struct {
		content    string
		expectedCT string
		expectedCC string
	}{
		"style.css": {
			content:    "body{color:red}",
			expectedCT: "text/css; charset=utf-8",
			expectedCC: "public, max-age=31536000",
		},
		"script.js": {
			content:    "alert('test')",
			expectedCT: "application/javascript; charset=utf-8",
			expectedCC: "public, max-age=31536000",
		},
		"image.png": {
			content:    "fake-png-data",
			expectedCT: "image/png",
			expectedCC: "public, max-age=2592000",
		},
		"font.woff": {
			content:    "fake-woff-data",
			expectedCT: "font/woff",
			expectedCC: "public, max-age=31536000",
		},
		"data.json": {
			content:    "{}",
			expectedCT: "application/octet-stream",
			expectedCC: "public, max-age=3600",
		},
	}

	for filename, fileData := range testFiles {
		err = os.WriteFile(filepath.Join(staticDir, filename), []byte(fileData.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	originalWD, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWD)

	for filename, fileData := range testFiles {
		t.Run("Content type for "+filename, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/static/"+filename, nil)
			w := httptest.NewRecorder()
			h.StaticFiles(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			if ct := w.Header().Get("Content-Type"); ct != fileData.expectedCT {
				t.Errorf("Expected Content-Type %q, got %q", fileData.expectedCT, ct)
			}

			if cc := w.Header().Get("Cache-Control"); cc != fileData.expectedCC {
				t.Errorf("Expected Cache-Control %q, got %q", fileData.expectedCC, cc)
			}

			// Verify security headers are set
			if xct := w.Header().Get("X-Content-Type-Options"); xct != "nosniff" {
				t.Errorf("Expected X-Content-Type-Options %q, got %q", "nosniff", xct)
			}
		})
	}
}
