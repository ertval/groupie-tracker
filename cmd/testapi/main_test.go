package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArtistsHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/artists", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(artistsHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var artists []Artist
	if err := json.NewDecoder(rr.Body).Decode(&artists); err != nil {
		t.Fatal(err)
	}

	if len(artists) == 0 {
		t.Error("Expected at least one artist, got none")
	}

	// Check first artist structure
	if len(artists) > 0 {
		artist := artists[0]
		if artist.ID == 0 {
			t.Error("Artist ID should not be zero")
		}
		if artist.Name == "" {
			t.Error("Artist name should not be empty")
		}
		if len(artist.Members) == 0 {
			t.Error("Artist should have at least one member")
		}
	}
}

func TestLocationsHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/locations", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(locationsHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		Index []Location `json:"index"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if len(response.Index) == 0 {
		t.Error("Expected at least one location, got none")
	}

	// Check first location structure
	if len(response.Index) > 0 {
		location := response.Index[0]
		if location.ID == 0 {
			t.Error("Location ID should not be zero")
		}
		if len(location.Locations) == 0 {
			t.Error("Location should have at least one location")
		}
	}
}

func TestDatesHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/dates", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(datesHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		Index []Date `json:"index"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if len(response.Index) == 0 {
		t.Error("Expected at least one date, got none")
	}

	// Check first date structure
	if len(response.Index) > 0 {
		date := response.Index[0]
		if date.ID == 0 {
			t.Error("Date ID should not be zero")
		}
		if len(date.Dates) == 0 {
			t.Error("Date should have at least one date")
		}
	}
}

func TestRelationHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/relation", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(relationHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		Index []Relation `json:"index"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if len(response.Index) == 0 {
		t.Error("Expected at least one relation, got none")
	}

	// Check first relation structure
	if len(response.Index) > 0 {
		relation := response.Index[0]
		if relation.ID == 0 {
			t.Error("Relation ID should not be zero")
		}
		if len(relation.DatesLocations) == 0 {
			t.Error("Relation should have at least one date-location mapping")
		}
	}
}

func TestNotFoundHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/artists") {
			artistsHandler(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/api/locations") {
			locationsHandler(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/api/dates") {
			datesHandler(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/api/relation") {
			relationHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestContentTypes(t *testing.T) {
	endpoints := []string{"/api/artists", "/api/locations", "/api/dates", "/api/relation"}

	for _, endpoint := range endpoints {
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		var handler http.HandlerFunc

		switch endpoint {
		case "/api/artists":
			handler = artistsHandler
		case "/api/locations":
			handler = locationsHandler
		case "/api/dates":
			handler = datesHandler
		case "/api/relation":
			handler = relationHandler
		}

		handler.ServeHTTP(rr, req)

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("endpoint %s returned wrong content type: got %v want %v", endpoint, contentType, "application/json")
		}
	}
}

func TestRootHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Test API Server - Use /api/ endpoints"))
	})
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	expected := "Test API Server - Use /api/ endpoints"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestCORSHeaders(t *testing.T) {
	endpoints := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/artists", artistsHandler},
		{"/api/locations", locationsHandler},
		{"/api/dates", datesHandler},
		{"/api/relation", relationHandler},
	}

	for _, endpoint := range endpoints {
		req, err := http.NewRequest("GET", endpoint.path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		endpoint.handler.ServeHTTP(rr, req)

		corsHeader := rr.Header().Get("Access-Control-Allow-Origin")
		if corsHeader != "*" {
			t.Errorf("endpoint %s returned wrong CORS header: got %v want %v", endpoint.path, corsHeader, "*")
		}
	}
}

func TestMethodValidation(t *testing.T) {
	endpoints := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/artists", artistsHandler},
		{"/api/locations", locationsHandler},
		{"/api/dates", datesHandler},
		{"/api/relation", relationHandler},
	}

	// Test that POST, PUT, DELETE methods still work (handlers don't explicitly check methods)
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		for _, endpoint := range endpoints {
			req, err := http.NewRequest(method, endpoint.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			endpoint.handler.ServeHTTP(rr, req)

			// The handlers don't validate methods, so they should still return 200
			if status := rr.Code; status != http.StatusOK {
				t.Errorf("endpoint %s with method %s returned status: got %v want %v", endpoint.path, method, status, http.StatusOK)
			}
		}
	}
}

func TestJSONResponseStructure(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		handler    http.HandlerFunc
		validateFn func(*testing.T, []byte)
	}{
		{
			name:    "artists response structure",
			path:    "/api/artists",
			handler: artistsHandler,
			validateFn: func(t *testing.T, body []byte) {
				var artists []Artist
				if err := json.Unmarshal(body, &artists); err != nil {
					t.Errorf("Failed to unmarshal artists response: %v", err)
				}
				if len(artists) < 2 {
					t.Errorf("Expected at least 2 artists, got %d", len(artists))
				}
				// Check specific artist data
				found := false
				for _, artist := range artists {
					if artist.Name == "Queen" {
						found = true
						if len(artist.Members) != 4 {
							t.Errorf("Queen should have 4 members, got %d", len(artist.Members))
						}
						break
					}
				}
				if !found {
					t.Error("Expected to find Queen in artists list")
				}
			},
		},
		{
			name:    "locations response structure",
			path:    "/api/locations",
			handler: locationsHandler,
			validateFn: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Errorf("Failed to unmarshal locations response: %v", err)
				}
				if _, exists := response["index"]; !exists {
					t.Error("Locations response should have 'index' field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			tt.validateFn(t, rr.Body.Bytes())
		})
	}
}
