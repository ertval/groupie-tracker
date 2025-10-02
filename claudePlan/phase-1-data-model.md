# Phase 1: Data Model & Loading Overhaul

## Overview
This phase focuses on simplifying the data model by removing cached/computed fields and replacing them with helper methods, introducing proper date handling, and creating a clean Catalog component to own normalized data.

## Step 1.1: Refactor Artist Struct
**Goal:** Remove cached/computed fields and use helper methods instead.

### Sub-steps:
1. **Backup current Artist struct** (in `internal/data/models.go`)
   - Copy current implementation to a comment block for reference

2. **Remove cached fields from Artist struct:**
   ```go
   // Remove these fields:
   - ConcertCount
   - MemberCount
   - Countries
   - FirstAlbumYear
   - DatesAtLocation
   ```

3. **Add Concert slice to Artist:**
   ```go
   type Artist struct {
       ID           int
       Name         string
       Image        string
       Members      []string
       CreationDate int
       FirstAlbum   string
       Concerts     []Concert  // New field
   }
   ```

4. **Implement helper methods on Artist:**
   ```go
   func (a *Artist) MemberCount() int
   func (a *Artist) ConcertCount() int
   func (a *Artist) FirstAlbumYear() int
   func (a *Artist) Countries() []string
   func (a *Artist) Slug() string
   ```

5. **Update all references:**
   - Search for `artist.ConcertCount` → replace with `artist.ConcertCount()`
   - Search for `artist.MemberCount` → replace with `artist.MemberCount()`
   - Search for `artist.Countries` → replace with `artist.Countries()`
   - Search for `artist.FirstAlbumYear` → replace with `artist.FirstAlbumYear()`

6. **Run tests:** `go test ./internal/data/...`

## Step 1.2: Refactor Concert/Date Handling
**Goal:** Parse dates once into `time.Time` and normalize location slugs.

### Sub-steps:
1. **Update Concert struct** (in `internal/data/models.go`):
   ```go
   type Concert struct {
       ArtistID     int
       Location     string
       LocationSlug string
       Date         time.Time  // Changed from string
       DateString   string     // Keep for display
   }
   ```

2. **Create date parsing utility:**
   ```go
   func parseDate(dateStr string) (time.Time, error)
   ```

3. **Create location slug utility:**
   ```go
   func normalizeLocationSlug(location string) string
   ```

4. **Update data loading to parse dates once:**
   - Modify the relation loading code to call `parseDate()`
   - Store both parsed `time.Time` and original string

5. **Update all date comparisons:**
   - Replace string comparisons with `time.Time` comparisons
   - Update filter logic to use `time.Time`

6. **Run tests:** `go test ./internal/data/...`

## Step 1.3: Refactor Location Struct
**Goal:** Remove wrapper types and cached aggregates.

### Sub-steps:
1. **Remove ArtistAtLocation wrapper:**
   - Identify all uses of this type
   - Plan replacement with direct Concert slice

2. **Simplify Location struct:**
   ```go
   type Location struct {
       Name      string
       Concerts  []Concert
   }
   ```

3. **Implement Location helper methods:**
   ```go
   func (l *Location) ArtistCount() int
   func (l *Location) TotalConcerts() int
   func (l *Location) YearRange() (int, int)
   func (l *Location) Slug() string
   func (l *Location) Country() string
   ```

4. **Update all Location references:**
   - Replace cached field access with method calls

5. **Run tests:** `go test ./internal/data/...`

## Step 1.4: Introduce Catalog Component
**Goal:** Create a lightweight catalog that owns normalized data.

### Sub-steps:
1. **Create new Catalog struct** (in `internal/data/catalog.go`):
   ```go
   type Catalog struct {
       Artists   []Artist
       Locations map[string]*Location
       Concerts  []Concert
   }
   ```

2. **Add Catalog builder methods:**
   ```go
   func NewCatalog() *Catalog
   func (c *Catalog) AddArtist(artist Artist)
   func (c *Catalog) AddConcert(concert Concert)
   func (c *Catalog) Build() error
   ```

3. **Move normalization logic into Catalog:**
   - Concert grouping by artist
   - Location grouping
   - Index building

4. **Create Catalog query methods:**
   ```go
   func (c *Catalog) ArtistByID(id int) (*Artist, error)
   func (c *Catalog) ArtistBySlug(slug string) (*Artist, error)
   func (c *Catalog) LocationBySlug(slug string) (*Location, error)
   func (c *Catalog) AllArtists() []Artist
   func (c *Catalog) AllLocations() []Location
   ```

5. **Run tests:** `go test ./internal/data/...`

## Step 1.5: Refactor Store to Use Catalog
**Goal:** Make Store orchestrate filters/search, delegate data to Catalog.

### Sub-steps:
1. **Update Store struct** (in `internal/data/store.go`):
   ```go
   type Store struct {
       catalog *Catalog
       // Remove duplicate maps/slices
   }
   ```

2. **Update Store initialization:**
   - Build Catalog first
   - Remove redundant indexing

3. **Delegate data access to Catalog:**
   ```go
   func (s *Store) GetArtist(id int) (*Artist, error) {
       return s.catalog.ArtistByID(id)
   }
   // Repeat for all data access methods
   ```

4. **Remove duplicate indexing logic:**
   - Delete maps that Catalog now handles
   - Remove redundant slices

5. **Run tests:** `go test ./internal/data/...`

## Step 1.6: Simplify Loading Pipeline
**Goal:** Make data loading sequential and clear.

### Sub-steps:
1. **Restructure loadData function** (in `internal/data/store.go`):
   ```go
   func (s *Store) loadData() error {
       // Stage 1: Fetch raw data
       artists, err := s.fetchArtists()
       if err != nil { return err }

       relations, err := s.fetchRelations()
       if err != nil { return err }

       // Stage 2: Normalize
       concerts := s.normalizeRelations(relations)

       // Stage 3: Build catalog
       catalog := NewCatalog()
       for _, a := range artists {
           catalog.AddArtist(a)
       }
       for _, c := range concerts {
           catalog.AddConcert(c)
       }

       if err := catalog.Build(); err != nil {
           return err
       }

       s.catalog = catalog

       // Stage 4: Optional image caching
       s.cacheImages()

       return nil
   }
   ```

2. **Remove goroutine fan-out:**
   - Delete worker pools
   - Remove unnecessary channels
   - Simplify error handling

3. **Extract normalization helpers:**
   ```go
   func normalizeRelations(raw *apiRelation) []Concert
   func normalizeArtist(raw *apiArtist) Artist
   ```

4. **Add sync.Once for one-time initialization:**
   ```go
   var (
       slugRegex     *regexp.Regexp
       slugRegexOnce sync.Once
   )

   func getSlugRegex() *regexp.Regexp {
       slugRegexOnce.Do(func() {
           slugRegex = regexp.MustCompile(`[^a-z0-9]+`)
       })
       return slugRegex
   }
   ```

5. **Run tests:** `go test ./internal/data/...`

## Step 1.7: Type Hygiene Cleanup
**Goal:** Ensure idiomatic Go naming and zero-value friendliness.

### Sub-steps:
1. **Review all exported types:**
   - Ensure PascalCase for exported
   - Ensure camelCase for unexported
   - Remove `Get` prefixes

2. **Make helper utilities unexported:**
   - `slugify()` → unexported
   - `extractCountry()` → unexported
   - Colocate with usage

3. **Verify zero-value behavior:**
   - Test empty Artist creation
   - Test empty Location creation
   - Ensure no nil panics

4. **Run `gofmt` and `go vet`:**
   ```bash
   gofmt -w .
   go vet ./...
   ```

5. **Run all tests:** `go test ./...`