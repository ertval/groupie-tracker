package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"groupie-tracker/internal/data"
)

// setupTestServer creates a test server with real data for integration tests
func setupTestServer() (*httptest.Server, error) {
	// Initialize global repository with test data
	repo = &data.Repository{}

	// Mock templates for testing (skip template loading)
	templates = make(map[string]*template.Template)

	// Create template function map
	funcMap := template.FuncMap{
		"join": func(items []string, sep string) string { return strings.Join(items, sep) },
	}

	// Create minimal template for search page
	searchTmpl, err := template.New("search.tmpl").Funcs(funcMap).Parse(`
		<html><body>
		<h1>Search Artists</h1>
		<input class="search-input" name="q" value="{{.Query}}">
		{{if .IsSearch}}
			{{if .Results.Artists}}
				<p>Found {{.Results.TotalResults}} artist{{if ne .Results.TotalResults 1}}s{{end}}</p>
				{{range .Results.Artists}}
				<div class="artist">{{.Name}} - {{join .Members ", "}} - {{.CreationYear}} - {{.ConcertCount}} concerts</div>
				{{end}}
			{{else}}
				<div class="no-results">No Results Found</div>
			{{end}}
		{{end}}
		</body></html>
	`)
	if err != nil {
		return nil, err
	}

	templates["search.tmpl"] = searchTmpl

	// Create test data
	artists := []data.Artist{
		{
			ID:           1,
			Name:         "Queen",
			Slug:         "queen",
			Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Countries:    []string{"UK", "USA", "Japan"},
			ConcertCount: 15,
			Image:        "http://example.com/queen.jpg",
		},
		{
			ID:           2,
			Name:         "Phil Collins",
			Slug:         "phil-collins",
			Members:      []string{"Phil Collins"},
			CreationYear: 1981,
			FirstAlbum:   "05-02-1981",
			Countries:    []string{"UK", "USA"},
			ConcertCount: 8,
			Image:        "http://example.com/phil.jpg",
		},
		{
			ID:           3,
			Name:         "Pink Floyd",
			Slug:         "pink-floyd",
			Members:      []string{"David Gilmour", "Roger Waters", "Nick Mason", "Richard Wright"},
			CreationYear: 1965,
			FirstAlbum:   "05-08-1967",
			Countries:    []string{"UK", "USA", "Germany"},
			ConcertCount: 12,
			Image:        "http://example.com/pink-floyd.jpg",
		},
	}

	locations := []data.Location{
		{
			Name: "London UK",
			Slug: "london-uk",
		},
		{
			Name: "New York USA",
			Slug: "new-york-usa",
		},
		{
			Name: "Philadelphia USA",
			Slug: "philadelphia-usa",
		},
	}

	// Mock the repository with test data
	repo.SetTestData(artists, locations)

	// Create test server
	mux := createServeMux()
	server := httptest.NewServer(withMiddleware(mux))
	return server, nil
}

// TestSearchEndpoints tests the search functionality end-to-end
func TestSearchEndpoints(t *testing.T) {
	server, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer server.Close()

	tests := []struct {
		name           string
		method         string
		path           string
		formData       url.Values
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response, body string)
	}{
		{
			name:           "GET /search returns search page",
			method:         "GET",
			path:           "/search",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "Search Artists") {
					t.Error("Response should contain 'Search Artists'")
				}
				if !strings.Contains(body, "search-input") {
					t.Error("Response should contain search input element")
				}
			},
		},
		{
			name:           "POST /search with query returns results",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {"Queen"}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "Queen") {
					t.Error("Response should contain Queen in results")
				}
				if !strings.Contains(body, "Found 1 artist") {
					t.Error("Response should show result count")
				}
				if !strings.Contains(body, "Freddie Mercury") {
					t.Error("Response should show band members")
				}
			},
		},
		{
			name:           "POST /search with member name returns artist",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {"Freddie Mercury"}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "Queen") {
					t.Error("Response should contain Queen (Freddie's band)")
				}
			},
		},
		{
			name:           "POST /search with filters",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {""}, "creationYearFrom": {"1980"}, "creationYearTo": {"1985"}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "Phil Collins") {
					t.Error("Response should contain Phil Collins (created 1981)")
				}
				if strings.Contains(body, "Queen") {
					t.Error("Response should not contain Queen (created 1970)")
				}
			},
		},
		{
			name:           "POST /search with no results",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {"Nonexistent Band"}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "No Results Found") {
					t.Error("Response should show 'No Results Found'")
				}
			},
		},
		{
			name:           "POST /search combined query and filters",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {"Phil"}, "memberCounts": {"1"}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "Phil Collins") {
					t.Error("Response should contain Phil Collins (matches both criteria)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.method == "POST" && tt.formData != nil {
				req, err = http.NewRequest("POST", server.URL+tt.path, strings.NewReader(tt.formData.Encode()))
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req, err = http.NewRequest(tt.method, server.URL+tt.path, nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body := make([]byte, 0)
			buf := make([]byte, 1024)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if err != nil {
					break
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp, string(body))
			}
		})
	}
}

// TestSearchSuggestionsAPI tests the suggestions JSON API
func TestSearchSuggestionsAPI(t *testing.T) {
	server, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer server.Close()

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		checkResponse  func(t *testing.T, suggestions []data.SearchSuggestion)
	}{
		{
			name:           "Empty query returns empty suggestions",
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, suggestions []data.SearchSuggestion) {
				if len(suggestions) != 0 {
					t.Errorf("Expected 0 suggestions, got %d", len(suggestions))
				}
			},
		},
		{
			name:           "Artist name query returns suggestions",
			query:          "Queen",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, suggestions []data.SearchSuggestion) {
				if len(suggestions) == 0 {
					t.Error("Expected suggestions for 'Queen'")
					return
				}

				found := false
				for _, s := range suggestions {
					if s.Text == "Queen" && s.Type == data.SuggestionTypeArtist {
						found = true
						if s.URL != "/artists/queen" {
							t.Errorf("Expected URL '/artists/queen', got '%s'", s.URL)
						}
						if s.ArtistID != 1 {
							t.Errorf("Expected ArtistID 1, got %d", s.ArtistID)
						}
						break
					}
				}
				if !found {
					t.Error("Should find Queen artist suggestion")
				}
			},
		},
		{
			name:           "Member name query returns suggestions",
			query:          "Freddie",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, suggestions []data.SearchSuggestion) {
				found := false
				for _, s := range suggestions {
					if s.Text == "Freddie Mercury" && s.Type == data.SuggestionTypeMember {
						found = true
						if !strings.Contains(s.Description, "Queen") {
							t.Errorf("Member suggestion should reference Queen band")
						}
						break
					}
				}
				if !found {
					t.Error("Should find Freddie Mercury member suggestion")
				}
			},
		},
		{
			name:           "Year query returns suggestions",
			query:          "1970",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, suggestions []data.SearchSuggestion) {
				found := false
				for _, s := range suggestions {
					if s.Text == "1970" && s.Type == data.SuggestionTypeCreation {
						found = true
						break
					}
				}
				if !found {
					t.Error("Should find 1970 creation year suggestion")
				}
			},
		},
		{
			name:           "Location query returns suggestions",
			query:          "London",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, suggestions []data.SearchSuggestion) {
				found := false
				for _, s := range suggestions {
					if s.Text == "London UK" && s.Type == data.SuggestionTypeLocation {
						found = true
						if s.URL != "/locations/london-uk" {
							t.Errorf("Expected location URL '/locations/london-uk', got '%s'", s.URL)
						}
						break
					}
				}
				if !found {
					t.Error("Should find London location suggestion")
				}
			},
		},
		{
			name:           "Partial query returns multiple suggestions",
			query:          "Phil",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, suggestions []data.SearchSuggestion) {
				if len(suggestions) < 2 {
					t.Errorf("Expected multiple suggestions for 'Phil', got %d", len(suggestions))
				}

				// Should find Phil Collins as both artist and member, plus Philadelphia location
				types := make(map[data.SearchSuggestionType]bool)
				for _, s := range suggestions {
					types[s.Type] = true
				}

				if !types[data.SuggestionTypeArtist] {
					t.Error("Should include artist suggestion")
				}
				if !types[data.SuggestionTypeMember] {
					t.Error("Should include member suggestion")
				}
				if !types[data.SuggestionTypeLocation] {
					t.Error("Should include location suggestion")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := server.URL + "/api/suggestions?q=" + url.QueryEscape(tt.query)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var suggestions []data.SearchSuggestion
			if err := json.NewDecoder(resp.Body).Decode(&suggestions); err != nil {
				t.Fatalf("Failed to decode JSON response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, suggestions)
			}
		})
	}
}

// TestSearchEdgeCases tests edge cases and error conditions
func TestSearchEdgeCases(t *testing.T) {
	server, err := setupTestServer()
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer server.Close()

	tests := []struct {
		name           string
		method         string
		path           string
		formData       url.Values
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response, body string)
	}{
		{
			name:           "Invalid method on /search",
			method:         "PUT",
			path:           "/search",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if resp.Header.Get("Allow") != "GET, POST" {
					t.Error("Should include Allow header with GET, POST")
				}
			},
		},
		{
			name:           "Invalid method on /api/suggestions",
			method:         "POST",
			path:           "/api/suggestions",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if resp.Header.Get("Allow") != "GET" {
					t.Error("Should include Allow header with GET")
				}
			},
		},
		{
			name:           "Special characters in search query",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {"<script>alert('xss')</script>"}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "No Results Found") {
					t.Error("Should show no results for XSS attempt")
				}
				// Verify no script execution
				if strings.Contains(body, "<script>") {
					t.Error("Response should not contain unescaped script tags")
				}
			},
		},
		{
			name:           "Very long search query",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {strings.Repeat("a", 1000)}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				// Should handle gracefully without crashing
				if !strings.Contains(body, "Search Artists") {
					t.Error("Should still render search page")
				}
			},
		},
		{
			name:           "Unicode characters in search",
			method:         "POST",
			path:           "/search",
			formData:       url.Values{"q": {"Ñiño 中文 🎵"}},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response, body string) {
				// Should handle gracefully
				if !strings.Contains(body, "Search Artists") {
					t.Error("Should still render search page for unicode input")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.method == "POST" && tt.formData != nil {
				req, err = http.NewRequest("POST", server.URL+tt.path, strings.NewReader(tt.formData.Encode()))
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req, err = http.NewRequest(tt.method, server.URL+tt.path, nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body := make([]byte, 0)
			buf := make([]byte, 1024)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if err != nil {
					break
				}
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp, string(body))
			}
		})
	}
}
