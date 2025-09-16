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
