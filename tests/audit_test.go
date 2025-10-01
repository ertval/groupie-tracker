package tests

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
)

func TestAuditCompliance(t *testing.T) {
	// Disable caching in tests to avoid creating files in CI/workspaces
	config.WithCache = false
	// Use configured API base URL and timeout for the store
	config.APIBaseURL = "https://groupietrackers.herokuapp.com"
	config.APIRequestTimeout = 30 * time.Second
	store := data.NewStore()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := store.LoadData(ctx); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	artists := store.Artists()
	if len(artists) == 0 {
		t.Error("No artists loaded")
	}
}
