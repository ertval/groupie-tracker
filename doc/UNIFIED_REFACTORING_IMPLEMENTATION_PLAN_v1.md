# Unified Refactoring Implementation Plan
**Target:** Groupie Tracker Codebase  
**Goal:** Reduce LOC by ~12-15% (~350-760 lines), improve performance 10-20%, maintain functionality  
**Approach:** 6 sequential phases with validation checkpoints  
**Constraint:** Standard library only, zero JavaScript, server-side architecture

---

## Overview

This plan consolidates three refactoring proposals into a single implementation roadmap. The focus is on eliminating redundancy, simplifying data structures, optimizing concurrent operations, and streamlining the web layer while maintaining the existing 3-layer architecture (API client → Data/Business Logic → HTTP handlers).

**Core Principles:**
- Use pointer-backed collections to reduce value copying
- Eliminate duplicate accessor methods and type aliases
- Simplify filter parameters using zero values instead of pointers
- Consolidate initialization into fewer, clearer functions
- Extract reusable helpers to reduce duplication
- Pre-compute indexes and metadata at startup
- Maintain immutable data after initial load

---

## Phase 1: Domain Model Simplification (Priority: CRITICAL)

### 1.1 Switch Artist Collections to Pointers
**Files:** `internal/data/store.go`, `internal/data/models.go`

Change `Store.artists` from `[]Artist` to `[]*Artist` to eliminate expensive struct copying during filtering and search operations. Update all index maps to hold pointers instead of values. This cascades to helper functions but dramatically reduces allocations.

Update `artistsByID`, `artistsBySlug`, and `artistPositions` to use `*Artist` values. Adjust all accessor methods and iteration loops to work with pointers.

**Validation:** Run existing tests in `internal/data/data_test.go` to ensure pointer semantics don't break equality checks or template rendering.

---

### 1.2 Remove Duplicate Accessor Methods
**Files:** `internal/data/store.go`

Delete redundant getter pairs that expose the same data with different names. Keep only the simpler, non-prefixed versions following Go conventions.

Remove these methods:
- `GetArtistFilterOptions()` - keep `ArtistFilterOptions()`
- `GetLocationFilterOptions()` - keep `LocationFilterOptions()`
- `GenerateAllSearchSuggestions()` - keep `Suggestions()`

Update all call sites in `internal/web/handlers.go` to use the retained methods.

**Validation:** Grep for deleted method names to ensure no orphaned calls remain. Confirm handlers compile.

---

### 1.3 Flatten Stats Structure
**Files:** `internal/data/models.go`

Replace the type alias pattern with a single direct struct definition. Remove `type storeStats struct` and the `type AppStats = storeStats` alias. Define `AppStats` directly with all fields inline.

Update the `Stats()` method in `store.go` to return the unified `AppStats` type. Ensure JSON tags remain correct for the health endpoint.

**Validation:** Check `/health` endpoint returns proper JSON structure. Verify type name appears correctly in error messages.

---

### 1.4 Simplify Filter Parameter Structures
**Files:** `internal/data/models.go`, `internal/data/filters.go`, `internal/web/templates.go`

Replace pointer-based range filters with zero-value semantics. Change all `*int` range fields to plain `int` where zero means "no constraint."

Transform `ArtistFilterParams` and `LocationFilterParams`:
- `CreationYearFrom *int` → `CreationYearMin int`
- `CreationYearTo *int` → `CreationYearMax int`
- Same pattern for first album year, concert counts, member counts

Update `matchesArtistFilters()` and `matchesLocationFilters()` to use simple `(min == 0 || value >= min)` checks instead of pointer nil tests.

Update form parsing functions in `templates.go` to call `p.int()` instead of `p.intPtr()`.

**Validation:** Test filter forms manually in browser. Ensure empty fields behave as "no constraint" and populated fields filter correctly. Run filter unit tests.

---

### 1.5 Introduce Typed Index Structs
**Files:** `internal/data/store.go`, `internal/data/models.go`

Create focused index structures to group related fields and clarify responsibilities:

Define `artistIndex` struct with fields: `list []*Artist`, `byID map[int]*Artist`, `bySlug map[string]*Artist`, `positions map[int]int`.

Define `locationIndex` struct with fields: `list []Location`, `bySlug map[string]Location`.

Embed these into `Store` struct. Update initialization in `loadData()` to populate index structs instead of individual Store fields.

Modify accessor methods to delegate to index struct fields. This reduces scattered field management and makes the Store struct smaller.

**Validation:** Ensure `ArtistByID()`, `ArtistBySlug()`, `AllArtists()` continue working. Test location lookups in detail pages.

---

## Phase 2: Store Loading & Concurrency Optimization (Priority: HIGH)

### 2.1 Extract Loading Stages
**Files:** `internal/data/store.go`

Break the monolithic `loadData()` method into three clear stages with explicit return values and error handling:

Create `fetchRawData(ctx)` that handles API calls and returns raw artist/relation data structures.

Create `buildDataset(rawArtists, rawRelations)` that transforms API models into domain models and returns enriched artist slice.

Create `finalizeIndexes(artists)` that builds all indexes, filter options, and suggestions, returning structured results.

Update `Load()` to orchestrate these stages sequentially with clear error propagation.

**Validation:** Run E2E tests to ensure data loads correctly. Check that errors from each stage surface properly.

---

### 2.2 Parallel Artist Enrichment
**Files:** `internal/data/store.go`

After raw data fetch, process artists concurrently using a bounded worker pool. Create helper function `enrichArtists()` that:

Calculates worker count as `min(runtime.NumCPU(), len(artists))`.

Uses a jobs channel to distribute artist enrichment work. Each worker runs `hydrateArtist()` which builds concert ledger, derives stats, normalizes countries.

Collects results into output slice maintaining original order. Uses WaitGroup for synchronization.

Replace the sequential loop in `transformAPIArtists` with this parallel pipeline.

**Validation:** Benchmark startup time before/after. Verify artists list ordering stays consistent. Run with `-race` flag.

---

### 2.3 Consolidate Index Building
**Files:** `internal/data/store.go`

Reduce the number of separate goroutines in Stage 4 by grouping related computations. Currently uses 4-5 independent goroutines; consolidate to 2-3.

Group artist index building (byID, bySlug, positions) into single goroutine.

Group filter options calculation (artists and locations) into single goroutine.

Keep suggestions generation in separate goroutine since it's independent.

Use single WaitGroup to coordinate completion. Store results in the new index structs defined in Phase 1.5.

**Validation:** Verify all indexes populate correctly. Test slug-based artist/location lookups in detail pages.

---

### 2.4 Incremental Stats Collection
**Files:** `internal/data/store.go`

Instead of walking the artist slice again in `calculateStats()`, accumulate statistics during the enrichment phase.

Create a stats accumulator struct with atomic counters that workers update during `hydrateArtist()`.

Track total members, total concerts, unique countries using concurrent-safe operations.

Build final `AppStats` from accumulator after enrichment completes, eliminating the separate stats loop.

**Validation:** Compare stats output before/after refactor. Ensure counts match exactly for members, concerts, countries.

---

### 2.5 Simplify Cache Management
**Files:** `internal/data/cache.go`, `internal/data/searches.go`

Create dedicated cache types with clear interfaces to encapsulate locking and eviction logic.

Define `searchCache` struct with methods: `get(query)`, `set(query, results)`, `touch(query)`. Internalize the order slice and LRU eviction logic.

Define `imageCache` struct with methods: `load()`, `path(slug)`. Encapsulate file path logic and atomic download tracking.

Replace inline cache operations in `SearchArtists()` and image loading with these typed cache calls.

**Validation:** Test search result caching with repeated queries. Verify LRU eviction happens at size limit. Check image cache prevents duplicate downloads.

---

## Phase 3: Filtering & Search Optimization (Priority: MEDIUM)

### 3.1 Extract Reusable Filter Helpers
**Files:** `internal/data/filters.go`

Create small, composable helper functions to eliminate duplication across filter implementations:

Define `inRange(value, min, max int) bool` for range checks with zero-value handling.

Define `containsInt(slice []int, value int) bool` for membership tests.

Define `hasIntersection(slice1, slice2 []string) bool` for country/location matching.

Refactor `matchesArtistFilters()` and `matchesLocationFilters()` to use these helpers, removing nested conditionals.

**Validation:** Run filter unit tests in `internal/data/filters_test.go`. Test edge cases like empty slices and zero bounds.

---

### 3.2 Pre-compute Country Sets
**Files:** `internal/data/store.go`, `internal/data/models.go`

During artist enrichment, convert the `Countries []string` slice to `CountriesSet map[string]struct{}` and store on each Artist.

Use this set in `matchesArtistFilters()` for O(1) intersection checks instead of nested loops.

Update templates to range over `Countries` slice for display (leave slice in place for compatibility).

**Validation:** Benchmark filter performance before/after. Ensure country filters return same results but faster.

---

### 3.3 Optimize Empty Search Fast Path
**Files:** `internal/data/searches.go`

Add early return in `SearchArtists()` when query is empty. Return the full pre-indexed artist list directly instead of scanning.

For non-empty queries, reuse a shared lowercase comparison buffer to reduce allocations during string matching.

Pre-size result slices with `make([]Artist, 0, len(artists))` to minimize reallocation during append operations.

**Validation:** Benchmark search with empty queries. Test that filtering-only requests skip unnecessary search logic.

---

### 3.4 Simplify Suggestion Filtering
**Files:** `internal/data/searches.go`

Store suggestions with normalized text pre-computed during generation. Add `NormalizedText string` field to `SearchSuggestion`.

Replace triple-slice accumulation in `filterSearchSuggestions()` with in-place stable partition using single pass.

Move type formatting (" - Artist", " - Member", etc.) into templates instead of concatenating in Go code.

**Validation:** Test suggestion endpoint returns correctly formatted results. Verify filtering by type still works.

---

### 3.5 Combine Search and Filter Pipelines
**Files:** `internal/data/searches.go`

When `SearchArtists()` receives both query and filters, apply them in single pass instead of two separate loops.

Create unified predicate that combines search term matching and filter criteria matching.

Cache results only when filters are empty (maintain current cache key semantics).

**Validation:** Test search + filter combinations. Ensure results match previous behavior but execute faster.

---

## Phase 4: Web Layer Streamlining (Priority: MEDIUM)

### 4.1 Create Shared Page Context
**Files:** `internal/web/handlers.go`, `internal/web/templates.go`, `internal/data/models.go`

Define base `PageContext` struct with common fields: `Title string`, `ExtraCSS string`, `Suggestions []SearchSuggestion`.

Create typed view models that embed `PageContext`: `ArtistsPage`, `LocationsPage`, `ArtistDetailPage`, etc.

Add helper constructors: `NewArtistsPage(artists, filters)`, `NewLocationsPage(locations, filters)`.

Update handlers to use these constructors instead of building anonymous structs inline.

**Validation:** Render all pages manually. Ensure titles, CSS, and suggestions appear correctly. Test that search bar in navbar works.

---

### 4.2 Cache Metadata on App Startup
**Files:** `internal/web/server.go`, `internal/web/handlers.go`

Add fields to `Server` struct: `suggestions []SearchSuggestion`, `artistFilters ArtistFilterOptions`, `locationFilters LocationFilterOptions`.

Populate these in `NewServer()` after store loads. Use in handlers instead of calling store methods on every request.

This reduces repeated store calls from O(requests) to O(1) at startup.

**Validation:** Verify filter dropdowns populate correctly. Check suggestions API still returns current data (not stale).

---

### 4.3 Simplify Form Parsing
**Files:** `internal/web/templates.go`

Create generic form parser struct with reusable methods for common patterns:

Define `formParser` type with methods: `int(field) int`, `intSlice(field) []int`, `stringSlice(field) []string`.

Consolidate `parseArtistFilterParams()` and `parseLocationFilterParams()` to use these helpers.

Reduce duplication in field extraction and type conversion logic.

**Validation:** Submit filter forms with various combinations. Test empty fields, single values, multiple selections.

---

### 4.4 Replace Custom Static Handler
**Files:** `internal/web/static.go`, `internal/web/routes.go`

Remove custom `StaticFiles` handler implementation with filesystem checks.

Use standard library: `http.StripPrefix("/static/", http.FileServer(http.Dir("static")))`.

Wrap with simple middleware to deny dotfiles if needed.

Keep special favicon handler separate if required for root path serving.

**Validation:** Test static asset loading: CSS files, images, favicon. Verify paths resolve correctly. Check 404s for missing files.

---

### 4.5 Consolidate Handler Route Registration
**Files:** `internal/web/routes.go`

Create helper functions for HTTP method restrictions to reduce repetitive `restrictMethod` calls:

Define `get()`, `post()`, `getPost()` helpers that wrap `mux.HandleFunc()` with method enforcement.

Use these helpers in `routes()` function to make method constraints more visible and reduce line noise.

**Validation:** Test that GET/POST restrictions still work. Ensure 405 errors return for wrong methods.

---

### 4.6 Separate Development Handlers
**Files:** `internal/web/handlers.go`, create `internal/web/dev_handlers.go`

Move all `/dev/*` endpoints to new file: `DevIndex`, `DevPanic`, `Dev404`, `Dev500`, `Dev500Tmpl`.

Keep production handlers in main `handlers.go`: `Home`, `Artists`, `ArtistDetail`, `Locations`, `LocationDetail`, `Search`.

This makes the main handler file smaller and production routes easier to scan.

**Validation:** Test dev endpoints still work when accessed. Ensure production pages unaffected.

---

## Phase 5: Configuration & Initialization (Priority: LOW)

### 5.1 Consolidate App and Server Types
**Files:** `internal/web/server.go`, `cmd/server/main.go`

Merge `App` and `Server` concepts into single `Server` type. Remove wrapper layer.

Move initialization logic from separate `StartApp()` into `NewServer()` constructor.

Return configured `*Server` with embedded HTTP server ready to start.

Add `ListenAndServe()` method that starts the server and handles errors.

**Validation:** Run main.go and verify server starts. Test graceful shutdown. Ensure E2E tests still work with new initialization.

---

### 5.2 Introduce Config Struct
**Files:** `internal/conf/conf.go`, `cmd/server/main.go`

Replace package-level variables with struct-based configuration.

Define `Config` struct with fields: `APIBaseURL`, `Port`, `WithCache`, `ReadTimeout`, `WriteTimeout`, etc.

Create `Load()` function that reads environment variables and returns populated config.

Pass config to `NewServer()` instead of relying on package globals.

**Validation:** Test environment variable overrides work. Verify default values apply when env vars unset.

---

## Phase 6: Testing & Validation (Priority: ONGOING)

### 6.1 Update Unit Tests
**Files:** `internal/data/data_test.go`, `internal/data/filters_test.go`, `internal/web/web_test.go`

Update fixtures in `fixtures.go` to work with pointer-based artist collections.

Modify filter tests to use new zero-value parameter semantics.

Add tests for new cache types and helper functions.

Ensure coverage remains at or above 70% threshold.

**Validation:** Run `go test ./internal/data -v -cover`. Check coverage report. Fix any failing assertions.

---

### 6.2 Update E2E Tests
**Files:** `tests/e2e_test.go`, `tests/integration_test.go`

Update test server initialization to use new `NewServer()` constructor.

Verify pointer-based responses don't break JSON serialization in API endpoints.

Test filter and search combinations through HTTP layer.

**Validation:** Run `go test ./tests -v`. Ensure all E2E scenarios pass. Check for race conditions with `-race` flag.

---

### 6.3 Performance Benchmarks
**Files:** Create `internal/data/benchmark_test.go`

Add benchmarks for critical paths:

Benchmark `Store.Load()` to measure startup time improvements.

Benchmark `FilterArtists()` with various parameter combinations.

Benchmark `SearchArtists()` with empty queries and complex queries.

Compare results before/after refactoring to validate performance gains.

**Validation:** Run `go test -bench=. -benchmem`. Target 10-15% faster startup, 20% faster filtering.

---

### 6.4 Manual Browser Testing
**Files:** N/A (manual testing)

Test all pages and interactions in browser:

Home page loads with correct stats and search bar.

Artists page: filter by year ranges, member counts, countries. Pagination works.

Artist detail page: shows concerts, members, correct image.

Locations page: filter by concert dates, continents. Sorting works.

Location detail page: shows artists who played there.

Search functionality: autocomplete suggestions, result highlighting.

**Validation:** Create checklist and mark each feature as working. Test on different browsers if possible.

---

## Success Criteria

### Quantitative Targets
- **LOC Reduction:** 350-760 lines removed (~12-15% reduction from ~2,800 application LOC)
- **Performance:** Startup time -10-15%, filter operations -20%, search maintains speed with caching
- **Test Coverage:** Maintain ≥70% coverage across all packages
- **Memory:** Reduce allocations by 15-20% during filtering and search operations

### Qualitative Targets
- All existing functionality preserved (zero breaking changes for end users)
- Clearer separation of concerns (3-layer architecture more obvious)
- Smaller, focused functions (no function over 100 LOC except templates)
- Better error messages and handling (proper context wrapping)
- Improved code readability (reduced nesting, clearer naming)

### Validation Gates
After each phase, verify:
- All tests pass: `go test ./...`
- No race conditions: `go test -race ./...`
- Coverage maintained: `go test -cover ./...`
- Server starts and responds: Manual smoke test
- No new errors in logs during typical usage

---

## Implementation Timeline

**Phase 1:** 2-3 days (foundation, highest impact on LOC reduction)  
**Phase 2:** 2-3 days (concurrency work, needs careful testing)  
**Phase 3:** 1-2 days (optimization, straightforward changes)  
**Phase 4:** 1-2 days (web layer, visible changes need manual testing)  
**Phase 5:** 1 day (configuration, low complexity)  
**Phase 6:** Ongoing (testing concurrent with each phase)

**Total Estimated Time:** 7-11 days with thorough testing

---

## Rollback Strategy

Each phase should be implemented in a separate Git branch with format: `refactor/phase-N-description`.

Before starting Phase N+1, ensure Phase N is:
- Fully tested (automated and manual)
- Merged to main branch
- Tagged with version: `refactor-phase-N-complete`

If issues discovered in Phase N during Phase N+1:
- Revert Phase N+1 changes
- Fix Phase N issues
- Re-run full test suite
- Resume Phase N+1

Critical rollback points:
- After Phase 1 (domain model changes affect everything)
- After Phase 2 (concurrency bugs can be subtle)
- After Phase 4 (user-facing changes)

---

## Notes for Implementation Agent

**Standard Library Only:** Do not introduce external dependencies. Use only `sync`, `context`, `http`, `html/template` from standard library.

**Zero JavaScript Rule:** All filtering, search, and interactivity must remain server-side with HTML forms and POST requests.

**Template Compatibility:** When changing struct fields, verify templates using `go test` that renders templates. Add accessor methods if needed to avoid template breakage.

**Pointer Semantics:** When switching to pointers, ensure nil checks where appropriate. Templates can handle nil pointers gracefully, but methods cannot.

**Concurrency Safety:** Always use `-race` flag during testing. Prefer immutable data structures after initialization. Use mutexes only for caches that change during runtime.

**Error Handling:** Use `fmt.Errorf("context: %w", err)` for wrapping. Never swallow errors. Log at appropriate levels.

**Testing Priority:** Unit tests for data layer, E2E tests for handlers, manual testing for templates. Cover happy paths first, then edge cases.

**Documentation:** Update inline comments when changing function signatures. Keep package-level documentation minimal. Code should be self-documenting with clear naming.
