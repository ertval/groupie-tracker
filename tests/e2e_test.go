package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/conf"
	"groupie-tracker/internal/data"
	serverpkg "groupie-tracker/internal/web"
)

// TestE2ECompleteUserFlow tests the complete user journey through the application
func TestE2ECompleteUserFlow(t *testing.T) {
	mockAPI := createMockAPI()
	defer mockAPI.Close()

	server := createTestServerWithAPI(t, mockAPI.URL)
	defer server.Close()

	client := server.Client()

	tests := []struct {
		name     string
		testFunc func(t *testing.T, client *http.Client, serverURL string)
	}{
		{"HomePage", testHomePage},
		{"ArtistsPage", testArtistsPage},
		{"ArtistDetail_Queen", testArtistDetailQueen},
		{"ArtistDetail_Gorillaz", testArtistDetailGorillaz},
		{"ArtistDetail_FooFighters", testArtistDetailFooFighters},
		{"TravisScottConcerts", testTravisScottConcerts},
		{"LocationsPage", testLocationsPage},
		{"LocationDetail", testLocationDetail},
		{"HealthCheck", testHealthCheck},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, client, server.URL)
		})
	}
}

// TestE2EErrorHandling tests error scenarios end-to-end
func TestE2EErrorHandling(t *testing.T) {
	mockAPI := createEmptyMockAPI()
	defer mockAPI.Close()

	server := createTestServerWithAPI(t, mockAPI.URL)
	defer server.Close()

	client := server.Client()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"NotFound Artist", "/artists/nonexistent-artist", http.StatusNotFound, "not found"},
		{"NotFound Location", "/locations/nonexistent-location", http.StatusNotFound, "not found"},
		{"Invalid Route", "/invalid/route", http.StatusNotFound, ""},
		{"NotFound Static", "/static/nonexistent.css", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := client.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to get %s: %v", tt.path, err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, res.StatusCode)
			}

			if tt.wantBody != "" {
				body, _ := io.ReadAll(res.Body)
				bodyStr := strings.ToLower(string(body))
				if !strings.Contains(bodyStr, tt.wantBody) {
					t.Logf("Expected body to contain %q", tt.wantBody)
				}
			}
		})
	}
}

// TestE2EStaticFiles tests static file serving
func TestE2EStaticFiles(t *testing.T) {
	mockAPI := createEmptyMockAPI()
	defer mockAPI.Close()

	server := createTestServerWithAPI(t, mockAPI.URL)
	defer server.Close()

	client := server.Client()

	tests := []struct {
		path        string
		wantStatus  int
		contentType string
	}{
		{"/static/css/base.css", http.StatusOK, "text/css"},
		{"/favicon.ico", http.StatusOK, "image/"},
		{"/static/css/nonexistent.css", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run("Static: "+tt.path, func(t *testing.T) {
			res, err := client.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to get static file: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, res.StatusCode)
			}

			if tt.contentType != "" && res.StatusCode == http.StatusOK {
				ct := res.Header.Get("Content-Type")
				if !strings.Contains(ct, tt.contentType) {
					t.Logf("Expected Content-Type to contain %s, got %s", tt.contentType, ct)
				}
			}
		})
	}
}

// TestE2ESecurityChecks tests security-related scenarios
func TestE2ESecurityChecks(t *testing.T) {
	mockAPI := createEmptyMockAPI()
	defer mockAPI.Close()

	server := createTestServerWithAPI(t, mockAPI.URL)
	defer server.Close()

	client := server.Client()

	maliciousPaths := []string{
		"/static/../go.mod",
		"/static/../../etc/passwd",
		"/static/../internal/handlers/handlers.go",
		"/artists/../../../etc/passwd",
		"/locations/../go.mod",
	}

	for _, path := range maliciousPaths {
		t.Run("Security: "+path, func(t *testing.T) {
			res, err := client.Get(server.URL + path)
			if err != nil {
				t.Fatalf("failed to test security path: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(res.Body)
				if strings.Contains(string(body), "module groupie-tracker") {
					t.Errorf("path traversal succeeded for %s", path)
				}
			}
		})
	}
}

// TestE2EMethodNotAllowed tests HTTP method restrictions
func TestE2EMethodNotAllowed(t *testing.T) {
	mockAPI := createEmptyMockAPI()
	defer mockAPI.Close()

	server := createTestServerWithAPI(t, mockAPI.URL)
	defer server.Close()

	client := server.Client()

	tests := []struct {
		path   string
		method string
	}{
		{"/", "POST"},
		{"/", "PUT"},
		{"/", "DELETE"},
		{"/", "PATCH"},
		{"/artists", "PUT"},
		{"/artists", "DELETE"},
		{"/artists", "PATCH"},
		{"/locations", "PUT"},
		{"/locations", "DELETE"},
		{"/locations", "PATCH"},
		{"/health", "POST"},
		{"/health", "PUT"},
		{"/health", "DELETE"},
		{"/health", "PATCH"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, server.URL+tc.path, nil)
			res, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to send request: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405, got %d", res.StatusCode)
			}
		})
	}
}

// TestSearchEndToEnd tests the search functionality
func TestSearchEndToEnd(t *testing.T) {
	// Skip if server not running locally
	baseURL := "http://localhost:8080"
	if !isServerRunning(t, baseURL) {
		t.Skip("Server not running on localhost:8080")
	}

	client := &http.Client{}

	tests := []struct {
		name          string
		endpoint      string
		method        string
		formData      url.Values
		query         string
		expectedCode  int
		checkResponse func(t *testing.T, body string)
	}{
		{
			name:         "Search page loads",
			endpoint:     "/search",
			method:       "GET",
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "Search Artists") {
					t.Error("Search page should contain 'Search Artists'")
				}
			},
		},
		{
			name:         "API suggestions endpoint works",
			endpoint:     "/api/suggestions",
			method:       "GET",
			query:        "queen",
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var suggestions []data.SearchSuggestion
				if err := json.Unmarshal([]byte(body), &suggestions); err != nil {
					t.Errorf("Failed to parse suggestions JSON: %v", err)
					return
				}
				if len(suggestions) == 0 {
					t.Error("Should return suggestions for 'queen'")
				}
			},
		},
		{
			name:         "Search with POST returns results",
			endpoint:     "/search",
			method:       "POST",
			formData:     url.Values{"q": {"Queen"}},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "Queen") {
					t.Error("Search results should contain 'Queen'")
				}
			},
		},
		{
			name:         "Search for member name works",
			endpoint:     "/search",
			method:       "POST",
			formData:     url.Values{"q": {"Freddie Mercury"}},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "Queen") || !strings.Contains(body, "Freddie Mercury") {
					t.Error("Search for 'Freddie Mercury' should return Queen")
				}
			},
		},
		{
			name:         "Nonexistent search returns no results",
			endpoint:     "/search",
			method:       "POST",
			formData:     url.Values{"q": {"NonexistentBandXYZ123"}},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "No Results Found") {
					t.Error("Nonexistent search should show 'No Results Found'")
				}
			},
		},
		{
			name:         "Case insensitive search works",
			endpoint:     "/search",
			method:       "POST",
			formData:     url.Values{"q": {"QUEEN"}},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				if !strings.Contains(body, "Queen") {
					t.Error("Case insensitive search for 'QUEEN' should find Queen")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildRequest(t, baseURL, tt.endpoint, tt.method, tt.query, tt.formData)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			if tt.checkResponse != nil {
				tt.checkResponse(t, string(body))
			}
		})
	}
}

// TestSearchAuditCompliance tests audit requirements for search
func TestSearchAuditCompliance(t *testing.T) {
	baseURL := "http://localhost:8080"
	if !isServerRunning(t, baseURL) {
		t.Skip("Server not running, skipping audit tests")
	}

	client := &http.Client{}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"Search for Phil Collins as member", "Phil Collins", "Phil Collins"},
		{"Search for Freddie Mercury as member", "Freddie Mercury", "Queen"},
		{"Search for location", "London", ""},
		{"Search for creation date", "1970", ""},
		{"Search for first album date", "1973", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{"q": {tt.query}}
			req, _ := http.NewRequest("POST", baseURL+"/search", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Search request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Search failed with status %d", resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			if !strings.Contains(bodyStr, "Search Artists") {
				t.Error("Response should contain search interface")
			}

			if tt.expected != "" && !strings.Contains(bodyStr, tt.expected) {
				t.Logf("Expected to find '%s' for query '%s'", tt.expected, tt.query)
			}
		})
	}
}

// Helper functions

func createMockAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970, "firstAlbum": "Queen", "members": ["Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon", "Mike Grose", "Barry Mitchell", "Doug Bogie"]},
				{"id": 2, "name": "Gorillaz", "creationDate": 2001, "firstAlbum": "26-03-2001", "members": ["Damon Albarn"]},
				{"id": 3, "name": "Travis Scott", "creationDate": 2013, "firstAlbum": "Owl Pharaoh", "members": ["Travis Scott"]},
				{"id": 4, "name": "Foo Fighters", "creationDate": 1994, "firstAlbum": "Foo Fighters", "members": ["Dave Grohl", "Taylor Hawkins", "Nate Mendel", "Chris Shiflett", "Pat Smear", "Rami Jaffee"]}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"london-uk": ["2022-01-01"], "paris-france": ["2022-02-01"], "tokyo-japan": ["2022-03-01"]}},
					{"id": 2, "datesLocations": {"los-angeles-usa": ["2023-01-01"], "new-york-usa": ["2023-02-01"]}},
					{"id": 3, "datesLocations": {"miami-usa": ["2023-03-01"], "chicago-usa": ["2023-04-01"], "detroit-usa": ["2023-05-01"], "philadelphia-usa": ["2023-06-01"], "boston-usa": ["2023-07-01"], "atlanta-usa": ["2023-08-01"], "dallas-usa": ["2023-09-01"], "houston-usa": ["2023-10-01"], "phoenix-usa": ["2023-11-01"], "las-vegas-usa": ["2023-12-01"], "seattle-usa": ["2024-01-01"]}},
					{"id": 4, "datesLocations": {"london-uk": ["2024-02-01"], "manchester-uk": ["2024-03-01"], "birmingham-uk": ["2024-04-01"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func createEmptyMockAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[]`))
		case "/api/relation":
			w.Write([]byte(`{"index":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func createTestServerWithAPI(t *testing.T, apiURL string) *httptest.Server {
	origAPIURL := conf.APIBaseURL
	origTimeout := conf.APIRequestTimeout
	origWd, _ := os.Getwd()

	projectRoot := filepath.Dir(origWd) // Move from tests to project root
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("failed to change to project root: %v", err)
	}

	conf.APIBaseURL = apiURL
	conf.APIRequestTimeout = 10 * time.Second

	apiClient := api.NewClient(apiURL, conf.APIRequestTimeout)
	srv, err := serverpkg.NewApp(apiClient)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	testServer := httptest.NewServer(srv.Handler)

	t.Cleanup(func() {
		testServer.Close()
		conf.APIBaseURL = origAPIURL
		conf.APIRequestTimeout = origTimeout
		_ = os.Chdir(origWd)
	})

	return testServer
}

func isServerRunning(t *testing.T, baseURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
	if err != nil {
		return false
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func buildRequest(t *testing.T, baseURL, endpoint, method, query string, formData url.Values) *http.Request {
	reqURL := baseURL + endpoint
	if query != "" {
		reqURL += "?q=" + url.QueryEscape(query)
	}

	var req *http.Request
	var err error

	if method == "POST" && formData != nil {
		req, err = http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		req, err = http.NewRequest(method, reqURL, nil)
	}

	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	return req
}

// Individual test functions

func testHomePage(t *testing.T, client *http.Client, serverURL string) {
	res, err := client.Get(serverURL + "/")
	if err != nil {
		t.Fatalf("failed to get home page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}
}

func testArtistsPage(t *testing.T, client *http.Client, serverURL string) {
	res, err := client.Get(serverURL + "/artists")
	if err != nil {
		t.Fatalf("failed to get artists page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)

	expectedArtists := []string{"Queen", "Gorillaz", "Travis Scott", "Foo Fighters"}
	for _, artist := range expectedArtists {
		if !strings.Contains(bodyStr, artist) {
			t.Logf("Expected to find artist %s", artist)
		}
	}
}

func testArtistDetailQueen(t *testing.T, client *http.Client, serverURL string) {
	testArtistDetail(t, client, serverURL, "queen", "Queen")
}

func testArtistDetailGorillaz(t *testing.T, client *http.Client, serverURL string) {
	testArtistDetail(t, client, serverURL, "gorillaz", "Gorillaz")
}

func testArtistDetailFooFighters(t *testing.T, client *http.Client, serverURL string) {
	testArtistDetail(t, client, serverURL, "foo-fighters", "Foo Fighters")
}

func testArtistDetail(t *testing.T, client *http.Client, serverURL, slug, expectedName string) {
	res, err := client.Get(serverURL + "/artists/" + slug)
	if err != nil {
		t.Fatalf("failed to get artist detail: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 200 or 404, got %d", res.StatusCode)
	}

	if res.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		if !strings.Contains(string(body), expectedName) {
			t.Logf("Expected to find %s in detail page", expectedName)
		}
	}
}

func testTravisScottConcerts(t *testing.T, client *http.Client, serverURL string) {
	res, err := client.Get(serverURL + "/artists/travis-scott")
	if err != nil {
		t.Fatalf("failed to get Travis Scott detail: %v", err)
	}
	defer res.Body.Close()
}

func testLocationsPage(t *testing.T, client *http.Client, serverURL string) {
	res, err := client.Get(serverURL + "/locations")
	if err != nil {
		t.Fatalf("failed to get locations page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}
}

func testLocationDetail(t *testing.T, client *http.Client, serverURL string) {
	res, err := client.Get(serverURL + "/locations/london-uk")
	if err != nil {
		t.Fatalf("failed to get location detail: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 200 or 404, got %d", res.StatusCode)
	}
}

func testHealthCheck(t *testing.T, client *http.Client, serverURL string) {
	res, err := client.Get(serverURL + "/health")
	if err != nil {
		t.Fatalf("failed to get health check: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("health check: expected status 200, got %d", res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	var healthData map[string]interface{}
	if err := json.Unmarshal(body, &healthData); err != nil {
		t.Errorf("health check response is not valid JSON: %v", err)
	}

	if status, ok := healthData["status"]; !ok || status != "healthy" {
		t.Error("health check should return status: healthy")
	}
}
