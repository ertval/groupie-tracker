package tests

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/conf"
	"groupie-tracker/internal/data"
)

// TestAuditCompliance tests that the data layer loads successfully from the external API
func TestAuditCompliance(t *testing.T) {
	// Disable caching in tests
	conf.WithCache = false
	conf.APIBaseURL = "https://groupietrackers.herokuapp.com"
	conf.APIRequestTimeout = 30 * time.Second

	apiClient := api.NewClient(conf.APIBaseURL, conf.APIRequestTimeout)
	store := data.NewStore(apiClient, conf.WithCache)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := store.Load(ctx); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	artists := store.Artists()
	if len(artists) == 0 {
		t.Error("No artists loaded")
	}

	t.Logf("Successfully loaded %d artists from external API", len(artists))
}

// Note: Browser-based E2E tests (playwright_test.go, visual_e2e_test.go) are kept
// in separate files as they require external dependencies and manual server startup.
// They test visual aspects and browser automation that complement the HTTP-level
// E2E tests in e2e_test.go.
