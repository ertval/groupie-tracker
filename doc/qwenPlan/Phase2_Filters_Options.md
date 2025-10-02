# Phase 2: Filters & Options Simplification - Implementation Guide

## Overview
This phase focuses on simplifying the filtering system by implementing a functional approach with predicate-based helpers, reusable filter builders, and simplified data structures. The goal is to create a more flexible and maintainable filtering system.

## Step-by-Step Implementation

### Step 1: Define Functional Filter Types
**File to create:** `internal/data/filter_types.go` (or add to existing models.go)

```go
// FilterFunc defines a function type for filtering artists
type FilterFunc func(*Artist) bool

// LocationFilterFunc defines a function type for filtering locations
type LocationFilterFunc func(Location) bool

// Combined filter helpers for AND semantics
func AndFilters(filters ...FilterFunc) FilterFunc {
	return func(artist *Artist) bool {
		for _, filter := range filters {
			if !filter(artist) {
				return false
			}
		}
		return true
	}
}

func AndLocationFilters(filters ...LocationFilterFunc) LocationFilterFunc {
	return func(location Location) bool {
		for _, filter := range filters {
			if !filter(location) {
				return false
			}
		}
		return true
	}
}
```

### Step 2: Create Range Filter Helper
**File to create:** `internal/data/range_filter.go`

```go
// RangeFilter represents a range filter for numeric values
type RangeFilter[T comparable] struct {
	Min *T
	Max *T
}

// Matches checks if a value falls within the range
func (r RangeFilter[T]) Matches(value T) bool {
	if r.Min != nil {
		min := *r.Min
		if compare(min, value) > 0 {
			return false
		}
	}
	if r.Max != nil {
		max := *r.Max
		if compare(value, max) > 0 {
			return false
		}
	}
	return true
}

// Helper function for numeric comparison
func compare[T ~int | ~int32 | ~int64 | ~float32 | ~float64](a, b T) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}
```

### Step 3: Create Set/Collection Filter Helpers
**File to create:** `internal/data/set_filters.go`

```go
// ContainsFilter provides utility for checking if a value exists in a collection
type ContainsFilter[T comparable] struct {
	Values map[T]bool
	EmptyOK bool // If true, empty Values means match all
}

// NewContainsFilter creates a new ContainsFilter from a slice
func NewContainsFilter[T comparable](values []T) ContainsFilter[T] {
	if len(values) == 0 {
		return ContainsFilter[T]{Values: make(map[T]bool), EmptyOK: true}
	}
	
	valuesMap := make(map[T]bool, len(values))
	for _, v := range values {
		valuesMap[v] = true
	}
	return ContainsFilter[T]{Values: valuesMap, EmptyOK: false}
}

// Contains checks if a value exists in the filter's collection
func (cf ContainsFilter[T]) Contains(value T) bool {
	if cf.EmptyOK {
		return true
	}
	return cf.Values[value]
}

// StringSetFilter provides a convenient set filter for string collections
type StringSetFilter = ContainsFilter[string]

// IntSetFilter provides a convenient set filter for int collections  
type IntSetFilter = ContainsFilter[int]
```

### Step 4: Simplify Filter Parameter Structures
**File to modify:** `internal/data/models.go`

```go
// Simplified ArtistFilterParams with direct field values instead of pointers
type ArtistFilterParams struct {
	CreationYear RangeFilter[int]    `json:"creationYear,omitempty"`
	FirstAlbumYear RangeFilter[int]  `json:"firstAlbumYear,omitempty"`
	MemberCounts []int               `json:"memberCounts,omitempty"`
	Countries    []string            `json:"countries,omitempty"`
}

// Simplified LocationFilterParams
type LocationFilterParams struct {
	ConcertCount RangeFilter[int]    `json:"concertCount,omitempty"`
	ArtistCount  RangeFilter[int]    `json:"artistCount,omitempty"`
	YearRange    RangeFilter[int]    `json:"yearRange,omitempty"`
	Countries    []string            `json:"countries,omitempty"`
}
```

### Step 5: Implement Filter Builder Functions
**File to create:** `internal/data/filter_builders.go`

```go
// Artist Filter Builder Functions
func CreationYearFilter(min, max *int) FilterFunc {
	var rangeFilter RangeFilter[int]
	if min != nil {
		rangeFilter.Min = min
	}
	if max != nil {
		rangeFilter.Max = max
	}
	return func(artist *Artist) bool {
		return rangeFilter.Matches(artist.CreationYear)
	}
}

func FirstAlbumYearFilter(min, max *int) FilterFunc {
	var rangeFilter RangeFilter[int]
	if min != nil {
		rangeFilter.Min = min
	}
	if max != nil {
		rangeFilter.Max = max
	}
	return func(artist *Artist) bool {
		albumYear := artist.FirstAlbumYear()
		if albumYear == 0 { // If no album year, only pass if filter doesn't care
			return rangeFilter.Min == nil && rangeFilter.Max == nil
		}
		return rangeFilter.Matches(albumYear)
	}
}

func MemberCountFilter(counts []int) FilterFunc {
	setFilter := NewContainsFilter(counts)
	return func(artist *Artist) bool {
		return setFilter.Contains(artist.MemberCount())
	}
}

func CountryFilter(countries []string) FilterFunc {
	setFilter := NewContainsFilter(countries)
	return func(artist *Artist) bool {
		artistCountries := artist.Countries()
		for _, country := range artistCountries {
			if setFilter.Contains(country) {
				return true
			}
		}
		return false // Artist must have performed in at least one of the specified countries
	}
}

// Location Filter Builder Functions
func ConcertCountFilter(min, max *int) LocationFilterFunc {
	var rangeFilter RangeFilter[int]
	if min != nil {
		rangeFilter.Min = min
	}
	if max != nil {
		rangeFilter.Max = max
	}
	return func(location Location) bool {
		return rangeFilter.Matches(location.TotalConcerts())
	}
}

func ArtistCountFilter(min, max *int) LocationFilterFunc {
	var rangeFilter RangeFilter[int]
	if min != nil {
		rangeFilter.Min = min
	}
	if max != nil {
		rangeFilter.Max = max
	}
	return func(location Location) bool {
		return rangeFilter.Matches(location.ArtistCount())
	}
}

func YearRangeFilter(min, max *int) LocationFilterFunc {
	var rangeFilter RangeFilter[int]
	if min != nil {
		rangeFilter.Min = min
	}
	if max != nil {
		rangeFilter.Max = max
	}
	return func(location Location) bool {
		earliest, latest := location.YearRange()
		// A location matches if its year range overlaps with the filter range
		if rangeFilter.Min != nil {
			if latest < *rangeFilter.Min {
				return false // Location's latest year is before filter's minimum
			}
		}
		if rangeFilter.Max != nil {
			if earliest > *rangeFilter.Max {
				return false // Location's earliest year is after filter's maximum
			}
		}
		return true
	}
}

func LocationCountryFilter(countries []string) LocationFilterFunc {
	setFilter := NewContainsFilter(countries)
	return func(location Location) bool {
		return setFilter.Contains(location.Country())
	}
}
```

### Step 6: Implement the FilterArtists Method with Functional Approach
**File to modify:** `internal/data/filters.go`

```go
// FilterArtists applies user-specified filter criteria to the artist collection and returns matching artists.
// Uses functional approach with predicate composition for flexible filtering.
func (s *Store) FilterArtists(params ArtistFilterParams) []*Artist {
	// Build filter functions from parameters
	var filters []FilterFunc
	
	if params.CreationYear.Min != nil || params.CreationYear.Max != nil {
		filters = append(filters, CreationYearFilter(params.CreationYear.Min, params.CreationYear.Max))
	}
	
	if params.FirstAlbumYear.Min != nil || params.FirstAlbumYear.Max != nil {
		filters = append(filters, FirstAlbumYearFilter(params.FirstAlbumYear.Min, params.FirstAlbumYear.Max))
	}
	
	if len(params.MemberCounts) > 0 {
		filters = append(filters, MemberCountFilter(params.MemberCounts))
	}
	
	if len(params.Countries) > 0 {
		filters = append(filters, CountryFilter(params.Countries))
	}
	
	// If no filters, return all artists
	if len(filters) == 0 {
		return s.Artists()
	}
	
	// Combine all filters with AND logic
	combinedFilter := AndFilters(filters...)
	
	// Apply the filter
	var result []*Artist
	for _, artist := range s.Artists() {
		if combinedFilter(artist) {
			result = append(result, artist)
		}
	}
	
	return result
}

// FilterLocations applies user-specified filter criteria to the location collection and returns matching locations.
// Uses functional approach with predicate composition for flexible filtering.
func (s *Store) FilterLocations(params LocationFilterParams) []Location {
	// Build filter functions from parameters
	var filters []LocationFilterFunc
	
	if params.ConcertCount.Min != nil || params.ConcertCount.Max != nil {
		filters = append(filters, ConcertCountFilter(params.ConcertCount.Min, params.ConcertCount.Max))
	}
	
	if params.ArtistCount.Min != nil || params.ArtistCount.Max != nil {
		filters = append(filters, ArtistCountFilter(params.ArtistCount.Min, params.ArtistCount.Max))
	}
	
	if params.YearRange.Min != nil || params.YearRange.Max != nil {
		filters = append(filters, YearRangeFilter(params.YearRange.Min, params.YearRange.Max))
	}
	
	if len(params.Countries) > 0 {
		filters = append(filters, LocationCountryFilter(params.Countries))
	}
	
	// If no filters, return all locations
	if len(filters) == 0 {
		return s.Locations()
	}
	
	// Combine all filters with AND logic
	combinedFilter := AndLocationFilters(filters...)
	
	// Apply the filter
	var result []Location
	for _, location := range s.Locations() {
		if combinedFilter(location) {
			result = append(result, location)
		}
	}
	
	return result
}
```

### Step 7: Simplify Filter Options Calculation
**File to modify:** `internal/data/filters.go`

```go
// calculateArtistFilterOptions derives available artist filter metadata from the dataset.
func (s *Store) calculateArtistFilterOptions(artists []*Artist) ArtistFilterOptions {
	if len(artists) == 0 {
		return ArtistFilterOptions{}
	}

	// Calculate min/max values in a single pass
	var (
		minCreationYear = artists[0].CreationYear
		maxCreationYear = artists[0].CreationYear
		minFirstAlbumYear = 0
		maxFirstAlbumYear = 0
	)
	
	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	for _, artist := range artists {
		if artist.CreationYear < minCreationYear {
			minCreationYear = artist.CreationYear
		}
		if artist.CreationYear > maxCreationYear {
			maxCreationYear = artist.CreationYear
		}

		albumYear := artist.FirstAlbumYear()
		if albumYear > 0 {
			if minFirstAlbumYear == 0 || albumYear < minFirstAlbumYear {
				minFirstAlbumYear = albumYear
			}
			if albumYear > maxFirstAlbumYear {
				maxFirstAlbumYear = albumYear
			}
		}

		memberCount := artist.MemberCount()
		memberCountSet[memberCount] = true

		for _, country := range artist.Countries() {
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	memberCounts := make([]int, 0, len(memberCountSet))
	for count := range memberCountSet {
		memberCounts = append(memberCounts, count)
	}
	sort.Ints(memberCounts)

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	if minFirstAlbumYear == 0 && minCreationYear > 0 {
		minFirstAlbumYear = minCreationYear
	}
	if maxFirstAlbumYear == 0 && maxCreationYear > 0 {
		maxFirstAlbumYear = maxCreationYear
	}

	return ArtistFilterOptions{
		CreationYearMin:   minCreationYear,
		CreationYearMax:   maxCreationYear,
		FirstAlbumYearMin: minFirstAlbumYear,
		FirstAlbumYearMax: maxFirstAlbumYear,
		MemberCounts:      memberCounts,
		Countries:         countries,
	}
}

// calculateLocationFilterOptions derives available location filter metadata.
func (s *Store) calculateLocationFilterOptions(locations []Location) LocationFilterOptions {
	if len(locations) == 0 {
		return LocationFilterOptions{}
	}

	// Calculate min/max values in a single pass
	minConcerts := locations[0].TotalConcerts()
	maxConcerts := locations[0].TotalConcerts()
	minArtists := locations[0].ArtistCount()
	maxArtists := locations[0].ArtistCount()
	minYear := 9999
	maxYear := 0
	
	countrySet := make(map[string]bool)

	for _, location := range locations {
		totalConcerts := location.TotalConcerts()
		if totalConcerts < minConcerts {
			minConcerts = totalConcerts
		}
		if totalConcerts > maxConcerts {
			maxConcerts = totalConcerts
		}

		artistCount := location.ArtistCount()
		if artistCount < minArtists {
			minArtists = artistCount
		}
		if artistCount > maxArtists {
			maxArtists = artistCount
		}

		earliest, latest := location.YearRange()
		if earliest > 0 && earliest < minYear {
			minYear = earliest
		}
		if latest > maxYear {
			maxYear = latest
		}

		country := location.Country()
		if country != "" {
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return LocationFilterOptions{
		ConcertCountMin: minConcerts,
		ConcertCountMax: maxConcerts,
		ArtistCountMin:  minArtists,
		ArtistCountMax:  maxArtists,
		ConcertYearMin:  minYear,
		ConcertYearMax:  maxYear,
		Countries:       countries,
	}
}
```

### Step 8: Update Form Parameter Parsing
**File to modify:** `internal/web/templates.go`

```go
// parseArtistFilterParams extracts artist filter parameters from form data.
// Simplified to work with the new structure.
func parseArtistFilterParams(r *http.Request) data.ArtistFilterParams {
	var params data.ArtistFilterParams

	// Parse creation year range
	params.CreationYear.Min = parseIntPtr(r, "creationYearFrom")
	params.CreationYear.Max = parseIntPtr(r, "creationYearTo")
	
	// Parse first album year range
	params.FirstAlbumYear.Min = parseIntPtr(r, "firstAlbumYearFrom")
	params.FirstAlbumYear.Max = parseIntPtr(r, "firstAlbumYearTo")
	
	// Parse member counts
	params.MemberCounts = parseIntSlice(r, "memberCounts")
	
	// Parse countries
	params.Countries = parseStringSlice(r, "countries")

	return params
}

// parseLocationFilterParams extracts location filter parameters from form data.
// Simplified to work with the new structure.
func parseLocationFilterParams(r *http.Request) data.LocationFilterParams {
	var params data.LocationFilterParams

	// Parse concert count range
	params.ConcertCount.Min = parseIntPtr(r, "concertCountFrom")
	params.ConcertCount.Max = parseIntPtr(r, "concertCountTo")
	
	// Parse artist count range
	params.ArtistCount.Min = parseIntPtr(r, "artistCountFrom")
	params.ArtistCount.Max = parseIntPtr(r, "artistCountTo")
	
	// Parse year range
	params.YearRange.Min = parseIntPtr(r, "concertYearFrom")
	params.YearRange.Max = parseIntPtr(r, "concertYearTo")
	
	// Parse countries
	params.Countries = parseStringSlice(r, "countries")

	return params
}
```

### Step 9: Update Tests to Work with New Filter System
**File to modify:** `internal/data/data_test.go`

```go
// TestFilterArtists tests all artist filtering functionality with the new functional approach
func TestFilterArtists(t *testing.T) {
	store := createTestStore(t)

	tests := []struct {
		name    string
		params  ArtistFilterParams
		wantMin int
		check   func(t *testing.T, artists []*Artist)
	}{
		{
			name: "Filter by creation year range 1995-2000",
			params: ArtistFilterParams{
				CreationYear: RangeFilter[int]{Min: intPtr(1995), Max: intPtr(2000)},
			},
			wantMin: 7,
			check: func(t *testing.T, artists []*Artist) {
				for _, artist := range artists {
					if artist.CreationYear < 1995 || artist.CreationYear > 2000 {
						t.Errorf("Artist %s has creation year %d which is outside the range [1995, 2000]", 
							artist.Name, artist.CreationYear)
					}
				}
			},
		},
		{
			name: "Filter by member counts",
			params: ArtistFilterParams{
				MemberCounts: []int{1, 5}, // Solo artists or 5-member bands
			},
			check: func(t *testing.T, artists []*Artist) {
				for _, artist := range artists {
					memberCount := artist.MemberCount()
					if memberCount != 1 && memberCount != 5 {
						t.Errorf("Artist %s has %d members, expected 1 or 5", artist.Name, memberCount)
					}
				}
			},
		},
		{
			name: "Filter by countries",
			params: ArtistFilterParams{
				Countries: []string{"USA", "UK"},
			},
			wantMin: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := store.FilterArtists(tt.params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterArtists() returned %d artists, want at least %d", len(results), tt.wantMin)
				t.Logf("Got artists: %v", getArtistNames(results))
			}

			if tt.check != nil {
				tt.check(t, results)
			}
		})
	}
}
```

## Testing Strategy for Phase 2
1. Update existing filter tests to work with the new functional approach
2. Test all individual filter functions in isolation
3. Test combined filter scenarios to ensure AND logic works correctly
4. Verify that empty filters return all results
5. Test edge cases like filters with nil/empty values
6. Ensure performance is maintained or improved

## Rollout Considerations
- The simplified filter system should be more maintainable and flexible
- All existing functionality should be preserved
- Consider backward compatibility if other parts of the system depend on old structures
- Update documentation and examples to reflect the new approach