# Phase 5: Code Polish & Documentation - Implementation Guide

## Overview
This phase focuses on polishing the codebase by removing redundant comments, normalizing naming conventions, organizing helper packages, and updating documentation. The goal is to create a clean, consistent, and maintainable codebase that follows Go idioms.

## Step-by-Step Implementation

### Step 1: Normalize Naming Conventions
**File to modify:** All Go files in the project

**Changes to apply:**
1. Remove `Get` prefixes from getter methods
2. Use consistent `ByID`, `BySlug`, `Adjacent`, `Search` naming patterns
3. Ensure all public methods follow idiomatic Go naming

**Examples of changes:**

**Before:**
```go
func (s *Store) GetArtistFilterOptions() ArtistFilterOptions { ... }
func (s *Store) GetLocationFilterOptions() LocationFilterOptions { ... }
func (a *Artist) GetConcertCount() int { ... }
```

**After:**
```go
func (s *Store) ArtistFilterOptions() ArtistFilterOptions { ... }
func (s *Store) LocationFilterOptions() LocationFilterOptions { ... }
func (a *Artist) ConcertCount() int { ... }
```

### Step 2: Remove Redundant Comments
**File to modify:** All Go files in the project

**Guidelines:**
1. Remove comments that just repeat what the code does
2. Keep comments that explain WHY something is done
3. Keep comments for complex business logic
4. Keep comments for exported functions/methods that explain their purpose

**Examples:**

**Remove this type of comment:**
```go
// concertCount returns the number of concerts for this artist
// by counting the concerts in the concerts slice
func (a *Artist) concertCount() int {
    return len(a.Concerts)  // Return the length of the concerts slice
}
```

**Keep this type of comment:**
```go
// concertCount returns the number of concerts for this artist.
// Note: This method computes the count on-demand rather than using a cached value
// to ensure data consistency with the Concerts slice.
func (a *Artist) concertCount() int {
    return len(a.Concerts)
}
```

### Step 3: Organize Helper Functions
**File to create:** `internal/utils/string_helpers.go`

```go
package utils

import (
	"regexp"
	"sort"
	"strings"
)

// Global compiled regex for performance
var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

// CreateSlug converts a string to URL-friendly slug format
func CreateSlug(name string) string {
	slug := slugRegex.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

// ExtractCountryFromLocation normalizes a location string and returns a display-ready country name
func ExtractCountryFromLocation(location string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(location)), "-")
	if len(parts) == 0 {
		return ""
	}

	country := strings.TrimSpace(parts[len(parts)-1])
	if country == "" {
		return ""
	}

	switch country {
	case "usa", "us":
		return "USA"
	case "uk":
		return "UK"
	case "uae":
		return "UAE"
	}

	words := strings.Fields(strings.ReplaceAll(country, "-", " "))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}

// ExtractYearFromDate extracts year from date string in DD-MM-YYYY or YYYY format
func ExtractYearFromDate(dateStr string) int {
	dateStr = strings.TrimSpace(dateStr)
	if len(dateStr) < 4 {
		return 0
	}

	// Try DD-MM-YYYY format first
	if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
		if year, err := strconv.Atoi(dateStr[6:10]); err == nil {
			return year
		}
	}

	// Try YYYY format
	if year, err := strconv.Atoi(dateStr[:4]); err == nil && year > 1900 && year < 3000 {
		return year
	}

	return 0
}

// NormalizeLocation converts raw location strings to consistent format
func NormalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), "_", "-"))
}

// RemoveDuplicates removes duplicate strings from a slice
func RemoveDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// RemoveDuplicatesInt removes duplicate integers from a slice
func RemoveDuplicatesInt(slice []int) []int {
	seen := make(map[int]bool)
	var result []int
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// SortStrings sorts a slice of strings in place
func SortStrings(slice []string) {
	sort.Strings(slice)
}

// SortInts sorts a slice of integers in place
func SortInts(slice []int) {
	sort.Ints(slice)
}
```

### Step 4: Create Numeric Utilities
**File to create:** `internal/utils/numeric_helpers.go`

```go
package utils

import (
	"fmt"
)

// Min returns the smaller of two comparable values
func Min[T comparable](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of two comparable values
func Max[T comparable](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Clamp restricts a value to be within min and max bounds
func Clamp[T comparable](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// PointerToInt converts an int to a pointer
func PointerToInt(i int) *int {
	return &i
}

// PointerToString converts a string to a pointer
func PointerToString(s string) *string {
	return &s
}

// IntSliceContains checks if a slice contains a specific integer
func IntSliceContains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// StringSliceContains checks if a slice contains a specific string
func StringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

### Step 5: Update Data Layer to Use New Utilities
**File to modify:** `internal/data/models.go` and related files

```go
// Update import to include utilities
import (
	"groupie-tracker/internal/utils"
	// ... other imports
)

// Update Artist helper methods to use new utilities
func (a *Artist) Slug() string {
	return utils.CreateSlug(a.Name)
}

func (l *Location) Slug() string {
	return utils.CreateSlug(l.Name)
}

func (a *Artist) Countries() []string {
	countries := make(map[string]bool)
	for _, concert := range a.Concerts {
		country := utils.ExtractCountryFromLocation(concert.Location)
		if country != "" {
			countries[country] = true
		}
	}
	
	result := make([]string, 0, len(countries))
	for country := range countries {
		result = append(result, country)
	}
	utils.SortStrings(result)
	return result
}

func (a *Artist) FirstAlbumYear() int {
	return utils.ExtractYearFromDate(a.FirstAlbum)
}
```

### Step 6: Update Store to Use New Utilities
**File to modify:** `internal/data/store.go`

```go
// Update import to include utilities
import (
	"groupie-tracker/internal/api"
	"groupie-tracker/internal/utils"
	// ... other imports
)

// Simplify country extraction
func extractCountryFromLocation(location string) string {
	return utils.ExtractCountryFromLocation(location)
}

// Simplify slug creation
func createSlug(name string) string {
	return utils.CreateSlug(name)
}
```

### Step 7: Update Web Layer to Use New Utilities
**File to modify:** `internal/web/handlers.go`

```go
// Update import to include utilities
import (
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/utils"
	// ... other imports
)

// Update any web-specific utilities that use string manipulation
```

### Step 8: Update README and Documentation
**File to modify:** `README.md`

Update the README to reflect the new simplified architecture:

**Before (relevant section):**
```
### Data Layer (`internal/data`)

- Fetches artists and relations concurrently via `api.Client`.
- Normalises locations, builds SEO-friendly slugs, and precomputes:
  - artist indexes (`byID`, `bySlug`, `position`) for O(1) lookups
  - location aggregates with artist counts and concert stats
  - filter metadata (year bounds, member counts, country lists)
  - search suggestions with cached lowercase tokens
- Optionally caches artist images with a four-worker pool.
```

**After:**
```
### Data Layer (`internal/data`)

- Fetches artists and relations sequentially via `api.Client`.
- Normalises locations, builds SEO-friendly slugs using helper functions:
  - artist indexes (`ByID`, `BySlug`) for O(1) lookups
  - computed properties via helper methods on Artist and Location structs
  - filter metadata (year bounds, member counts, country lists)
  - search suggestions with precomputed tokens for efficient lookup
- Optionally caches artist images with a simple caching interface.
```

### Step 9: Update Project Structure Documentation
**File to create:** `doc/ARCHITECTURE_SUMMARY.md`

```markdown
# Architecture Summary

## Overview
Groupie Tracker is a simplified web application that renders artist and concert data with a clean, server-side architecture. The refactored codebase follows Go idioms and the KISS principle.

## Components

### 1. API Client (`internal/api/`)
- Simple HTTP client for the Groupie Trackers API
- Minimal request/response handling with timeout protection

### 2. Data Layer (`internal/data/`)
- Core domain models (Artist, Location) with helper methods
- Functional filtering using predicate functions
- Simplified search with precomputed tokens
- Sequential data loading pipeline

### 3. Utilities (`internal/utils/`)
- String manipulation and normalization utilities
- Numeric helper functions
- Reusable common functions

### 4. Web Layer (`internal/web/`)
- View models for all pages with common fields
- Simplified handlers with minimal business logic
- Template rendering with reusable functions
- Standard middleware stack

## Key Design Principles
- Lean structs with computed values via helper methods
- Functional filtering with predicate composition
- Sequential execution by default
- Data normalization during load phase
- Idiomatic Go naming conventions
```

### Step 10: Add Inline Documentation for Complex Logic
**File to modify:** Add documentation to complex functions

```go
// FilterArtists applies user-specified filter criteria to the artist collection and returns matching artists.
// Uses functional approach with predicate composition for flexible filtering.
// 
// The filtering logic:
// 1. Builds individual filter functions from the provided parameters
// 2. Combines filters with AND semantics using AndFilters
// 3. Applies the combined filter to the artist collection
// 
// Performance: This function iterates through the artist collection once.
// For better performance with complex filters, consider pre-filtering
// with the most selective criteria first.
func (s *Store) FilterArtists(params ArtistFilterParams) []*Artist {
	// Implementation as described in previous phases
}
```

### Step 11: Clean Up Go Doc Comments
**File to modify:** All files with exported functions

Ensure all exported functions have clear, concise documentation:

```go
// Slug returns a URL-friendly identifier for the artist
// by converting the artist name to lowercase and replacing
// non-alphanumeric characters with hyphens.
func (a *Artist) Slug() string {
	return utils.CreateSlug(a.Name)
}

// ConcertCount returns the number of concerts in the artist's schedule.
// This value is computed dynamically from the Concerts slice to ensure
// consistency with the current data.
func (a *Artist) ConcertCount() int {
	return len(a.Concerts)
}
```

### Step 12: Update Config Package if Needed
**File to modify:** `internal/conf/conf.go`

If configuration values need to be adjusted for the simplified architecture, update them:

```go
package conf

import "time"

// Server and API configuration defaults. Tests can override these values.
var (
	// Enable or disable image caching (true = enabled, false = disabled)
	WithCache = false

	// API base URL used by the data layer when fetching data
	APIBaseURL = "https://groupietrackers.herokuapp.com"

	// Request timeout for API calls
	APIRequestTimeout = 30 * time.Second

	// HTTP server defaults (port and timeouts)
	DefaultPort  = ":8080"
	ReadTimeout  = 15 * time.Second
	WriteTimeout = 15 * time.Second
	IdleTimeout  = 60 * time.Second
)

// Update any other configuration values as needed for the simplified system
```

## Testing Strategy for Phase 5
1. Verify that all renamed methods still compile and function correctly
2. Ensure all new utility functions are thoroughly tested
3. Run the complete test suite to confirm no regressions
4. Verify that documentation updates are accurate
5. Review all changes for consistency with Go idioms

## Rollout Considerations
- The naming changes may affect external imports if any exist
- Ensure all internal references to renamed functions are updated
- The utility package should be small and focused with clear responsibilities
- Documentation updates should accurately reflect the simplified architecture
- Consider creating a migration guide if external consumers might be affected