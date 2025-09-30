package tests

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/data"
	"groupie-tracker/internal/testsupport"
)

func TestAuditCompliance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := data.Load(ctx, testsupport.MinimalDataset())
	if err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	artists := store.Artists()
	if len(artists) == 0 {
		t.Error("No artists loaded")
	}
}
