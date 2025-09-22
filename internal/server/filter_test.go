package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"groupie-tracker/internal/data"
)

// TestFilterArtists tests the filter artists API endpoint
func TestFilterArtists(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name         string
		method       string
		body         interface{}
		wantStatus   int
		wantResponse bool // true if we expect a valid JSON response
	}{
		{
			name:   "Valid filter request",
			method: "POST",
			body: data.FilterParams{
				CreationYearFrom: intPtr(1990),
				CreationYearTo:   intPtr(2000),
			},
			wantStatus:   http.StatusOK,
			wantResponse: true,
		},
		{
			name:         "Empty filter request",
			method:       "POST",
			body:         data.FilterParams{},
			wantStatus:   http.StatusOK,
			wantResponse: true,
		},
		{
			name:       "GET method not allowed",
			method:     "GET",
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT method not allowed",
			method:     "PUT",
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Invalid JSON body",
			method:     "POST",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.body != nil {
				var bodyBytes []byte
				if str, ok := tt.body.(string); ok {
					bodyBytes = []byte(str)
				} else {
					bodyBytes, err = json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("Failed to marshal body: %v", err)
					}
				}
				req = httptest.NewRequest(tt.method, "/api/filter-artists", bytes.NewReader(bodyBytes))
			} else {
				req = httptest.NewRequest(tt.method, "/api/filter-artists", nil)
			}

			w := httptest.NewRecorder()
			app.FilterArtists(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("FilterArtists() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if tt.wantResponse && w.Code == http.StatusOK {
				// Check that we get valid JSON response
				var artists []data.Artist
				if err := json.NewDecoder(w.Body).Decode(&artists); err != nil {
					t.Errorf("FilterArtists() returned invalid JSON: %v", err)
				}
			}
		})
	}
}

// TestFilterOptions tests the filter options API endpoint
func TestFilterOptions(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "Valid GET request",
			method:     "GET",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST method not allowed",
			method:     "POST",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT method not allowed",
			method:     "PUT",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/filter-options", nil)
			w := httptest.NewRecorder()

			app.FilterOptions(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("FilterOptions() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if w.Code == http.StatusOK {
				// Check that we get valid JSON response
				var options data.FilterOptions
				if err := json.NewDecoder(w.Body).Decode(&options); err != nil {
					t.Errorf("FilterOptions() returned invalid JSON: %v", err)
				}

				// Basic validation of the response structure
				// Note: With minimal test data, some ranges might be 0, so we just check the structure
				if options.Countries == nil {
					t.Error("FilterOptions() should return countries array (even if empty)")
				}
				if options.MemberCounts == nil {
					t.Error("FilterOptions() should return memberCounts array (even if empty)")
				}
			}
		})
	}
}

// TestArtistsWithFilters tests that the Artists handler includes filter options
func TestArtistsWithFilters(t *testing.T) {
	t.Skip("Skipping template rendering test - templates not available in test environment")
	app := newTestApplication(t)

	req := httptest.NewRequest("GET", "/artists", nil)
	w := httptest.NewRecorder()

	app.Artists(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Artists() status = %v, want %v", w.Code, http.StatusOK)
	}

	// The response should contain the filter options in the template data
	// Since we can't easily test template rendering, we just ensure it doesn't crash
	// and returns 200 OK
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}
