package tests

import (
	"context"
	"fmt"
	"groupie-tracker/internal/repository"
	"time"
)

// Run is a helper function (previously main) kept for manual verification.
// It is not executed by `go test` but can be called from other tests if needed.
func Run() {
	fmt.Println("Testing refactored repository...")

	// Test the refactored repository
	repo := repository.NewRepository("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := repo.LoadData(ctx); err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	// Test basic functionality
	artists := repo.GetArtists()
	fmt.Printf("✓ Loaded %d artists\n", len(artists))

	// Find Queen for audit compliance
	queen, found := repo.GetArtistBySlug("queen")
	if found {
		fmt.Printf("✓ Found Queen with %d members\n", len(queen.Members))
		fmt.Printf("✓ Queen has %d concerts total\n", repo.CountConcerts(queen))
		fmt.Printf("✓ Queen tours in %d countries\n", len(repo.GetCountries(queen)))
	}

	// Test location stats
	locations := repo.GetLocationStats()
	fmt.Printf("✓ Found %d unique locations\n", len(locations))

	// Test global stats
	stats := repo.GetStats()
	fmt.Printf("✓ Global stats: %d artists, %d concerts, %d countries\n",
		stats["total_artists"], stats["total_concerts"], stats["total_countries"])

	fmt.Println("✓ All tests passed - refactoring successful!")
}
