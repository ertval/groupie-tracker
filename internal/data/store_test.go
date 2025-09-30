package data

import (
	"context"
	"testing"
	"time"
)

func TestStoreBasic(t *testing.T) {
	store := NewStore()

	// Test initialization
	if store == nil {
		t.Fatal("Store should not be nil")
	}

	// Test initial state
	artists := store.Artists()
	if len(artists) != 0 {
		t.Error("Store should start with 0 artists before loading data")
	}

	locations := store.Locations()
	if len(locations) != 0 {
		t.Error("Store should start with 0 locations before loading data")
	}
}

func TestStoreLoadData(t *testing.T) {
	store := NewStore()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := store.LoadData(ctx)
	if err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	// Verify data was loaded
	artists := store.Artists()
	if len(artists) == 0 {
		t.Error("Expected at least one artist after loading data")
	}

	locations := store.Locations()
	if len(locations) == 0 {
		t.Error("Expected at least one location after loading data")
	}

	stats := store.Stats()
	if stats.TotalArtists == 0 {
		t.Error("Expected non-zero artist count in stats")
	}
}
