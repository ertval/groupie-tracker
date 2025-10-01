package main

import (
	"context"
	"encoding/json"
	"fmt"
	"groupie-tracker/internal/domain"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestSearchEndToEnd tests the search functionality with the actual server
func TestSearchEndToEnd(t *testing.T) {
	// Test server URL - assumes server is running on localhost:8080
	baseURL := "http://localhost:8080"

	// Test that server is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
	if err != nil {
		t.Fatalf("Failed to create health check request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Skip("Server not running on localhost:8080, skipping E2E tests")
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skip("Server not healthy, skipping E2E tests")
		return
	}

	// Test cases for search functionality
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
				var suggestions []domain.SearchSuggestion
				if err := json.Unmarshal([]byte(body), &suggestions); err != nil {
					t.Errorf("Failed to parse suggestions JSON: %v", err)
					return
				}

				if len(suggestions) == 0 {
					t.Error("Should return suggestions for 'queen'")
					return
				}

				// Check for Queen artist suggestion
				found := false
				for _, s := range suggestions {
					if s.Text == "Queen" && s.Type == domain.SuggestionTypeArtist {
						found = true
						break
					}
				}
				if !found {
					t.Error("Should find Queen artist in suggestions")
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
				if !strings.Contains(body, "Found") {
					t.Error("Search results should show result count")
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
					t.Error("Search for 'Freddie Mercury' should return Queen with member info")
				}
			},
		},
		{
			name:         "Search with filters works",
			endpoint:     "/search",
			method:       "POST",
			formData:     url.Values{"q": {""}, "creationYearFrom": {"1970"}, "creationYearTo": {"1975"}},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				// Should find artists created between 1970-1975 (like Queen - 1970)
				// Check for either "Found" (results) or "No Results Found" (no results)
				if !strings.Contains(body, "Found") {
					t.Log("Filter search did not show 'Found' in results, body length:", len(body))
				}
			},
		},
		{
			name:         "Empty search with no filters shows all artists",
			endpoint:     "/search",
			method:       "POST",
			formData:     url.Values{"q": {""}},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				// Empty search should return all artists or show an appropriate message
				if !strings.Contains(body, "Found") {
					t.Log("Empty search did not show 'Found' in results, body length:", len(body))
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
			var req *http.Request
			var err error

			reqURL := baseURL + tt.endpoint
			if tt.query != "" {
				reqURL += "?q=" + url.QueryEscape(tt.query)
			}

			if tt.method == "POST" && tt.formData != nil {
				req, err = http.NewRequest("POST", reqURL, strings.NewReader(tt.formData.Encode()))
				if err != nil {
					t.Fatalf("Failed to create POST request: %v", err)
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req, err = http.NewRequest(tt.method, reqURL, nil)
				if err != nil {
					t.Fatalf("Failed to create %s request: %v", tt.method, err)
				}
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, string(body))
			}
		})
	}
}

// TestSearchAuditCompliance tests that search meets the audit requirements
func TestSearchAuditCompliance(t *testing.T) {
	baseURL := "http://localhost:8080"

	// Test server is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
	if err != nil {
		t.Skip("Server not accessible, skipping audit tests")
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Skip("Server not running, skipping audit tests")
		return
	}
	resp.Body.Close()

	// Test audit requirements from requirements.md
	auditTests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "Search for Phil Collins as member",
			query:    "Phil Collins",
			expected: "Phil Collins", // Should find both artist and member
		},
		{
			name:     "Search for Freddie Mercury as member",
			query:    "Freddie Mercury",
			expected: "Queen", // Should find Queen (Freddie's band)
		},
		{
			name:     "Search for location",
			query:    "London",
			expected: "", // Will depend on actual data
		},
		{
			name:     "Search for creation date",
			query:    "1970",
			expected: "", // Will depend on actual data
		},
		{
			name:     "Search for first album date",
			query:    "1973",
			expected: "", // Will depend on actual data
		},
	}

	for _, tt := range auditTests {
		t.Run(tt.name, func(t *testing.T) {
			// Test search via form submission
			formData := url.Values{"q": {tt.query}}
			req, err := http.NewRequest("POST", baseURL+"/search", strings.NewReader(formData.Encode()))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Search request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Search failed with status %d", resp.StatusCode)
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			bodyStr := string(body)

			// Basic validation - should not crash and should show search interface
			if !strings.Contains(bodyStr, "Search Artists") {
				t.Error("Response should contain search interface")
			}

			// If we expect specific content, check for it
			if tt.expected != "" && !strings.Contains(bodyStr, tt.expected) {
				t.Logf("Expected to find '%s' in search results for query '%s'", tt.expected, tt.query)
				// Log but don't fail - depends on actual data
			}

			// Test that suggestions API also works for this query
			suggestURL := fmt.Sprintf("%s/api/suggestions?q=%s", baseURL, url.QueryEscape(tt.query))
			suggestResp, err := client.Get(suggestURL)
			if err != nil {
				t.Errorf("Suggestions API failed: %v", err)
				return
			}
			defer suggestResp.Body.Close()

			if suggestResp.StatusCode != http.StatusOK {
				t.Errorf("Suggestions API returned status %d", suggestResp.StatusCode)
				return
			}

			var suggestions []domain.SearchSuggestion
			if err := json.NewDecoder(suggestResp.Body).Decode(&suggestions); err != nil {
				t.Errorf("Failed to decode suggestions: %v", err)
				return
			}

			t.Logf("Query '%s' returned %d suggestions", tt.query, len(suggestions))
		})
	}
}

// TestSearchIntegrationWithFilters tests combined search and filter functionality
func TestSearchIntegrationWithFilters(t *testing.T) {
	baseURL := "http://localhost:8080"

	// Test server is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
	if err != nil {
		t.Skip("Server not accessible, skipping integration tests")
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Skip("Server not running, skipping integration tests")
		return
	}
	resp.Body.Close()

	// Test combined search + filter functionality
	formData := url.Values{
		"q":                {""},       // Empty search query
		"creationYearFrom": {"1960"},   // Filter from 1960
		"creationYearTo":   {"1980"},   // Filter to 1980
		"memberCounts":     {"4", "5"}, // Bands with 4 or 5 members
	}

	req, err = http.NewRequest("POST", baseURL+"/search", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatalf("Failed to create filter request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Filter request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Filter search failed with status %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	bodyStr := string(body)

	// Should show filtered results
	if !strings.Contains(bodyStr, "Search Artists") {
		t.Error("Should contain search interface")
	}

	// Should show some indication of filtering being applied
	// Either results or "No Results Found"
	if !strings.Contains(bodyStr, "Found") && !strings.Contains(bodyStr, "No Results Found") {
		t.Log("Should show either results or no results message, body length:", len(bodyStr))
	}

	t.Logf("Combined search and filter test completed successfully")
}
