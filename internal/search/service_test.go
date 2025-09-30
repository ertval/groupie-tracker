package search

import (
	"context"
	"testing"

	"groupie-tracker/internal/data"
	"groupie-tracker/internal/testsupport"
)

func TestServiceSearchAndSuggestions(t *testing.T) {
	store, err := data.Load(context.Background(), testsupport.MinimalDataset())
	if err != nil {
		t.Fatalf("failed to build store: %v", err)
	}

	svc := NewService(store)

	result := svc.Search(Params{Query: "example"})
	if result.TotalResults != 1 {
		t.Fatalf("query mismatch: got %d results", result.TotalResults)
	}

	filters := Filters{CreationYearMin: 2005}
	filtered := svc.Search(Params{Filters: filters})
	if filtered.TotalResults != 1 {
		t.Fatalf("filters reduced results incorrectly: %d", filtered.TotalResults)
	}

	suggestions := svc.Suggest("exa", 5)
	if len(suggestions) == 0 {
		t.Fatalf("expected suggestions for prefix")
	}

	options := svc.FilterOptions()
	if options.CreationYearMin == 0 || options.CreationYearMax == 0 {
		t.Fatalf("expected filter options to be populated")
	}
}
