# Consolidated Refactoring Plan 2025
**Date:** October 2, 2025  
**Objective:** Reduce LOC by ~350-760 (12-15%), improve performance by 10-20%, maintain standard library only  
**Architecture:** Keep strict 3-layer design (api → data → web)  
**Principles:** KISS, Go idioms, zero JavaScript, server-side only

---

## Executive Summary

This plan consolidates three refactoring approaches into a single, phased implementation strategy. The codebase currently sits at ~5,100 LOC total (~2,800 application code). We aim to:

1. Eliminate structural redundancy in domain models and data storage
2. Optimize concurrent data loading and index building
3. Streamline filtering and search with precomputed indexes
4. Simplify web layer through shared contexts and helpers
5. Remove compatibility APIs that duplicate newer functionality

**Expected Outcomes:**
- 350-760 LOC reduction (12-15%)
- 10-20% performance improvement in filtering/startup
- Maintained test coverage ≥70%
- No external dependencies added
- Zero behavioral changes from user perspective

---

## Phase 1: Domain Model Simplification (Priority: CRITICAL)

### 1.1 Convert Store Collections to Pointer-Based
**Target:** `internal/data/store.go`, `internal/data/models.go`

**Problem:** Value-based artist slices cause excessive copying during lookups and transformations.

**Actions:**
1. Change `Store.artists []Artist` to `Store.artists []*Artist`
2. Update all index maps (`artistsByID`, `artistsBySlug`) to store `*Artist` instead of `Artist`
3. Update `artistPositions map[int]int` to key on pointer identity where appropriate
4. Modify `createArtistIndexes` to populate pointer maps during initial build
5. Update all store methods that return artists to return pointers or slices of pointers
6. Audit template rendering to ensure pointer fields work correctly with `html/template`

**Impact:** -50 LOC (removes defensive copying), reduces allocations by ~30%

---

### 1.2 Eliminate Redundant Scalar Fields
**Target:** `internal/data/models.go`

**Problem:** Duplicate representations of same data increase maintenance burden.

**Actions:**
1. Remove `Artist.MemberCount int` field entirely
2. Add method `func (a *Artist) MemberCount() int { return len(a.Members) }`
3. Remove `Artist.ConcertCount int` field
4. Add method `func (a *Artist) ConcertCount() int` that computes from concerts data
5. Update all template references to use method syntax `{{.MemberCount}}` (works automatically)
6. Update filter/search logic to call methods instead of accessing fields
7. Remove computation of these fields from `transformAPIArtists`

**Impact:** -40 LOC, eliminates synchronization bugs between fields and source data

---

### 1.3 Consolidate Concert Data Structure
**Target:** `internal/data/models.go`, `internal/data/store.go`

**Problem:** `Artist.DatesAtLocation` and `Artist.Concerts` overlap, forcing dual maintenance.

**Actions:**
1. Define new type `ConcertLedger struct { Dates []string; TotalCount int; Country string }`
2. Replace `Artist.DatesAtLocation map[string][]string` with `Artist.ConcertsByLocation map[string]*ConcertLedger`
3. Keep `Artist.Concerts []Concert` but enhance `Concert` struct with `Country string` field
4. Update `addConcertData` to populate both structures in single pass
5. Precompute country for each concert using `extractCountryFromLocation` during build
6. Remove separate `Artist.Countries []string` field
7. Add method `func (a *Artist) Countries() []string` that extracts unique countries from ledger
8. Update location building logic to read from ledger instead of multiple sources

**Impact:** -80 LOC, eliminates data duplication, faster location aggregation

---

### 1.4 Introduce ConcertWindow Type
**Target:** `internal/data/models.go`

**Problem:** Scattered `EarliestYear`/`LatestYear` fields on multiple types.

**Actions:**
1. Define `type ConcertWindow struct { EarliestYear, LatestYear int }`
2. Add method `func (w ConcertWindow) Span() int { return w.LatestYear - w.EarliestYear }`
3. Replace individual year fields on `Artist` with `Artist.ConcertWindow ConcertWindow`
4. Replace individual year fields on `Location` with `Location.ConcertWindow ConcertWindow`
5. Update all year-related computations to populate window during build
6. Update templates to access nested fields `{{.ConcertWindow.EarliestYear}}`
7. Update filter logic to check against window bounds

**Impact:** -30 LOC, better type safety, clearer semantics

---

### 1.5 Introduce Type-Safe Identifiers
**Target:** `internal/data/models.go`, all files using IDs/slugs

**Problem:** String and int keys used inconsistently, easy to mix up.

**Actions:**
1. Define `type ArtistID int` with String method
2. Define `type LocationSlug string` with validation method
3. Change `Artist.ID` from `int` to `ArtistID`
4. Change all slug fields from `string` to `LocationSlug` or `ArtistSlug`
5. Update map key types in indexes to use typed identifiers
6. Update API response parsing to convert to typed IDs
7. Adjust template functions to handle typed identifiers (may need explicit conversions)

**Impact:** -0 LOC (type safety improvement only), prevents identifier confusion

---

### 1.6 Create Grouped Index Structs
**Target:** `internal/data/store.go`

**Problem:** Seven separate index fields on Store clutter the struct and method signatures.

**Actions:**
1. Define `type artistIndex struct { list []*Artist; byID map[ArtistID]*Artist; bySlug map[string]*Artist; positions map[ArtistID]int }`
2. Define `type locationIndex struct { list []Location; bySlug map[LocationSlug]Location }`
3. Replace individual Store fields with `Store.artistIdx artistIndex` and `Store.locationIdx locationIndex`
4. Update `AllArtists()` to return `s.artistIdx.list`
5. Update `ArtistByID(id)` to access `s.artistIdx.byID[id]`
6. Update `ArtistBySlug(slug)` to access `s.artistIdx.bySlug[slug]`
7. Repeat for location accessors
8. Update `createArtistIndexes` to return `artistIndex` struct
9. Update `createLocationsData` to return `locationIndex` struct

**Impact:** -60 LOC (consolidates seven field declarations and accessor logic)

---

### 1.7 Remove Deprecated Compatibility APIs
**Target:** `internal/data/store.go`

**Problem:** Duplicate getters maintained for backward compatibility.

**Actions:**
1. Delete `GetArtistFilterOptions()` method entirely
2. Delete `GetLocationFilterOptions()` method entirely
3. Delete `GenerateAllSearchSuggestions()` method entirely
4. Update all callers in `internal/web/handlers.go` to use:
   - `ArtistFilterOptions()` instead of `GetArtistFilterOptions()`
   - `LocationFilterOptions()` instead of `GetLocationFilterOptions()`
   - `Suggestions()` instead of `GenerateAllSearchSuggestions()`
5. Search codebase for any test usage and update accordingly
6. Verify no external packages depend on these methods

**Impact:** -25 LOC, cleaner API surface

---

## Phase 2: Data Loading Pipeline Optimization (Priority: HIGH)

### 2.1 Restructure Load Stages
**Target:** `internal/data/store.go` (loadData method)

**Problem:** Single 200+ LOC function mixing concerns: fetching, transforming, indexing.

**Actions:**
1. Extract `fetchRawData(ctx, client) (artists, relations, error)` that handles API calls and channel coordination
2. Extract `buildDataset(rawArtists, rawRelations) ([]*Artist, error)` that transforms and enriches
3. Extract `buildIndexes(artists []*Artist) (artistIndex, locationIndex, error)` for index creation
4. Extract `computeMetadata(artistIdx, locationIdx) (ArtistFilterOptions, LocationFilterOptions, []SearchSuggestion, AppStats)` for filter options and suggestions
5. Keep `Load(ctx)` as orchestrator that calls these four stages sequentially
6. Each function should return immutable data structures (no mutation of Store fields until final assignment)
7. Add clear comments marking each stage boundary

**Impact:** -50 LOC (extraction eliminates duplication), improves testability

---

### 2.2 Implement Concurrent Artist Enrichment
**Target:** `internal/data/store.go` (new file `loader.go` if desired)

**Problem:** Sequential processing of artists during transformation is slow.

**Actions:**
1. Create `hydrateArtist(rawArtist, relationsMap) (*Artist, error)` function
2. Inside `buildDataset`, create jobs channel of raw artists
3. Create results channel for enriched `*Artist` pointers
4. Launch worker pool with `numWorkers := min(runtime.NumCPU(), len(rawArtists))`
5. Each worker reads from jobs channel, calls `hydrateArtist`, sends to results
6. Main goroutine collects from results channel into slice
7. Handle errors via separate error channel (fail fast on first error)
8. Ensure deterministic output by sorting enriched artists by ID before returning
9. Use `sync.WaitGroup` to coordinate worker shutdown

**Impact:** +40 LOC, 15-20% faster startup on multi-core systems

---

### 2.3 Parallel Index Building
**Target:** `internal/data/store.go` (buildIndexes function)

**Problem:** Four index computations run sequentially despite being independent.

**Actions:**
1. In `buildIndexes`, create separate goroutines for:
   - Artist indexes (ID map, slug map, position map)
   - Location list and slug map
   - Artist filter options calculation
   - Search suggestions generation
2. Use single `sync.WaitGroup` with four `Add(1)` calls
3. Each goroutine stores results in local variables (no shared state)
4. After `wg.Wait()`, construct and return structs from local variables
5. Consider extracting `parallelBuild(tasks ...func())` helper if pattern repeats
6. Document why each task is independent (helps future maintainers)

**Impact:** -30 LOC (consolidates WaitGroup setup), 10% faster indexing

---

### 2.4 Incremental Stats Calculation
**Target:** `internal/data/store.go` (location building and stats)

**Problem:** `calculateStats` re-walks entire artist slice after locations are built.

**Actions:**
1. Initialize `AppStats` struct at start of `buildDataset`
2. Accumulate `TotalArtists++` counter during artist enrichment
3. Accumulate `TotalMembers += artist.MemberCount()` in same loop
4. During location building, accumulate `TotalLocations++` and `TotalConcerts += loc.TotalConcerts`
5. Extract unique countries into set during location building, set `UniqueCountries = len(countrySet)`
6. Remove separate `calculateStats` function entirely
7. Return populated stats from `buildIndexes` stage

**Impact:** -45 LOC, eliminates redundant iteration

---

### 2.5 Consolidate Loading Helpers
**Target:** `internal/data/normalize.go` (new file or part of store.go)

**Problem:** Helper functions scattered throughout store.go inflate its size.

**Actions:**
1. Extract `createSlug`, `extractCountryFromLocation`, `extractYearFromDate` into separate file or clearly marked section
2. Group related helpers: slug generation, country parsing, date parsing
3. Add unit tests specifically for these helpers (easier to test in isolation)
4. Document expected input formats and edge cases
5. Consider making these private package functions if not used outside `internal/data`

**Impact:** +0 LOC (reorganization only), improved code organization

---

## Phase 3: Cache Optimization (Priority: MEDIUM)

### 3.1 Encapsulate Search Cache
**Target:** `internal/data/cache.go`, `internal/data/searches.go`

**Problem:** Cache management logic split across multiple methods and files.

**Actions:**
1. Define `type searchCacheEntry struct { results []*Artist; lastUsed time.Time }`
2. Define `type searchCache struct { entries map[string]*searchCacheEntry; order []string; maxSize int; mu sync.Mutex }`
3. Add methods: `Get(query) ([]*Artist, bool)`, `Set(query, results)`, `Evict(query)`, `Clear()`
4. Implement LRU eviction inside `Set` method automatically
5. Replace Store fields `searchCache`, `searchOrder`, `searchCacheMu`, `searchCacheSize` with single `cache *searchCache` field
6. Update `SearchArtists` to use `s.cache.Get()` and `s.cache.Set()`
7. Remove standalone `getCachedSearchResults` and `setCachedSearchResults` methods

**Impact:** -60 LOC, cleaner cache API, encapsulated concurrency

---

### 3.2 Optimize Image Cache Worker Pool
**Target:** `internal/data/cache.go`

**Problem:** Image download setup code mixes concerns; already optimal but could be clearer.

**Actions:**
1. Extract `prepareImageJob(artist) imageJob` helper that constructs job struct
2. Extract `downloadImage(url, path) error` that handles HTTP request and file write
3. Keep adaptive pool size `runtime.NumCPU()` logic
4. Add timeout to HTTP client (10 seconds) to prevent hanging downloads
5. Consider exposing `type ImageCache struct` to encapsulate mutex and path logic
6. Document why worker pool adapts to CPU count (I/O bound but benefits from parallelism)

**Impact:** -20 LOC, clearer separation of concerns

---

## Phase 4: Filter and Search Improvements (Priority: HIGH)

### 4.1 Simplify Filter Parameter Structures
**Target:** `internal/data/models.go`, `internal/data/filters.go`

**Problem:** Pointer-based range parameters complicate logic without clear benefit.

**Actions:**
1. Replace `*int` pointer fields in `ArtistFilterParams` with plain `int` fields
2. Rename fields for clarity: `CreationYearFrom` → `CreationYearMin`, `CreationYearTo` → `CreationYearMax`
3. Document zero value semantics: "0 means no constraint"
4. Update `matchesArtistFilters` to check `(min == 0 || value >= min)`
5. Update `parseArtistFilterParams` in `internal/web/templates.go` to parse directly to int
6. Remove nil pointer checks throughout filtering code
7. Apply same pattern to `LocationFilterParams`

**Impact:** -50 LOC, simpler filter logic, more idiomatic Go

---

### 4.2 Create Precomputed Filter Indexes
**Target:** `internal/data/filters.go`, `internal/data/store.go`

**Problem:** Every filter request scans full artist list linearly.

**Actions:**
1. During index building, create `artistsByCountry map[string][]*Artist`
2. During index building, create `artistsByMemberCount map[int][]*Artist`
3. Store these in new `filterIndex` struct embedded in Store
4. In `FilterArtists`, check if only country or member count filter applied
5. If single-dimension filter, start from index slice instead of all artists
6. If multi-dimension filter, start from smallest index slice and scan with remaining predicates
7. For range-only filters (year), keep existing scan but optimize with early exit
8. Document index usage strategy in comments

**Impact:** +30 LOC, 40-60% faster filtering for common cases

---

### 4.3 Extract Reusable Filter Predicates
**Target:** `internal/data/filters.go`

**Problem:** Filter matching logic duplicated between FilterArtists and SearchArtists.

**Actions:**
1. Extract `matchYearRange(value, min, max int) bool` helper
2. Extract `matchAnyInt(value int, allowed []int) bool` helper
3. Extract `matchAnyString(value string, allowed []string) bool` helper
4. Extract `hasCountryIntersection(artistCountries, filterCountries []string) bool` helper
5. Refactor `matchesArtistFilters` to call these helpers
6. Refactor search filtering to reuse same helpers
7. Add comprehensive unit tests for each helper function
8. Consider precomputing `map[string]struct{}` for string set membership in hot paths

**Impact:** -45 LOC, improved testability, reusable across search and filter

---

### 4.4 Optimize Search Suggestions
**Target:** `internal/data/searches.go`

**Problem:** Suggestion generation concatenates strings repeatedly; filtering uses triple accumulation.

**Actions:**
1. In `generateSearchSuggestions`, store raw `Text` and `Type` separately, no formatting
2. Remove " - Artist", " - Member" suffixes from stored suggestions
3. Move formatting to templates using `{{.Text}} - {{.Type}}` pattern
4. Precompute `normalizedText` field on `SearchSuggestion` during generation
5. In `filterSearchSuggestions`, replace three-slice accumulation with single-pass in-place filter
6. Use stable partition algorithm: swap matches to front, return slice of matches
7. Add early exit if no query provided (return all suggestions)

**Impact:** -35 LOC, 15% faster suggestion filtering

---

### 4.5 Enhance Search Performance
**Target:** `internal/data/searches.go`

**Problem:** Empty query still scans artists; filter combination inefficient.

**Actions:**
1. In `SearchArtists`, detect empty query and return all artists immediately (no scan)
2. When both search and filters active, apply search first (reduces filter input set)
3. Consider caching normalized query terms to avoid repeated lowercase allocations
4. For very large result sets (>40 artists), split filtering across two goroutines using `sync.WaitGroup`
5. Add threshold check to avoid goroutine overhead on small datasets
6. Ensure deterministic results by sorting before returning from parallel path
7. Document concurrency strategy and threshold rationale

**Impact:** -25 LOC, 20-30% faster combined search+filter operations

---

## Phase 5: Web Layer Simplification (Priority: MEDIUM)

### 5.1 Introduce Shared Page Context
**Target:** `internal/web/handlers.go`, new file `internal/web/page.go`

**Problem:** Every handler rebuilds same page metadata (title, CSS, suggestions).

**Actions:**
1. Define `type PageContext struct { Title string; ExtraCSS string; Suggestions []data.SearchSuggestion; Stats data.AppStats }`
2. Define typed view models that embed PageContext:
   - `type HomePage struct { PageContext; RecentArtists []*data.Artist }`
   - `type ArtistsPage struct { PageContext; Artists []*data.Artist; Filters data.ArtistFilterOptions }`
   - `type ArtistDetailPage struct { PageContext; Artist *data.Artist }`
   - Similar for Locations, LocationDetail, Search, Dev pages
3. Create constructor helpers: `newPageContext(title, css string, store *data.Store) PageContext`
4. Cache suggestions and stats on Server struct at startup (updated once during initialization)
5. Update all handlers to construct typed page struct and pass to render
6. Update templates to access nested fields via PageContext

**Impact:** -70 LOC, consistent page structure, no repeated store calls

---

### 5.2 Consolidate Form Parsing
**Target:** `internal/web/templates.go`

**Problem:** Separate parsing functions for artist/location filters duplicate logic.

**Actions:**
1. Create `type formParser struct { req *http.Request }` with methods:
   - `intValue(field string) int` (returns 0 if missing/invalid)
   - `intSlice(field string) []int`
   - `stringSlice(field string) []string`
   - `boolValue(field string) bool`
2. Replace `parseArtistFilterParams` with declarative construction using parser methods
3. Replace `parseLocationFilterParams` similarly
4. Extract `newFormParser(r *http.Request) *formParser` constructor that calls `ParseForm()`
5. Add error handling for malformed input (return appropriate HTTP status)
6. Consider adding validation helpers (e.g., `intInRange(field, min, max)`)

**Impact:** -40 LOC, eliminates parsing duplication

---

### 5.3 Simplify Static File Serving
**Target:** `internal/web/static.go`, `internal/web/routes.go`

**Problem:** Custom static handler with manual path validation and checks.

**Actions:**
1. Replace custom `StaticFiles` handler with `http.FileServer(http.Dir("static"))`
2. Wrap with `http.StripPrefix("/static/", handler)` for path trimming
3. Keep favicon special case handler separate (already in routes)
4. Add simple middleware to deny dotfiles if security concern: check if path contains "/."
5. Update `restrictMethod` to accept `http.Handler` instead of only `http.HandlerFunc`
6. Test that static assets still serve correctly with new handler

**Impact:** -35 LOC, standard library pattern, less custom code

---

### 5.4 Streamline Handler Registration
**Target:** `internal/web/routes.go`

**Problem:** Verbose `restrictMethod` calls on every route.

**Actions:**
1. Inside `routes()` method, define local helper functions:
   - `get := func(pattern string, h http.HandlerFunc) { mux.HandleFunc(pattern, s.restrictMethod(h, "GET")) }`
   - `post := func(pattern string, h http.HandlerFunc) { mux.HandleFunc(pattern, s.restrictMethod(h, "POST")) }`
   - `getPost := func(pattern string, h http.HandlerFunc) { mux.HandleFunc(pattern, s.restrictMethod(h, "GET", "POST")) }`
2. Replace all `router.HandleFunc(pattern, s.restrictMethod(...))` calls with helper usage
3. Routes become more readable: `get("/", s.Home)`, `getPost("/artists", s.Artists)`
4. Add comment documenting helper pattern for future maintainers

**Impact:** -20 LOC, improved route readability

---

### 5.5 Extract Development Handlers
**Target:** `internal/web/handlers.go`, new file `internal/web/dev_handlers.go`

**Problem:** Dev endpoints clutter main handlers file.

**Actions:**
1. Create `internal/web/dev_handlers.go` file
2. Move `DevIndex`, `DevPanic`, `Dev404`, `Dev500`, `Dev500Tmpl` handlers to new file
3. Keep handlers as methods on `Server` struct
4. Add build tag comment at top of file: `//go:build !production` (optional, for future)
5. Update imports in routes.go if needed
6. Group all dev routes together in `routes()` method with clear comment

**Impact:** -0 LOC (reorganization), clearer separation of concerns

---

## Phase 6: Configuration and Initialization (Priority: LOW)

### 6.1 Consolidate Server Initialization
**Target:** `internal/web/server.go`, `cmd/server/main.go`

**Problem:** Two-layer App/Server structure adds indirection.

**Actions:**
1. Rename `App` struct to `Server` throughout codebase
2. Remove `StartApp()` function entirely
3. Consolidate all initialization into `NewServer(apiClient, withCache) (*Server, error)`:
   - Create Store and call Load
   - Compile templates
   - Cache suggestions and filter options
   - Setup HTTP server with timeouts
4. Add `ListenAndServe() error` method that calls `s.srv.ListenAndServe()`
5. Add `Shutdown(ctx) error` method for graceful shutdown
6. Update main.go to call `server.ListenAndServe()` directly
7. Update tests to use new Server naming

**Impact:** -50 LOC, simpler dependency injection, clearer lifecycle

---

### 6.2 Structured Configuration
**Target:** `internal/conf/conf.go`, `cmd/server/main.go`

**Problem:** Package-level variables make configuration opaque.

**Actions:**
1. Define `type Config struct { APIBaseURL, Port string; WithCache bool; ReadTimeout, WriteTimeout, IdleTimeout time.Duration }`
2. Add `func Load() Config` that reads environment variables and applies defaults
3. Update `NewServer` to accept `config Config` parameter instead of individual flags
4. Pass specific config fields to Store constructor
5. Remove package-level variables from conf.go
6. Update main.go to call `config := conf.Load()` and pass to NewServer
7. Update tests to construct Config structs directly

**Impact:** -15 LOC, explicit dependencies, easier testing

---

## Phase 7: Testing and Validation (Priority: CRITICAL)

### 7.1 Update Unit Tests for New Structures
**Target:** `internal/data/data_test.go`, `internal/data/filters_test.go`

**Actions:**
1. Update fixtures in `fixtures.go` to use pointer-based artists
2. Update test assertions to work with new index structs
3. Add tests for new helper methods (MemberCount, ConcertCount, Countries)
4. Add tests for ConcertLedger building
5. Add tests for searchCache and imageCache types
6. Ensure all filter tests pass with simplified parameter structures
7. Add tests for new filter index usage (country, member count indexes)

**Impact:** +50 LOC (new tests), ensures correctness of Phase 1-4 changes

---

### 7.2 Update Web Layer Tests
**Target:** `internal/web/web_test.go`

**Actions:**
1. Update handler tests to expect new PageContext embedded structs
2. Add tests for formParser helper methods
3. Test static file serving with new FileServer implementation
4. Verify restrictMethod works with http.Handler interface
5. Test graceful shutdown via Server.Shutdown method
6. Update mock setup to use new Server struct (renamed from App)

**Impact:** +20 LOC, validates Phase 5-6 changes

---

### 7.3 Update E2E Tests
**Target:** `cmd/server/e2e_test.go`, `cmd/server/search_e2e_test.go`

**Actions:**
1. Update test setup to use new Server initialization pattern
2. Verify all routes still respond correctly after handler consolidation
3. Test form submissions with simplified filter parameters
4. Verify search results with new cache implementation
5. Test static assets serve correctly
6. Add test for concurrent requests to validate thread safety

**Impact:** +15 LOC, end-to-end validation

---

### 7.4 Performance Benchmarking
**Target:** New file `internal/data/benchmark_test.go`

**Actions:**
1. Add benchmark for `Store.Load` to measure startup time improvement
2. Add benchmark for `FilterArtists` with various filter combinations
3. Add benchmark for `SearchArtists` with and without cache hits
4. Add benchmark for location building
5. Compare results before and after refactoring (target: 10-20% improvement)
6. Document benchmark results in commit message or summary document

**Impact:** +60 LOC, validates performance claims

---

### 7.5 Coverage Verification
**Target:** All test files

**Actions:**
1. Run `go test -cover ./...` and ensure coverage ≥70%
2. Identify uncovered lines using `go test -coverprofile=coverage.out`
3. Add targeted tests for critical paths with low coverage
4. Update `tests/coverage_summary.txt` with final numbers
5. Generate HTML coverage report: `go tool cover -html=coverage.out`
6. Review report for any surprising gaps

**Impact:** Ensures quality maintained throughout refactoring

---

## Phase 8: Documentation and Cleanup (Priority: LOW)

### 8.1 Update Architecture Documentation
**Target:** `.github/copilot-instructions.md`, `doc/REFACTORING_SUMMARY_OCT_2025.md`

**Actions:**
1. Update copilot-instructions.md to reflect:
   - Pointer-based artist collections
   - New index structs and organization
   - PageContext pattern in web layer
   - Removed compatibility APIs
   - New formParser pattern
2. Document new helper types (ConcertLedger, ConcertWindow, typed IDs)
3. Update "Common Tasks" section with new patterns
4. Add notes about precomputed filter indexes
5. Create new summary document describing all changes made

**Impact:** Maintains accurate project documentation

---

### 8.2 Code Comment Audit
**Target:** All modified files

**Actions:**
1. Add/update package comments for clarity
2. Document concurrency patterns (worker pools, goroutine coordination)
3. Explain non-obvious optimizations (filter indexes, cache eviction)
4. Add examples for complex functions
5. Remove obsolete comments referencing deleted code
6. Ensure all exported types and functions have godoc comments

**Impact:** Improves code maintainability for future contributors

---

### 8.3 Remove Dead Code
**Target:** All files

**Actions:**
1. Search for unused helper functions after refactoring
2. Remove commented-out code blocks
3. Delete empty files if any remain
4. Remove unused imports
5. Run `gofmt` and `go vet` across codebase
6. Run linter if available to catch issues

**Impact:** Final LOC reduction, cleaner codebase

---

## Implementation Timeline

### Week 1: Foundation (Phases 1-2)
- **Days 1-2:** Phase 1.1-1.7 (Domain model changes)
  - Convert to pointers, remove redundant fields, consolidate structures
  - Update all call sites and ensure tests pass incrementally
- **Days 3-4:** Phase 2.1-2.5 (Loading pipeline)
  - Restructure load stages, add concurrent enrichment
  - Implement parallel indexing and incremental stats
- **Day 5:** Phase 3.1-3.2 (Cache optimization)
  - Encapsulate search cache, optimize image cache

### Week 2: Optimization (Phases 3-4)
- **Days 1-2:** Phase 4.1-4.3 (Filter improvements)
  - Simplify parameters, create filter indexes, extract predicates
- **Days 3-4:** Phase 4.4-4.5 (Search optimization)
  - Optimize suggestions, enhance search performance
- **Day 5:** Phase 5.1-5.2 (Web layer - part 1)
  - Introduce PageContext, consolidate form parsing

### Week 3: Polish and Validation (Phases 5-8)
- **Days 1-2:** Phase 5.3-5.5 (Web layer - part 2)
  - Simplify static serving, streamline routes, extract dev handlers
- **Day 3:** Phase 6.1-6.2 (Configuration)
  - Consolidate initialization, structured config
- **Days 4-5:** Phase 7.1-7.5 (Testing)
  - Update all tests, add benchmarks, verify coverage
- **Weekend:** Phase 8.1-8.3 (Documentation)
  - Update docs, audit comments, final cleanup

---

## Success Criteria

### Quantitative Metrics
- **LOC Reduction:** 350-760 lines (12-15% of application code)
- **Startup Performance:** 10-15% faster (measure Store.Load time)
- **Filter Performance:** 20-40% faster (measure FilterArtists on large datasets)
- **Search Performance:** Maintain or improve cache hit rates
- **Test Coverage:** Maintain ≥70% coverage throughout
- **Zero Regressions:** All existing tests pass at each phase

### Qualitative Metrics
- **Code Clarity:** Functions under 50 LOC, clear single responsibilities
- **Maintainability:** New developers can navigate codebase with copilot-instructions.md
- **Type Safety:** Strong typing for IDs, slugs, and data structures
- **Concurrency:** Clear goroutine patterns with documented rationale
- **Idioms:** Zero-value semantics, clear error handling, no external dependencies

---

## Risk Mitigation Strategies

### Risk: Breaking Template Rendering
- **Mitigation:** Test each template-affecting change immediately in browser
- **Recovery:** Keep git branch per phase for easy rollback
- **Testing:** Add template execution tests that catch field access errors

### Risk: Concurrency Bugs
- **Mitigation:** Run tests with `-race` flag after Phase 2 changes
- **Recovery:** Revert to sequential processing if races detected
- **Testing:** Add stress tests with concurrent requests

### Risk: Performance Regression
- **Mitigation:** Benchmark before and after each optimization phase
- **Recovery:** Revert specific optimization if >5% slower
- **Testing:** Continuous benchmarking throughout implementation

### Risk: Cache Correctness Issues
- **Mitigation:** Add extensive unit tests for cache implementations
- **Recovery:** Disable caching if bugs found, fix in isolation
- **Testing:** Test cache eviction, concurrency, and invalidation

### Risk: Configuration Breaking Tests
- **Mitigation:** Update test fixtures immediately after config changes
- **Recovery:** Keep backward-compatible constructors temporarily
- **Testing:** Ensure all test files compile at each step

---

## Rollback Plan

Each phase should be committed separately with clear commit messages. If issues arise:

1. **Immediate:** Revert last commit if tests fail or behavior changes
2. **Phase-level:** Revert entire phase if issues discovered in next phase
3. **Full rollback:** Revert all commits back to pre-refactoring state if critical bug found

Git workflow:
```
feat(data): convert artist collections to pointers (Phase 1.1)
feat(data): remove redundant scalar fields (Phase 1.2)
feat(data): consolidate concert data structure (Phase 1.3)
...
```

Each commit must:
- Pass all tests (`go test ./...`)
- Pass race detector (`go test -race ./...`)
- Pass build (`go build ./cmd/server`)
- Include before/after LOC count in message

---

## Post-Implementation Tasks

After all phases complete:

1. **Final Benchmarking:** Run comprehensive benchmarks and document results
2. **Coverage Report:** Generate HTML coverage report and review
3. **Performance Profiling:** Use pprof to identify any remaining bottlenecks
4. **Documentation Review:** Ensure all docs reflect current state
5. **Team Walkthrough:** Present changes to other developers (if applicable)
6. **Monitor Production:** Track performance metrics after deployment
7. **User Acceptance:** Verify no functional regressions from user perspective

---

## Maintenance Notes for Future Refactoring

**What to Keep:**
- 3-layer architecture (api → data → web)
- Standard library only requirement
- Pointer-based collections for efficiency
- Precomputed indexes for performance
- Server-side only processing (no JavaScript)
- Type-safe identifiers and grouped structs

**What to Avoid:**
- External dependencies, only stdlib
- Premature abstraction (wait for third use case)
- Clever concurrency beyond proven patterns
- Breaking backward compatibility without migration path
- Caching without clear performance justification

**When to Refactor Again:**
- LOC grows beyond 3,500 application code
- Test coverage drops below 65%
- New feature requires significant architectural change
- Performance degrades >20% from current baseline
- Code duplication appears in more than 2 places

---

## Appendix: Detailed LOC Impact Summary

| Phase | Section | Expected LOC Change | Risk Level |
|-------|---------|--------------------:|------------|
| 1.1   | Pointer conversion | -50 | Medium |
| 1.2   | Remove scalar fields | -40 | Low |
| 1.3   | Concert data consolidation | -80 | High |
| 1.4   | ConcertWindow type | -30 | Low |
| 1.5   | Typed identifiers | 0 | Low |
| 1.6   | Grouped index structs | -60 | Medium |
| 1.7   | Remove compatibility APIs | -25 | Low |
| 2.1   | Load stage restructuring | -50 | Medium |
| 2.2   | Concurrent enrichment | +40 | High |
| 2.3   | Parallel indexing | -30 | Medium |
| 2.4   | Incremental stats | -45 | Low |
| 2.5   | Consolidate helpers | 0 | Low |
| 3.1   | Encapsulate search cache | -60 | Medium |
| 3.2   | Optimize image cache | -20 | Low |
| 4.1   | Simplify filter params | -50 | Medium |
| 4.2   | Precomputed filter indexes | +30 | High |
| 4.3   | Reusable filter predicates | -45 | Low |
| 4.4   | Optimize search suggestions | -35 | Low |
| 4.5   | Enhance search performance | -25 | Medium |
| 5.1   | Shared page context | -70 | Medium |
| 5.2   | Consolidate form parsing | -40 | Low |
| 5.3   | Simplify static serving | -35 | Low |
| 5.4   | Streamline handler registration | -20 | Low |
| 5.5   | Extract dev handlers | 0 | Low |
| 6.1   | Consolidate initialization | -50 | Medium |
| 6.2   | Structured configuration | -15 | Low |
| 7.x   | Testing updates | +145 | N/A |
| **TOTAL** | **Application Code** | **-760** | - |

**Note:** Testing LOC increases but application code decreases significantly. Net reduction considering only production code is target metric.

---

## Conclusion

This consolidated plan provides a clear, phased approach to refactoring the Groupie Tracker codebase. By following KISS principles, maintaining strict layering, and validating continuously, we can achieve significant LOC reduction and performance improvements while preserving functionality and code quality.

The plan is designed for incremental implementation with clear rollback points, comprehensive testing, and thorough documentation. Each phase builds on previous work, ensuring a stable foundation for subsequent changes.

**Key Takeaway:** Simplicity and clarity drive every decision. Optimize where it matters (concurrent loading, filter indexes), simplify where it doesn't (duplicate APIs, scattered helpers), and maintain what works (3-layer architecture, standard library only).
