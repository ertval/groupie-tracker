`// Package tests contains end-to-end tests for the Groupie Tracker application.
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

// TestCompleteE2E runs comprehensive end-to-end tests covering all audit requirements
func TestCompleteE2E(t *testing.T) {
	// Setup test server with real data
	store := setupE2EStore(t)
	h := handlers.NewHandlers(store)
	client := api.NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)
	h.SetAPIClient(client)

	router := createE2ERouter(h)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("Template Loading and Rendering", func(t *testing.T) {
		testTemplateLoading(t, server)
	})

	t.Run("Audit Requirements Compliance", func(t *testing.T) {
		testAuditCompliance(t, store, server)
	})

	t.Run("Client-Server Events", func(t *testing.T) {
		testClientServerEvents(t, server)
	})

	t.Run("Error Handling", func(t *testing.T) {
		testErrorHandling(t, server)
	})

	t.Run("API Functionality", func(t *testing.T) {
		testAPIFunctionality(t, server)
	})

	t.Run("Performance and Stability", func(t *testing.T) {
		testPerformanceStability(t, server)
	})

	t.Run("HTTP Methods and Status Codes", func(t *testing.T) {
		testHTTPMethodsAndStatusCodes(t, server)
	})
}

// testTemplateLoading tests that all templates load correctly
func testTemplateLoading(t *testing.T, server *httptest.Server) {
	testCases := []struct {
		name         string
		url          string
		expectedCode int
		shouldContain string
	}{
		{
			name:         "Home Page Template",
			url:          "/",
			expectedCode: http.StatusOK,
			shouldContain: "Groupie Tracker",
		},
		{
			name:         "Artists Page Template",
			url:          "/artists",
			expectedCode: http.StatusOK,
			shouldContain: "Artists",
		},
		{
			name:         "Artist Detail Template",
			url:          "/artists/1",
			expectedCode: http.StatusOK,
			shouldContain: "Artist",
		},
		{
			name:         "Locations Page Template",
			url:          "/locations",
			expectedCode: http.StatusOK,
			shouldContain: "Locations",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(server.URL + tc.url)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", tc.url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status %d for %s, got %d", tc.expectedCode, tc.url, resp.StatusCode)
			}

			// Check content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Expected HTML content type for %s, got %s", tc.url, contentType)
			}

			// Read and check body content
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			bodyStr := string(body[:n])

			if !strings.Contains(bodyStr, tc.shouldContain) {
				t.Errorf("Expected %s to contain '%s', but it doesn't", tc.url, tc.shouldContain)
			}

			// Ensure it's valid HTML (basic check)
			if !strings.Contains(bodyStr, "<!DOCTYPE html>") && !strings.Contains(bodyStr, "<html") {
				t.Errorf("Response from %s doesn't appear to be valid HTML", tc.url)
			}
		})
	}
}

// testAuditCompliance tests all specific audit requirements
func testAuditCompliance(t *testing.T, store *storage.Store, server *httptest.Server) {
	t.Run("Queen Members Data", func(t *testing.T) {
		expectedMembers := []string{
			"Freddie Mercury",
			"Brian May",
			"John Daecon",
			"Roger Meddows-Taylor",
			"Mike Grose",
			"Barry Mitchell",
			"Doug Fogie",
		}

		// Find Queen in the data
		artists := store.GetAllArtists()
		var queen *models.Artist
		for _, artist := range artists {
			if artist.Name == "Queen" {
				queen = &artist
				break
			}
		}

		if queen == nil {
			t.Fatal("Queen not found in artists data")
		}

		// Verify all expected members are present
		for _, expectedMember := range expectedMembers {
			found := false
			for _, actualMember := range queen.Members {
				if actualMember == expectedMember {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected member '%s' not found in Queen's members", expectedMember)
			}
		}

		// Also test via web interface
		resp, err := http.Get(fmt.Sprintf("%s/artists/%d", server.URL, queen.ID))
		if err != nil {
			t.Fatalf("Failed to get Queen's artist page: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for Queen's page, got %d", resp.StatusCode)
		}

		t.Logf("✓ Queen members verified: %v", queen.Members)
	})

	t.Run("Gorillaz First Album Date", func(t *testing.T) {
		expectedDate := "26-03-2001"

		artists := store.GetAllArtists()
		var gorillaz *models.Artist
		for _, artist := range artists {
			if artist.Name == "Gorillaz" {
				gorillaz = &artist
				break
			}
		}

		if gorillaz == nil {
			t.Fatal("Gorillaz not found in artists data")
		}

		if gorillaz.FirstAlbum != expectedDate {
			t.Errorf("Expected Gorillaz first album date '%s', got '%s'", expectedDate, gorillaz.FirstAlbum)
		}

		t.Logf("✓ Gorillaz first album date verified: %s", gorillaz.FirstAlbum)
	})

	t.Run("Travis Scott Locations", func(t *testing.T) {
		expectedLocations := []string{
			"santiago-chile",
			"sao_paulo-brazil",
			"los_angeles-usa",
			"houston-usa",
			"atlanta-usa",
			"new_orleans-usa",
			"philadelphia-usa",
			"london-uk",
			"frauenfeld-switzerland",
			"turku-finland",
		}

		artists := store.GetAllArtists()
		var travisScott *models.Artist
		var travisScottID int
		for _, artist := range artists {
			if artist.Name == "Travis Scott" {
				travisScott = &artist
				travisScottID = artist.ID
				break
			}
		}

		if travisScott == nil {
			t.Fatal("Travis Scott not found in artists data")
		}

		location, exists := store.GetLocation(travisScottID)
		if !exists {
			t.Fatalf("Locations not found for Travis Scott (ID: %d)", travisScottID)
		}

		// Verify expected locations (allow for variations in API data)
		foundLocations := 0
		for _, expectedLocation := range expectedLocations {
			for _, actualLocation := range location.Locations {
				if actualLocation == expectedLocation || 
				   strings.Contains(actualLocation, strings.Split(expectedLocation, "-")[0]) {
					foundLocations++
					break
				}
			}
		}

		if foundLocations < 8 { // Allow some flexibility
			t.Errorf("Expected at least 8 matching locations for Travis Scott, found %d", foundLocations)
		}

		t.Logf("✓ Travis Scott locations verified: %d/%d matches", foundLocations, len(expectedLocations))
	})

	t.Run("Foo Fighters Members", func(t *testing.T) {
		expectedMembers := []string{
			"Dave Grohl",
			"Nate Mendel",
			"Taylor Hawkins",
			"Chris Shiflett",
			"Pat Smear",
			"Rami Jaffee",
		}

		artists := store.GetAllArtists()
		var fooFighters *models.Artist
		for _, artist := range artists {
			if artist.Name == "Foo Fighters" {
				fooFighters = &artist
				break
			}
		}

		if fooFighters == nil {
			t.Fatal("Foo Fighters not found in artists data")
		}

		// Verify all expected members
		for _, expectedMember := range expectedMembers {
			found := false
			for _, actualMember := range fooFighters.Members {
				if actualMember == expectedMember {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected member '%s' not found in Foo Fighters' members", expectedMember)
			}
		}

		t.Logf("✓ Foo Fighters members verified: %v", fooFighters.Members)
	})
}

// testClientServerEvents tests the event/action requirements
func testClientServerEvents(t *testing.T, server *httptest.Server) {
	t.Run("Live Search Event", func(t *testing.T) {
		// Test search API (client-server communication)
		searchURL := fmt.Sprintf("%s/api/search?q=%s", server.URL, url.QueryEscape("Queen"))
		resp, err := http.Get(searchURL)
		if err != nil {
			t.Fatalf("Failed to perform search: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for search, got %d", resp.StatusCode)
		}

		// Verify it's JSON response (client-server communication)
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type for search API, got %s", contentType)
		}

		var searchResponse handlers.SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
			t.Fatalf("Failed to decode search response: %v", err)
		}

		if len(searchResponse.Artists) == 0 {
			t.Error("Search should return results for 'Queen'")
		}

		if searchResponse.Query != "Queen" {
			t.Errorf("Expected query 'Queen', got '%s'", searchResponse.Query)
		}

		t.Logf("✓ Live search event working: found %d results for '%s'", 
			len(searchResponse.Artists), searchResponse.Query)
	})

	t.Run("Autocomplete Suggestions Event", func(t *testing.T) {
		// Test suggestions API (another client-server event)
		suggestURL := fmt.Sprintf("%s/api/suggest?q=%s", server.URL, url.QueryEscape("Gori"))
		resp, err := http.Get(suggestURL)
		if err != nil {
			t.Fatalf("Failed to get suggestions: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for suggestions, got %d", resp.StatusCode)
		}

		var suggestResponse handlers.SuggestResponse
		if err := json.NewDecoder(resp.Body).Decode(&suggestResponse); err != nil {
			t.Fatalf("Failed to decode suggest response: %v", err)
		}

		if len(suggestResponse.Suggestions) == 0 {
			t.Error("Suggestions should return results for 'Gori'")
		}

		t.Logf("✓ Autocomplete event working: got %d suggestions for '%s'", 
			len(suggestResponse.Suggestions), suggestResponse.Query)
	})

	t.Run("Data Refresh Event", func(t *testing.T) {
		// Test refresh API (POST request - client-server event)
		resp, err := http.Post(server.URL+"/api/refresh", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to refresh data: %v", err)
		}
		defer resp.Body.Close()

		// Should accept the request (implementation may vary)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status 200 or 202 for refresh, got %d", resp.StatusCode)
		}

		t.Logf("✓ Data refresh event responds with status %d", resp.StatusCode)
	})
}

// testErrorHandling tests error scenarios
func testErrorHandling(t *testing.T, server *httptest.Server) {
	testCases := []struct {
		name         string
		url          string
		expectedCode int
		description  string
	}{
		{
			name:         "404 for Non-existent Artist",
			url:          "/artists/99999",
			expectedCode: http.StatusNotFound,
			description:  "Should return 404 for non-existent artist",
		},
		{
			name:         "400 for Invalid Artist ID",
			url:          "/artists/invalid",
			expectedCode: http.StatusBadRequest,
			description:  "Should return 400 for invalid artist ID format",
		},
		{
			name:         "404 for Invalid Path",
			url:          "/invalid-path",
			expectedCode: http.StatusNotFound,
			description:  "Should return 404 for non-existent paths",
		},
		{
			name:         "404 for Invalid API Endpoint",
			url:          "/api/invalid",
			expectedCode: http.StatusNotFound,
			description:  "Should return 404 for invalid API endpoints",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(server.URL + tc.url)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", tc.url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status %d for %s, got %d", tc.expectedCode, tc.url, resp.StatusCode)
			}

			t.Logf("✓ %s: got expected status %d", tc.description, resp.StatusCode)
		})
	}
}

// testAPIFunctionality tests all API endpoints
func testAPIFunctionality(t *testing.T, server *httptest.Server) {
	t.Run("Health Check API", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/healthz")
		if err != nil {
			t.Fatalf("Failed to get health check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for health check, got %d", resp.StatusCode)
		}

		var healthResponse handlers.HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		if healthResponse.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", healthResponse.Status)
		}

		t.Logf("✓ Health check working: %s", healthResponse.Status)
	})

	t.Run("Search API Edge Cases", func(t *testing.T) {
		testCases := []struct {
			query string
			desc  string
		}{
			{"", "empty query"},
			{"xyz123nonexistent", "non-existent query"},
			{"a", "single character"},
			{"queen", "lowercase"},
			{"QUEEN", "uppercase"},
			{"Queen Member", "multi-word"},
		}

		for _, tc := range testCases {
			searchURL := fmt.Sprintf("%s/api/search?q=%s", server.URL, url.QueryEscape(tc.query))
			resp, err := http.Get(searchURL)
			if err != nil {
				t.Fatalf("Failed to search for '%s': %v", tc.query, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for search '%s', got %d", tc.query, resp.StatusCode)
			}

			var searchResponse handlers.SearchResponse
			if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
				t.Fatalf("Failed to decode search response for '%s': %v", tc.query, err)
			}

			t.Logf("✓ Search API handles %s: %d results", tc.desc, len(searchResponse.Artists))
		}
	})
}

// testPerformanceStability tests server stability and performance
func testPerformanceStability(t *testing.T, server *httptest.Server) {
	t.Run("Concurrent Request Handling", func(t *testing.T) {
		const numRequests = 20
		const timeoutSeconds = 10

		done := make(chan struct{}, numRequests)
		errors := make(chan error, numRequests)

		endpoints := []string{
			"/",
			"/artists",
			"/artists/1",
			"/locations",
			"/api/search?q=test",
			"/api/suggest?q=te",
			"/healthz",
		}

		start := time.Now()

		// Launch concurrent requests
		for i := 0; i < numRequests; i++ {
			go func(requestID int) {
				defer func() {
					done <- struct{}{}
				}()

				endpoint := endpoints[requestID%len(endpoints)]
				resp, err := http.Get(server.URL + endpoint)
				if err != nil {
					errors <- fmt.Errorf("request %d to %s failed: %v", requestID, endpoint, err)
					return
				}
				defer resp.Body.Close()

				// Verify we get a valid HTTP response
				if resp.StatusCode < 200 || resp.StatusCode >= 600 {
					errors <- fmt.Errorf("request %d got invalid status %d", requestID, resp.StatusCode)
					return
				}
			}(i)
		}

		// Wait for all requests to complete or timeout
		completedRequests := 0
		timeout := time.After(timeoutSeconds * time.Second)

		for completedRequests < numRequests {
			select {
			case <-done:
				completedRequests++
			case err := <-errors:
				t.Errorf("Concurrent request error: %v", err)
			case <-timeout:
				t.Fatalf("Timeout: only %d/%d requests completed", completedRequests, numRequests)
			}
		}

		duration := time.Since(start)
		t.Logf("✓ Handled %d concurrent requests in %v (%.2f req/sec)", 
			numRequests, duration, float64(numRequests)/duration.Seconds())

		// Performance check: should handle requests reasonably quickly
		if duration > 5*time.Second {
			t.Errorf("Performance concern: %d requests took %v (>5s)", numRequests, duration)
		}
	})

	t.Run("Memory Stability", func(t *testing.T) {
		// Make many requests to check for memory leaks
		for i := 0; i < 50; i++ {
			resp, err := http.Get(server.URL + "/api/search?q=test")
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
			resp.Body.Close() // Important: close response bodies
		}
		t.Logf("✓ Made 50 requests without issues (basic memory stability)")
	})
}

// testHTTPMethodsAndStatusCodes tests proper HTTP method handling
func testHTTPMethodsAndStatusCodes(t *testing.T, server *httptest.Server) {
	t.Run("Method Validation", func(t *testing.T) {
		testCases := []struct {
			method       string
			url          string
			expectedCode int
			description  string
		}{
			{"GET", "/", http.StatusOK, "GET home should work"},
			{"POST", "/", http.StatusMethodNotAllowed, "POST home should be rejected"},
			{"PUT", "/artists", http.StatusMethodNotAllowed, "PUT artists should be rejected"},
			{"DELETE", "/api/search", http.StatusMethodNotAllowed, "DELETE search should be rejected"},
			{"GET", "/api/search?q=test", http.StatusOK, "GET search should work"},
			{"POST", "/api/refresh", http.StatusOK, "POST refresh should work"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%s %s", tc.method, tc.url), func(t *testing.T) {
				req, err := http.NewRequest(tc.method, server.URL+tc.url, nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("Failed to execute request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tc.expectedCode {
					t.Errorf("Expected status %d for %s %s, got %d", 
						tc.expectedCode, tc.method, tc.url, resp.StatusCode)
				}

				t.Logf("✓ %s", tc.description)
			})
		}
	})
}

// Helper functions

func setupE2EStore(t *testing.T) *storage.Store {
	store := storage.NewStore()
	client := api.NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := client.FetchAllData(ctx)
	if err != nil {
		t.Fatalf("Failed to load data from API: %v", err)
	}

	storeData := storage.StoreData{
		Artists:   data.Artists,
		Locations: data.Locations,
		Dates:     data.Dates,
		Relations: data.Relations,
	}
	store.LoadData(storeData)

	return store
}

func createE2ERouter(h *handlers.Handlers) http.Handler {
	mux := http.NewServeMux()

	// Static file serving
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Web routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			h.NotFoundHandler(w, r)
			return
		}
		h.HomeHandler(w, r)
	})
	mux.HandleFunc("/artists", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/artists" {
			h.NotFoundHandler(w, r)
			return
		}
		h.ArtistsHandler(w, r)
	})
	mux.HandleFunc("/artists/", h.ArtistDetailHandler)
	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/locations" {
			h.NotFoundHandler(w, r)
			return
		}
		h.LocationsHandler(w, r)
	})

	// API routes
	mux.HandleFunc("/api/search", h.SearchHandler)
	mux.HandleFunc("/api/suggest", h.SuggestHandler)
	mux.HandleFunc("/api/refresh", h.RefreshHandler)
	mux.HandleFunc("/healthz", h.HealthHandler)

	return mux
}
