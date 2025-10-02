# Phase 2: Filters & Options Simplification

## Overview
This phase focuses on creating a clean, composable filter framework using predicate functions, introducing reusable range and set helpers, and simplifying filter parsing from HTTP requests.

## Step 2.1: Create Predicate-Based Filter Framework
**Goal:** Implement composable filter functions.

### Sub-steps:
1. **Create filter types** (in `internal/data/filters.go`):
   ```go
   type ArtistFilterFunc func(*Artist) bool
   type LocationFilterFunc func(*Location) bool
   ```

2. **Implement basic filter builders:**
   ```go
   func CreationYearBetween(min, max int) ArtistFilterFunc
   func HasMemberCount(counts ...int) ArtistFilterFunc
   func InCountries(countries ...string) ArtistFilterFunc
   func FirstAlbumYearBetween(min, max int) ArtistFilterFunc
   ```

3. **Create filter combiner:**
   ```go
   func AndFilters(filters ...ArtistFilterFunc) ArtistFilterFunc {
       return func(a *Artist) bool {
           for _, f := range filters {
               if !f(a) {
                   return false
               }
           }
           return true
       }
   }
   ```

4. **Run tests:** Create test cases for each filter builder

## Step 2.2: Create Range and Set Helpers
**Goal:** Reusable utilities for common filter patterns.

### Sub-steps:
1. **Create IntRange type:**
   ```go
   type IntRange struct {
       Min int
       Max int
   }

   func (r IntRange) Contains(value int) bool {
       return value >= r.Min && value <= r.Max
   }

   func (r IntRange) IsZero() bool {
       return r.Min == 0 && r.Max == 0
   }
   ```

2. **Create StringSet type:**
   ```go
   type StringSet map[string]struct{}

   func NewStringSet(items ...string) StringSet
   func (s StringSet) Contains(item string) bool
   func (s StringSet) IsEmpty() bool
   ```

3. **Create IntSet type:**
   ```go
   type IntSet map[int]struct{}

   func NewIntSet(items ...int) IntSet
   func (s IntSet) Contains(item int) bool
   func (s IntSet) IsEmpty() bool
   ```

4. **Update filter builders to use these types:**
   ```go
   func CreationYearInRange(r IntRange) ArtistFilterFunc {
       if r.IsZero() {
           return func(*Artist) bool { return true }
       }
       return func(a *Artist) bool {
           return r.Contains(a.CreationDate)
       }
   }
   ```

5. **Run tests:** `go test ./internal/data/...`

## Step 2.3: Create Filter Structs with Match Methods
**Goal:** Optional explicit filter types for complex cases.

### Sub-steps:
1. **Create ArtistFilters struct:**
   ```go
   type ArtistFilters struct {
       CreationYear IntRange
       MemberCounts IntSet
       Countries    StringSet
       FirstAlbum   IntRange
   }

   func (f ArtistFilters) Match(a *Artist) bool
   func (f ArtistFilters) IsEmpty() bool
   func (f ArtistFilters) ToFilterFunc() ArtistFilterFunc
   ```

2. **Create LocationFilters struct:**
   ```go
   type LocationFilters struct {
       Countries StringSet
       YearRange IntRange
   }

   func (f LocationFilters) Match(l *Location) bool
   func (f LocationFilters) IsEmpty() bool
   func (f LocationFilters) ToFilterFunc() LocationFilterFunc
   ```

3. **Implement Match methods using predicates:**
   ```go
   func (f ArtistFilters) Match(a *Artist) bool {
       if !f.CreationYear.IsZero() && !f.CreationYear.Contains(a.CreationDate) {
           return false
       }
       if !f.MemberCounts.IsEmpty() && !f.MemberCounts.Contains(a.MemberCount()) {
           return false
       }
       // ... etc
       return true
   }
   ```

4. **Run tests:** `go test ./internal/data/...`

## Step 2.4: Generate Filter Options Metadata
**Goal:** Compute available filter options from normalized data.

### Sub-steps:
1. **Create FilterOptions types:**
   ```go
   type ArtistFilterOptions struct {
       CreationYears IntRange
       MemberCounts  []int
       Countries     []string
       FirstAlbum    IntRange
   }

   type LocationFilterOptions struct {
       Countries []string
       YearRange IntRange
   }
   ```

2. **Implement options generator in Catalog:**
   ```go
   func (c *Catalog) ArtistFilterOptions() ArtistFilterOptions {
       // Scan all artists to collect unique values
       // Return sorted, deduplicated results
   }

   func (c *Catalog) LocationFilterOptions() LocationFilterOptions {
       // Scan all locations to collect unique values
       // Return sorted, deduplicated results
   }
   ```

3. **Call during Catalog.Build():**
   ```go
   func (c *Catalog) Build() error {
       // ... existing build logic ...

       // Precompute filter options
       c.artistOptions = c.computeArtistFilterOptions()
       c.locationOptions = c.computeLocationFilterOptions()

       return nil
   }
   ```

4. **Update Store methods:**
   ```go
   func (s *Store) GetArtistFilterOptions() ArtistFilterOptions {
       return s.catalog.artistOptions
   }
   ```

5. **Run tests:** `go test ./internal/data/...`

## Step 2.5: Simplify Filter Parsing
**Goal:** Clean up request parameter parsing.

### Sub-steps:
1. **Create unified filter parser:**
   ```go
   func ParseArtistFilters(r *http.Request) (ArtistFilters, error)
   func ParseLocationFilters(r *http.Request) (LocationFilters, error)
   ```

2. **Remove pointer-heavy parsing:**
   - Use value types instead of pointers
   - Return zero values for empty filters

3. **Consolidate duplicate parsers:**
   - Merge similar parsing logic
   - Extract common helpers (parseIntRange, parseIntSet, etc.)

4. **Update handlers to use new parsers:**
   ```go
   filters, err := ParseArtistFilters(r)
   if err != nil {
       // handle error
   }
   results := store.FilterArtists(filters)
   ```

5. **Run tests:** `go test ./internal/web/...`