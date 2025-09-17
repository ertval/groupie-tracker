package tests

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/data"
)

func TestAuditCompliance(t *testing.T) {
	store := data.NewRepository("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := store.LoadData(ctx)
	if err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	artists := store.GetArtists()
	if len(artists) == 0 {
		t.Error("No artists loaded")
	}
}
