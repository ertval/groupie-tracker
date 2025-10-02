# Consolidated Idiomatic Go Refactoring - Detailed Implementation Plan

## Overview
This document provides a detailed, step-by-step implementation guide for each phase of the consolidation refactoring plan. Follow these steps sequentially to ensure a smooth transition.

---

## Phase 1: Data Model & Loading Overhaul

### Step 1.1: Refactor Artist Struct
**Goal:** Remove cached/computed fields and use helper methods instead.

#### Sub-steps:
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

### Step 1.2: Refactor Concert/Date Handling
**Goal:** Parse dates once into `time.Time` and normalize location slugs.

#### Sub-steps:
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

### Step 1.3: Refactor Location Struct
**Goal:** Remove wrapper types and cached aggregates.

#### Sub-steps:
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

### Step 1.4: Introduce Catalog Component
**Goal:** Create a lightweight catalog that owns normalized data.

#### Sub-steps:
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

### Step 1.5: Refactor Store to Use Catalog
**Goal:** Make Store orchestrate filters/search, delegate data to Catalog.

#### Sub-steps:
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

### Step 1.6: Simplify Loading Pipeline
**Goal:** Make data loading sequential and clear.

#### Sub-steps:
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

### Step 1.7: Type Hygiene Cleanup
**Goal:** Ensure idiomatic Go naming and zero-value friendliness.

#### Sub-steps:
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

---

## Phase 2: Filters & Options Simplification

### Step 2.1: Create Predicate-Based Filter Framework
**Goal:** Implement composable filter functions.

#### Sub-steps:
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

### Step 2.2: Create Range and Set Helpers
**Goal:** Reusable utilities for common filter patterns.

#### Sub-steps:
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

### Step 2.3: Create Filter Structs with Match Methods
**Goal:** Optional explicit filter types for complex cases.

#### Sub-steps:
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

### Step 2.4: Generate Filter Options Metadata
**Goal:** Compute available filter options from normalized data.

#### Sub-steps:
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

### Step 2.5: Simplify Filter Parsing
**Goal:** Clean up request parameter parsing.

#### Sub-steps:
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

---

## Phase 3: Search & Suggestion Refactor

### Step 3.1: Remove Heavy Search Infrastructure
**Goal:** Eliminate LRU cache, mutexes, and complex bookkeeping.

#### Sub-steps:
1. **Identify current search components:**
   - List all search-related types
   - Document current flow
   - Note what to keep vs. remove

2. **Remove LRU cache:**
   - Delete cache implementation
   - Remove cache-related mutexes
   - Simplify to direct search

3. **Remove order bookkeeping:**
   - Delete search result ordering logic
   - Simplify to sorted slice return

4. **Clean up SearchParams and SearchResult types:**
   - Evaluate if still needed
   - Simplify or remove

5. **Run tests:** `go test ./internal/data/...`

### Step 3.2: Build Normalized Token Index
**Goal:** Create simple search index during catalog build.

#### Sub-steps:
1. **Create search index types:**
   ```go
   type SearchIndex struct {
       artistTokens   map[int][]string  // artistID -> normalized tokens
       locationTokens map[string][]string // locationSlug -> normalized tokens
   }
   ```

2. **Create token normalization function:**
   ```go
   func normalizeTokens(text string) []string {
       // Lowercase, split, remove special chars
       // Return deduplicated tokens
   }
   ```

3. **Build index during Catalog.Build():**
   ```go
   func (c *Catalog) buildSearchIndex() *SearchIndex {
       index := &SearchIndex{
           artistTokens:   make(map[int][]string),
           locationTokens: make(map[string][]string),
       }
       
       for _, a := range c.Artists {
           tokens := normalizeTokens(a.Name)
           tokens = append(tokens, normalizeTokens(strings.Join(a.Members, " "))...)
           // Add other searchable fields
           index.artistTokens[a.ID] = tokens
       }
       
       // Similar for locations
       
       return index
   }
   ```

4. **Add search index to Catalog:**
   ```go
   type Catalog struct {
       // ... existing fields ...
       searchIndex *SearchIndex
   }
   ```

5. **Run tests:** `go test ./internal/data/...`

### Step 3.3: Implement Direct Search
**Goal:** Simple, direct search using the token index.

#### Sub-steps:
1. **Create unified search method:**
   ```go
   func (s *Store) Search(query string) SearchResults {
       normalized := normalizeTokens(query)
       
       artists := s.searchArtists(normalized)
       locations := s.searchLocations(normalized)
       
       return SearchResults{
           Artists:   artists,
           Locations: locations,
       }
   }
   ```

2. **Implement token matching:**
   ```go
   func (s *Store) searchArtists(queryTokens []string) []Artist {
       var results []Artist
       
       for _, artist := range s.catalog.Artists {
           if matchesTokens(s.catalog.searchIndex.artistTokens[artist.ID], queryTokens) {
               results = append(results, artist)
           }
       }
       
       return results
   }
   
   func matchesTokens(docTokens, queryTokens []string) bool {
       // Check if any query token matches any doc token
       for _, qt := range queryTokens {
           for _, dt := range docTokens {
               if strings.Contains(dt, qt) {
                   return true
               }
           }
       }
       return false
   }
   ```

3. **Add relevance sorting:**
   ```go
   func sortByRelevance(results []Artist, query string) {
       sort.Slice(results, func(i, j int) bool {
           // Exact name match first
           // Then prefix match
           // Then contains
           // Then alphabetical
       })
   }
   ```

4. **Update handlers:**
   - Replace old search calls with new `Store.Search()`
   - Simplify result handling

5. **Run tests:** `go test ./internal/data/...`

### Step 3.4: Refactor Suggestions
**Goal:** Simplify suggestion generation.

#### Sub-steps:
1. **Decide on approach:**
   - **Option A:** Client-side JSON index (recommended for small datasets)
   - **Option B:** Simple server-side endpoint

2. **Option A: Client-Side Implementation:**
   ```go
   // Create endpoint for search index
   func (h *Handlers) handleSearchIndex(w http.ResponseWriter, r *http.Request) {
       index := h.store.GetSearchIndex()
       json.NewEncoder(w).Encode(index)
   }
   
   // Client-side JavaScript handles filtering/suggestions
   ```

3. **Option B: Server-Side Implementation:**
   ```go
   func (s *Store) GetSuggestions(query string, limit int) []Suggestion {
       // Simple prefix/contains matching
       // Return top N results with formatted text/URL
   }
   
   type Suggestion struct {
       Text string
       URL  string
       Type string // "artist" or "location"
   }
   ```

4. **Create suggestion formatting helpers:**
   ```go
   func formatArtistSuggestion(a *Artist) Suggestion
   func formatLocationSuggestion(l *Location) Suggestion
   ```

5. **Update web handlers:**
   - Remove complex suggestion infrastructure
   - Use new simple approach

6. **Run tests:** `go test ./internal/web/...`

### Step 3.5: API Cleanup
**Goal:** Remove legacy search types.

#### Sub-steps:
1. **Audit usage of SearchParams:**
   - Find all references
   - Plan replacement

2. **Audit usage of SearchResult:**
   - Find all references
   - Plan replacement

3. **Remove or simplify types:**
   - Delete if no longer needed
   - Simplify to basic structs if still useful

4. **Update all callers:**
   - Use new simpler types
   - Update tests

5. **Run tests:** `go test ./...`

---

## Phase 4: Web Layer & Template Simplification

### Step 4.1: Create Shared View Models
**Goal:** Eliminate repetitive anonymous structs.

#### Sub-steps:
1. **Create view package:**
   ```bash
   mkdir internal/view
   touch internal/view/models.go
   ```

2. **Define base Page struct:**
   ```go
   package view
   
   type Page struct {
       Title       string
       Description string
       Data        interface{}
       Assets      Assets
       Error       *ErrorInfo
   }
   
   type Assets struct {
       CSS []string
       JS  []string
   }
   
   type ErrorInfo struct {
       Code    int
       Message string
   }
   ```

3. **Define specific page types:**
   ```go
   type HomePage struct {
       Page
       Stats      SiteStats
       Featured   []Artist
   }
   
   type ArtistListPage struct {
       Page
       Artists       []Artist
       FilterOptions ArtistFilterOptions
       ActiveFilters ArtistFilters
   }
   
   type ArtistDetailPage struct {
       Page
       Artist    Artist
       Prev      *Artist
       Next      *Artist
       Locations []Location
   }
   
   // ... etc
   ```

4. **Create view builder functions:**
   ```go
   func NewHomePage(store *data.Store) HomePage
   func NewArtistListPage(artists []Artist, options ArtistFilterOptions, filters ArtistFilters) ArtistListPage
   func NewArtistDetailPage(artist Artist, store *data.Store) ArtistDetailPage
   ```

5. **Update handlers to use view models:**
   ```go
   func (h *Handlers) handleHome(w http.ResponseWriter, r *http.Request) {
       page := view.NewHomePage(h.store)
       h.render(w, "home.tmpl", page)
   }
   ```

6. **Run tests:** `go test ./internal/web/...`

### Step 4.2: Slim Down Handlers
**Goal:** Push business logic to data layer.

#### Sub-steps:
1. **Create reusable request helpers:**
   ```go
   // In internal/web/helpers.go
   
   func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
       if r.Method != method {
           w.Header().Set("Allow", method)
           http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
           return false
       }
       return true
   }
   
   func parseFilters(r *http.Request) (data.ArtistFilters, error) {
       // Consolidated filter parsing
   }
   
   func respondJSON(w http.ResponseWriter, data interface{}) error {
       w.Header().Set("Content-Type", "application/json")
       return json.NewEncoder(w).Encode(data)
   }
   
   func respondError(w http.ResponseWriter, code int, message string) {
       // Standardized error response
   }
   ```

2. **Move sorting logic to Store:**
   ```go
   // In internal/data/store.go
   
   func (s *Store) ArtistsSortedBy(field string, ascending bool) []Artist {
       artists := s.catalog.AllArtists()
       // Sorting logic
       return artists
   }
   ```

3. **Refactor handler pattern:**
   ```go
   func (h *Handlers) handleArtists(w http.ResponseWriter, r *http.Request) {
       if !requireMethod(w, r, http.MethodGet) {
           return
       }
       
       filters, err := parseFilters(r)
       if err != nil {
           respondError(w, http.StatusBadRequest, err.Error())
           return
       }
       
       artists := h.store.FilterArtists(filters)
       options := h.store.GetArtistFilterOptions()
       
       page := view.NewArtistListPage(artists, options, filters)
       h.render(w, "artists.tmpl", page)
   }
   ```

4. **Update all handlers:**
   - Apply consistent pattern
   - Remove embedded business logic
   - Use helper functions

5. **Run tests:** `go test ./internal/web/...`

### Step 4.3: Clarify Middleware
**Goal:** Clean and reusable middleware chain.

#### Sub-steps:
1. **Create method restriction middleware:**
   ```go
   func methodOnly(method string) func(http.Handler) http.Handler {
       return func(next http.Handler) http.Handler {
           return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
               if r.Method != method {
                   w.Header().Set("Allow", method)
                   http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
                   return
               }
               next.ServeHTTP(w, r)
           })
       }
   }
   ```

2. **Review existing middleware:**
   - Logging middleware
   - Recovery middleware
   - Security headers middleware

3. **Establish clear middleware ordering:**
   ```go
   func (h *Handlers) setupMiddleware() http.Handler {
       mux := http.NewServeMux()
       // Register routes
       
       // Apply middleware in order
       var handler http.Handler = mux
       handler = h.logging(handler)
       handler = h.recovery(handler)
       handler = h.securityHeaders(handler)
       
       return handler
   }
   ```

4. **Document middleware purpose:**
   - Add clear comments
   - Document ordering requirements

5. **Run tests:** `go test ./internal/web/...`

### Step 4.4: Optimize Template Handling
**Goal:** Compile once, render efficiently.

#### Sub-steps:
1. **Update template compilation:**
   ```go
   type Templates struct {
       templates map[string]*template.Template
       funcMap   template.FuncMap
   }
   
   func LoadTemplates(dir string) (*Templates, error) {
       t := &Templates{
           templates: make(map[string]*template.Template),
           funcMap:   makeFuncMap(),
       }
       
       // Parse base template
       base := template.Must(template.New("base").Funcs(t.funcMap).ParseFiles(
           filepath.Join(dir, "base.tmpl"),
       ))
       
       // Parse all page templates
       files, err := filepath.Glob(filepath.Join(dir, "*.tmpl"))
       if err != nil {
           return nil, err
       }
       
       for _, file := range files {
           name := filepath.Base(file)
           if name == "base.tmpl" {
               continue
           }
           
           tmpl, err := base.Clone()
           if err != nil {
               return nil, err
           }
           
           tmpl, err = tmpl.ParseFiles(file)
           if err != nil {
               return nil, err
           }
           
           t.templates[name] = tmpl
       }
       
       return t, nil
   }
   ```

2. **Update render helper:**
   ```go
   func (h *Handlers) render(w http.ResponseWriter, name string, data interface{}) {
       tmpl, ok := h.templates.templates[name]
       if !ok {
           log.Printf("Template not found: %s", name)
           http.Error(w, "Internal Server Error", http.StatusInternalServerError)
           return
       }
       
       w.Header().Set("Content-Type", "text/html; charset=utf-8")
       
       if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
           log.Printf("Template execution error: %v", err)
           http.Error(w, "Internal Server Error", http.StatusInternalServerError)
       }
   }
   ```

3. **Add template functions:**
   ```go
   func makeFuncMap() template.FuncMap {
       return template.FuncMap{
           "add":      func(a, b int) int { return a + b },
           "sub":      func(a, b int) int { return a - b },
           "join":     strings.Join,
           "lower":    strings.ToLower,
           "title":    strings.Title,
           "slugify":  slugify,
           "contains": strings.Contains,
       }
   }
   ```

4. **Initialize templates once:**
   ```go
   func main() {
       templates, err := LoadTemplates("templates")
       if err != nil {
           log.Fatal(err)
       }
       
       handlers := web.NewHandlers(store, templates)
       // ...
   }
   ```

5. **Run tests:** `go test ./internal/web/...`

---

## Phase 5: Code Polish & Documentation

### Step 5.1: Remove Redundant Comments
**Goal:** Keep only rationale-level comments.

#### Sub-steps:
1. **Audit all comments:**
   ```bash
   grep -r "// " internal/
   ```

2. **Remove obvious comments:**
   ```go
   // BAD: Remove these
   // Get artist by ID
   func (s *Store) GetArtist(id int) (*Artist, error)
   
   // GOOD: Keep these
   // artistsByCountry builds a reverse index from country to artists.
   // This is precomputed to avoid O(n) scans on each filter operation.
   func artistsByCountry(artists []Artist) map[string][]int
   ```

3. **Keep comments for:**
   - Non-obvious design decisions
   - Performance considerations
   - Edge cases and gotchas
   - Public API documentation

4. **Update package-level docs:**
   ```go
   // Package data provides the core domain models and data access layer
   // for the Groupie Tracker application.
   package data
   ```

5. **Run `gofmt`:** `gofmt -w .`

### Step 5.2: Normalize Naming Conventions
**Goal:** Consistent, idiomatic Go names.

#### Sub-steps:
1. **Remove Get prefixes:**
   ```go
   // Before
   GetArtist()
   GetArtists()
   GetLocation()
   
   // After
   Artist()           // Single item by ID
   ArtistByID()       // Explicit lookup
   ArtistBySlug()     // Alternative lookup
   Artists()          // Multiple items
   AllArtists()       // All items
   ```

2. **Standardize method names:**
   - Use `ByID`, `BySlug` for lookups
   - Use `All` prefix for full collections
   - Use `Filter` prefix for filtered collections
   - Use `Search` for search operations

3. **Review and update:**
   ```bash
   # Find all Get* methods
   grep -r "func.*Get[A-Z]" internal/
   
   # Update each one
   ```

4. **Update all callers:**
   - Use find/replace carefully
   - Run tests after each batch

5. **Run tests:** `go test ./...`

### Step 5.3: Package Responsibility Review
**Goal:** Ensure each package has clear, focused responsibility.

#### Sub-steps:
1. **Review package structure:**
   ```
   internal/
   ├── api/       - External API client
   ├── data/      - Domain models, catalog, store, filters
   ├── view/      - View models for templates
   └── web/       - HTTP handlers, middleware, routing
   ```

2. **Verify package independence:**
   - `api` should not import `web` or `view`
   - `data` should not import `web` or `view`
   - `view` can import `data`
   - `web` can import `data` and `view`

3. **Move misplaced code:**
   - Identify code in wrong package
   - Move to appropriate location
   - Update imports

4. **Check for circular dependencies:**
   ```bash
   go list -f '{{ join .Deps "\n" }}' ./...
   ```

5. **Run tests:** `go test ./...`

### Step 5.4: Update Documentation
**Goal:** Reflect new architecture in docs.

#### Sub-steps:
1. **Update README.md:**
   - Project structure section
   - Architecture overview
   - Build/run instructions

2. **Create/update ARCHITECTURE.md:**
   ```markdown
   # Architecture
   
   ## Overview
   This application follows a clean, idiomatic Go architecture...
   
   ## Package Structure
   - `internal/api`: External API client
   - `internal/data`: Core domain and data access
   - `internal/view`: View models
   - `internal/web`: HTTP layer
   
   ## Data Flow
   1. API client fetches raw data
   2. Catalog normalizes and indexes
   3. Store provides filtered/searched access
   4. Handlers build view models
   5. Templates render HTML
   ```

3. **Create migration guide:**
   ```markdown
   # Migration Guide
   
   ## Changed APIs
   - `store.GetArtist()` → `store.Artist()`
   - `artist.ConcertCount` → `artist.ConcertCount()`
   ...
   ```

4. **Update doc/ folder:**
   - Add summary of refactoring
   - Link to this implementation plan
   - Archive old plans

5. **Review and commit:**
   ```bash
   git add doc/
   git commit -m "docs: update architecture documentation"
   ```

### Step 5.5: Code Quality Check
**Goal:** Ensure code quality standards.

#### Sub-steps:
1. **Run gofmt:**
   ```bash
   gofmt -w .
   ```

2. **Run go vet:**
   ```bash
   go vet ./...
   ```

3. **Run golint (if available):**
   ```bash
   golint ./...
   ```

4. **Check for common issues:**
   - Unused variables
   - Unused imports
   - Shadowed variables
   - Error handling

5. **Run all tests:**
   ```bash
   go test ./... -v
   ```

---

## Phase 6: Testing & Validation

### Step 6.1: Update Data Layer Tests
**Goal:** Test new helper methods and catalog.

#### Sub-steps:
1. **Test Artist helper methods:**
   ```go
   func TestArtist_MemberCount(t *testing.T) {
       tests := []struct{
           name     string
           artist   Artist
           expected int
       }{
           {"no members", Artist{}, 0},
           {"one member", Artist{Members: []string{"John"}}, 1},
           {"multiple members", Artist{Members: []string{"John", "Paul"}}, 2},
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got := tt.artist.MemberCount()
               if got != tt.expected {
                   t.Errorf("got %d, want %d", got, tt.expected)
               }
           })
       }
   }
   
   // Similar tests for:
   // - ConcertCount()
   // - FirstAlbumYear()
   // - Countries()
   // - Slug()
   ```

2. **Test Location helper methods:**
   ```go
   func TestLocation_ArtistCount(t *testing.T)
   func TestLocation_TotalConcerts(t *testing.T)
   func TestLocation_YearRange(t *testing.T)
   func TestLocation_Country(t *testing.T)
   ```

3. **Test Catalog:**
   ```go
   func TestCatalog_Build(t *testing.T)
   func TestCatalog_ArtistByID(t *testing.T)
   func TestCatalog_ArtistBySlug(t *testing.T)
   func TestCatalog_LocationBySlug(t *testing.T)
   ```

4. **Run data tests:**
   ```bash
   go test ./internal/data/... -v -cover
   ```

### Step 6.2: Test Filter Framework
**Goal:** Comprehensive filter test coverage.

#### Sub-steps:
1. **Test filter builders:**
   ```go
   func TestCreationYearBetween(t *testing.T)
   func TestHasMemberCount(t *testing.T)
   func TestInCountries(t *testing.T)
   func TestFirstAlbumYearBetween(t *testing.T)
   ```

2. **Test range and set helpers:**
   ```go
   func TestIntRange_Contains(t *testing.T)
   func TestIntRange_IsZero(t *testing.T)
   func TestStringSet_Contains(t *testing.T)
   func TestIntSet_Contains(t *testing.T)
   ```

3. **Test filter Match methods:**
   ```go
   func TestArtistFilters_Match(t *testing.T) {
       tests := []struct{
           name    string
           filters ArtistFilters
           artist  Artist
           want    bool
       }{
           {
               name: "empty filters match all",
               filters: ArtistFilters{},
               artist: Artist{ID: 1, Name: "Test"},
               want: true,
           },
           {
               name: "creation year filter",
               filters: ArtistFilters{
                   CreationYear: IntRange{Min: 2000, Max: 2010},
               },
               artist: Artist{ID: 1, CreationDate: 2005},
               want: true,
           },
           // More test cases
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got := tt.filters.Match(&tt.artist)
               if got != tt.want {
                   t.Errorf("got %v, want %v", got, tt.want)
               }
           })
       }
   }
   ```

4. **Test filter composition:**
   ```go
   func TestAndFilters(t *testing.T)
   ```

5. **Run filter tests:**
   ```bash
   go test ./internal/data/ -run TestFilter -v
   ```

### Step 6.3: Test Search and Token Index
**Goal:** Validate search functionality.

#### Sub-steps:
1. **Test token normalization:**
   ```go
   func TestNormalizeTokens(t *testing.T) {
       tests := []struct{
           input    string
           expected []string
       }{
           {"The Beatles", []string{"the", "beatles"}},
           {"AC/DC", []string{"ac", "dc"}},
           {"Maroon 5", []string{"maroon", "5"}},
       }
       
       for _, tt := range tests {
           t.Run(tt.input, func(t *testing.T) {
               got := normalizeTokens(tt.input)
               if !reflect.DeepEqual(got, tt.expected) {
                   t.Errorf("got %v, want %v", got, tt.expected)
               }
           })
       }
   }
   ```

2. **Test token matching:**
   ```go
   func TestMatchesTokens(t *testing.T)
   ```

3. **Test search index building:**
   ```go
   func TestCatalog_BuildSearchIndex(t *testing.T)
   ```

4. **Test search method:**
   ```go
   func TestStore_Search(t *testing.T) {
       // Create test catalog with known artists
       // Search for various queries
       // Assert correct results returned
   }
   ```

5. **Run search tests:**
   ```bash
   go test ./internal/data/ -run TestSearch -v
   ```

### Step 6.4: Update Handler Tests
**Goal:** Test new view models and handler flow.

#### Sub-steps:
1. **Test view model creation:**
   ```go
   func TestNewHomePage(t *testing.T)
   func TestNewArtistListPage(t *testing.T)
   func TestNewArtistDetailPage(t *testing.T)
   ```

2. **Test handler helpers:**
   ```go
   func TestRequireMethod(t *testing.T)
   func TestParseFilters(t *testing.T)
   func TestRespondJSON(t *testing.T)
   ```

3. **Test handlers:**
   ```go
   func TestHandleHome(t *testing.T) {
       store := setupTestStore()
       templates := setupTestTemplates()
       handlers := web.NewHandlers(store, templates)
       
       req := httptest.NewRequest("GET", "/", nil)
       w := httptest.NewRecorder()
       
       handlers.handleHome(w, req)
       
       if w.Code != http.StatusOK {
           t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
       }
   }
   
   // Similar tests for all handlers
   ```

4. **Test middleware:**
   ```go
   func TestMethodOnly(t *testing.T)
   func TestLoggingMiddleware(t *testing.T)
   func TestRecoveryMiddleware(t *testing.T)
   ```

5. **Run web tests:**
   ```bash
   go test ./internal/web/... -v -cover
   ```

### Step 6.5: Edge Case Testing
**Goal:** Cover edge cases and error paths.

#### Sub-steps:
1. **Test empty/zero-value inputs:**
   ```go
   func TestArtist_EmptyConcerts(t *testing.T)
   func TestFilters_EmptyFilters(t *testing.T)
   func TestSearch_EmptyQuery(t *testing.T)
   ```

2. **Test not found scenarios:**
   ```go
   func TestStore_ArtistNotFound(t *testing.T)
   func TestStore_LocationNotFound(t *testing.T)
   ```

3. **Test invalid inputs:**
   ```go
   func TestParseFilters_InvalidYear(t *testing.T)
   func TestParseFilters_InvalidMemberCount(t *testing.T)
   ```

4. **Test minimal dataset:**
   ```go
   func TestCatalog_SingleArtist(t *testing.T)
   func TestCatalog_NoLocations(t *testing.T)
   ```

5. **Run all tests:**
   ```bash
   go test ./... -v -cover
   ```

### Step 6.6: Integration and E2E Tests
**Goal:** Test full application flow.

#### Sub-steps:
1. **Update integration tests:**
   ```go
   // In tests/integration_test.go
   
   func TestFullDataLoad(t *testing.T) {
       // Test complete data loading pipeline
   }
   
   func TestFilterAndSearch(t *testing.T) {
       // Test filter + search combination
   }
   ```

2. **Update E2E tests:**
   ```go
   // In tests/e2e_test.go
   
   func TestHomePage(t *testing.T)
   func TestArtistList(t *testing.T)
   func TestArtistDetail(t *testing.T)
   func TestSearch(t *testing.T)
   ```

3. **Test complete user flows:**
   - Home → Artists → Detail
   - Home → Search → Results
   - Artists → Filter → Results
   - Artists → Detail → Adjacent

4. **Run integration tests:**
   ```bash
   go test ./tests/... -v
   ```

5. **Check coverage:**
   ```bash
   go test ./... -cover -coverprofile=coverage.out
   go tool cover -html=coverage.out -o coverage.html
   ```

### Step 6.7: Performance Benchmarking
**Goal:** Ensure performance is acceptable.

#### Sub-steps:
1. **Create benchmark tests:**
   ```go
   // In internal/data/benchmark_test.go
   
   func BenchmarkCatalogBuild(b *testing.B) {
       for i := 0; i < b.N; i++ {
           catalog := NewCatalog()
           // Add test data
           catalog.Build()
       }
   }
   
   func BenchmarkSearch(b *testing.B) {
       store := setupTestStore()
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           store.Search("beatles")
       }
   }
   
   func BenchmarkFilterArtists(b *testing.B) {
       store := setupTestStore()
       filters := ArtistFilters{
           CreationYear: IntRange{Min: 2000, Max: 2010},
       }
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           store.FilterArtists(filters)
       }
   }
   ```

2. **Run benchmarks:**
   ```bash
   go test ./internal/data/ -bench=. -benchmem
   ```

3. **Document results:**
   ```markdown
   # Performance Benchmarks
   
   ## Data Loading
   - Catalog build: X ms
   - Full data load: Y ms
   
   ## Search
   - Simple search: X μs
   - Complex search: Y μs
   
   ## Filtering
   - Single filter: X μs
   - Multiple filters: Y μs
   
   All operations are sub-millisecond on typical hardware.
   ```

4. **Set performance baselines:**
   - Document acceptable thresholds
   - Create regression tests if needed

5. **Commit results:**
   ```bash
   git add doc/PERFORMANCE_BENCHMARKS.md
   git commit -m "docs: add performance benchmark results"
   ```

---

## Phase 7: Rollout Strategy

### Step 7.1: Prepare for Rollout
**Goal:** Ensure smooth deployment.

#### Sub-steps:
1. **Create feature branch:**
   ```bash
   git checkout -b refactor/consolidated-simplification
   ```

2. **Commit incrementally:**
   - Each phase as separate commits
   - Clear commit messages
   - Reference issues/plan sections

3. **Run full test suite:**
   ```bash
   go test ./... -v -race -cover
   ```

4. **Update documentation:**
   - README.md
   - ARCHITECTURE.md
   - Migration guide

5. **Tag milestones:**
   ```bash
   git tag -a v2.0.0-beta.1 -m "Complete data model refactor"
   ```

### Step 7.2: Smoke Testing
**Goal:** Verify application works end-to-end.

#### Sub-steps:
1. **Build application:**
   ```bash
   go build -o groupie-tracker ./cmd/server/
   ```

2. **Run locally:**
   ```bash
   ./groupie-tracker
   ```

3. **Manual test checklist:**
   - [ ] Home page loads
   - [ ] Artist list displays
   - [ ] Filters work
   - [ ] Artist detail page loads
   - [ ] Adjacent navigation works
   - [ ] Location pages load
   - [ ] Search works
   - [ ] Suggestions work
   - [ ] No console errors
   - [ ] No broken images

4. **Test edge cases:**
   - [ ] Empty search
   - [ ] Non-existent artist
   - [ ] Non-existent location
   - [ ] All filters applied
   - [ ] Clear filters

5. **Document any issues:**
   - Create issues in tracker
   - Fix before merge

### Step 7.3: Performance Validation
**Goal:** Verify performance goals met.

#### Sub-steps:
1. **Measure startup time:**
   ```bash
   time ./groupie-tracker
   # Should be < 1 second
   ```

2. **Measure search performance:**
   - Open DevTools Network tab
   - Perform search
   - Check response time (should be < 100ms)

3. **Measure filter performance:**
   - Apply various filters
   - Check response time (should be < 100ms)

4. **Check memory usage:**
   ```bash
   # Use pprof or similar
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

5. **Document results:**
   - Compare to baselines
   - Note any regressions
   - Fix if needed

### Step 7.4: Code Review Preparation
**Goal:** Prepare for team review.

#### Sub-steps:
1. **Self-review:**
   - Read through all changes
   - Check for leftover debug code
   - Verify comments are helpful
   - Check formatting

2. **Create pull request:**
   ```markdown
   # Consolidated Idiomatic Go Refactoring
   
   ## Overview
   This PR implements the complete refactoring plan...
   
   ## Changes
   - Phase 1: Data model simplification
   - Phase 2: Filter framework
   - Phase 3: Search refactor
   - Phase 4: Web layer cleanup
   - Phase 5: Code polish
   - Phase 6: Testing
   
   ## Testing
   - All tests pass
   - Coverage: X%
   - Benchmarks show improvement
   
   ## Migration Notes
   - See MIGRATION.md for API changes
   ```

3. **Prepare demo:**
   - Screenshots or video
   - Show before/after
   - Highlight improvements

4. **Update changelog:**
   ```markdown
   # Changelog
   
   ## [2.0.0] - 2025-10-XX
   
   ### Changed
   - Complete refactoring to idiomatic Go
   - Simplified data model with helper methods
   - New filter framework
   - Improved search performance
   
   ### Removed
   - Complex caching layer
   - Redundant indexes
   - Unnecessary concurrency
   ```

5. **Request review:**
   - Tag reviewers
   - Link to documentation
   - Be available for questions

### Step 7.5: Post-Merge Tasks
**Goal:** Clean up after merge.

#### Sub-steps:
1. **Merge to main:**
   ```bash
   git checkout main
   git merge refactor/consolidated-simplification
   git push origin main
   ```

2. **Tag release:**
   ```bash
   git tag -a v2.0.0 -m "Consolidated idiomatic Go refactoring"
   git push origin v2.0.0
   ```

3. **Update deployment:**
   - Deploy to staging
   - Verify functionality
   - Deploy to production

4. **Archive old plans:**
   ```bash
   mkdir doc/archive
   mv doc/old-plan-*.md doc/archive/
   ```

5. **Celebrate:**
   - Document lessons learned
   - Share results with team
   - Plan next improvements

---

## Appendix

### A. Testing Checklist
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All E2E tests pass
- [ ] Coverage > 80%
- [ ] No race conditions (`go test -race`)
- [ ] Benchmarks run successfully

### B. Code Quality Checklist
- [ ] `gofmt` applied
- [ ] `go vet` passes
- [ ] No `golint` warnings
- [ ] No unused imports
- [ ] No unused variables
- [ ] Proper error handling

### C. Documentation Checklist
- [ ] README updated
- [ ] ARCHITECTURE.md updated
- [ ] Migration guide created
- [ ] API documentation current
- [ ] Comments are helpful
- [ ] Examples work

### D. Performance Checklist
- [ ] Data load < 1s
- [ ] Search < 100ms
- [ ] Filter < 100ms
- [ ] Memory usage reasonable
- [ ] No memory leaks

### E. Deployment Checklist
- [ ] Builds successfully
- [ ] Runs without errors
- [ ] All pages load
- [ ] All features work
- [ ] No console errors
- [ ] No broken images

---

## Notes

### Common Pitfalls
1. **Forgetting to update tests** - Always update tests with code
2. **Breaking changes without migration** - Document all API changes
3. **Premature optimization** - Profile before optimizing
4. **Over-engineering** - Keep it simple
5. **Incomplete error handling** - Check all error paths

### Best Practices
1. **Commit often** - Small, focused commits
2. **Test as you go** - Don't accumulate technical debt
3. **Document decisions** - Future you will thank you
4. **Ask for help** - Don't struggle alone
5. **Take breaks** - Fresh eyes catch more bugs

### Resources
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Testing Best Practices](https://go.dev/blog/testing)
- [Go Performance Tips](https://github.com/dgryski/go-perfbook)

---

**End of Implementation Plan**
