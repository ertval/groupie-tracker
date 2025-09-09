package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

func TestSimplifiedArchitectureIntegration(t *testing.T) {
	// Create simplified store
	store := storage.NewSimplifiedStore()

	// Create test data
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970, Members: []string{"Freddie Mercury"}, Slug: "queen"},
			{ID: 2, Name: "Beatles", CreationYear: 1960, Members: []string{"John Lennon", "Paul McCartney"}, Slug: "beatles"},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk", "liverpool-uk"}},
			{ID: 2, Locations: []string{"new_york-usa", "los_angeles-usa"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"01-01-2020", "02-01-2020"}},
			{ID: 2, Dates: []string{"15-06-2020", "20-06-2020"}},
		},
		Relations: []models.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"london-uk":    {"01-01-2020"},
					"liverpool-uk": {"02-01-2020"},
				},
			},
			{
				ID: 2,
				DatesLocations: map[string][]string{
					"new_york-usa":    {"15-06-2020"},
					"los_angeles-usa": {"20-06-2020"},
				},
			},
		},
	}

	// Load data into store
	store.LoadData(testData)

	// Create simplified service
	service := service.NewSimplifiedService(store)

	// Create simplified handlers with mock API client
	mockClient := api.NewClient("http://mock", 30)
	h := handlers.NewSimplifiedHandlers(store, mockClient)

	// Test that handlers are properly initialized
	if h == nil {
		t.Fatal("Expected handlers to be initialized")
	}

	// Test that service functions work
	locations := service.CalculateLocationStats()
	if len(locations) != 4 { // london-uk, liverpool-uk, new_york-usa, los_angeles-usa
		t.Errorf("Expected 4 locations, got %d", len(locations))
	}

	// Test that store search works
	results := store.SearchArtists("Queen")
	if len(results) != 1 {
		t.Errorf("Expected 1 search result for 'Queen', got %d", len(results))
	}

	// Test HTTP handlers
	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{"home page", "GET", "/", http.StatusOK},
		{"artists page", "GET", "/artists", http.StatusOK},
		{"artist by slug", "GET", "/artists/queen", http.StatusOK},
		{"locations page", "GET", "/locations", http.StatusOK},
		{"search API", "GET", "/api/search?q=Queen", http.StatusOK},
		{"suggest API", "GET", "/api/suggest?q=Que", http.StatusOK},
		{"health check", "GET", "/healthz", http.StatusOK},
		{"not found", "GET", "/nonexistent", http.StatusNotFound},
	}

	// Create router with simplified handlers
	mux := createRouter(h)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}
		})
	}
}

func TestSimplifiedHandlersLocationsSortingFixed(t *testing.T) {
	// Create simplified store
	store := storage.NewSimplifiedStore()

	// Create test data with specific concert counts to test sorting
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970},
			{ID: 2, Name: "Beatles", CreationYear: 1960},
		},
		Relations: []models.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"london-uk":     {"01-01-2020", "02-01-2020", "03-01-2020"}, // 3 concerts
					"liverpool-uk":  {"04-01-2020"},                             // 1 concert
					"manchester-uk": {"05-01-2020", "06-01-2020"},               // 2 concerts
				},
			},
			{
				ID: 2,
				DatesLocations: map[string][]string{
					"london-uk":     {"15-06-2020"}, // +1 = 4 total concerts
					"birmingham-uk": {"20-06-2020"}, // 1 concert
				},
			},
		},
	}

	// Load data into store
	store.LoadData(testData)

	// Create simplified service
	service := service.NewSimplifiedService(store)

	// Calculate and sort location stats
	locationStats := service.CalculateLocationStats()
	sortedStats := service.SortLocationStatsByConcertCount(locationStats)

	// Verify sorting: london-uk should be first with 4 concerts
	if len(sortedStats) == 0 {
		t.Fatal("Expected location stats to be calculated")
	}

	if sortedStats[0].Name != "london-uk" {
		t.Errorf("Expected london-uk to be most popular, got %s", sortedStats[0].Name)
	}

	if sortedStats[0].ConcertCount != 4 {
		t.Errorf("Expected london-uk to have 4 concerts, got %d", sortedStats[0].ConcertCount)
	}

	// Verify the second most popular location
	if len(sortedStats) > 1 && sortedStats[1].Name != "manchester-uk" {
		t.Errorf("Expected manchester-uk to be second most popular, got %s", sortedStats[1].Name)
	}

	t.Logf("✅ Location sorting fixed: %s has %d concerts (most popular)",
		sortedStats[0].Name, sortedStats[0].ConcertCount)
}
