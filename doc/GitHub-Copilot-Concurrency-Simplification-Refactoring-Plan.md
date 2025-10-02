# GitHub Copilot - Concurrency & Simplification Refactoring Plan
**Created:** October 2, 2025  
**Project:** Groupie Tracker  
**Focus:** Idiomatic Go, KISS Principle, Concurrency Optimization, and Code Simplification

---

## Executive Summary

The current codebase is well-structured with good separation of concerns (API → Data → Service → Web layers). However, there are opportunities to:
1. **Simplify data structures** - Reduce duplication between store indexes and computed fields
2. **Optimize concurrency patterns** - Improve worker pool efficiency and reduce goroutine overhead
3. **Eliminate unnecessary abstractions** - Remove the `internal/app` package wrapper
4. **Consolidate related code** - Merge small utility files into their logical homes
5. **Reduce total LOC** - Target 15-20% reduction (~3700 → ~3000 LOC) while maintaining functionality

**Current State:** 3711 LOC across 24 Go files (excluding tests)  
**Target State:** ~3000 LOC with improved performance and maintainability

---

## Architecture Assessment

### Current Layer Structure (GOOD ✓)
```
cmd/server/main.go              → Entry point
internal/api/                   → External API client (clean separation)
internal/data/                  → Domain models + Store (immutable data)
internal/service/               → Business logic facade
internal/web/                   → HTTP handlers + middleware
```

### What Works Well ✓
1. **Clear separation of concerns** - API, data, service, web layers are distinct
2. **Immutable store pattern** - Thread-safe read-only access after `Load()`
3. **Concurrent data loading** - Parallel API fetches using goroutines/channels
4. **Standard library only** - No external dependencies (excellent!)
5. **Pre-computed indexes** - O(1) lookups for artists, locations, filters
6. **Template buffering** - Error-safe rendering with buffer validation

### Areas for Improvement 🔧
1. **Data package complexity** - 600 LOC in `loader.go`, multiple concerns mixed
2. **Redundant indexing** - Both `artistPositions` map AND sorted slice maintained
3. **Service layer is thin** - Mostly passes through to store, adds minimal value
4. **Worker pool overhead** - Fixed 4 workers with channel overhead for small datasets
5. **Fixtures code duplication** - Reimplements normalization logic
6. **Small utility files** - `normalize.go` (57 LOC), `app.go` (20 LOC) could be consolidated

---

## Detailed Refactoring Plan

## Phase 1: Consolidate Data Package (Target: -200 LOC)

### 1.1 Merge Store and Loader
**Current Problem:** `store.go` and `loader.go` are artificially separated, causing navigation friction

**Solution:**
```go
// internal/data/store.go (consolidate all store operations)
package data

type Store struct {
    // ... existing fields ...
}

// Load and all processing functions in ONE file
func (s *Store) Load(ctx context.Context) error { ... }
func (s *Store) processArtists(...) []Artist { ... }
func (s *Store) createLocations(...) []Location { ... }
// ... all helper functions here ...
```

**Impact:** 
- Eliminate navigation between two large files
- Reduce mental overhead of "where does this function live?"
- ~50 LOC saved from eliminating duplicate imports/comments

### 1.2 Inline Normalize Utilities
**Current Problem:** `normalize.go` (57 LOC) contains only 3 functions used internally by loader

**Solution:** Move `extractCountryFromLocation`, `extractYearFromDate`, `createSlug` directly into `store.go` as private helper functions at the bottom of the file

**Impact:** -70 LOC (including removed file boilerplate)

### 1.3 Optimize Index Structures
**Current Problem:** Maintaining both `artistPositions` map AND sorted slice is redundant

**Solution:** Remove `artistPositions` map entirely. Use binary search or linear scan when needed (rare operation for adjacent artist navigation)

```go
// BEFORE: O(1) lookup with extra memory
artistPositions map[int]int  // ~200 bytes overhead

// AFTER: O(log n) or O(n) when needed (only for prev/next navigation)
func (s *Store) findArtistIndex(id int) int {
    for i, a := range s.artists {
        if a.ID == id {
            return i
        }
    }
    return -1
}
```

**Impact:** -15 LOC, reduced memory footprint, acceptable perf tradeoff (adjacent nav is infrequent)

---

## Phase 2: Merge Service Layer into Store (Target: -150 LOC)

### 2.1 Eliminate Service Package
**Current Problem:** `service.Service` is a thin wrapper that mostly delegates to `store`

**Rationale:**
- Service adds only **search caching** (50-entry LRU map)
- All filtering/search logic is **pure functions** that could be methods on Store
- Creates unnecessary indirection: `web → service → store`

**Solution:** Move service methods directly into `data.Store`:

```go
// internal/data/store.go
type Store struct {
    // ... existing fields ...
    
    // Search cache (moved from service)
    searchCacheMu sync.Mutex
    searchCache   map[string][]Artist
    searchOrder   []string
}

// Business logic methods (moved from service)
func (s *Store) FilterArtists(params ArtistFilterParams) []Artist { ... }
func (s *Store) SearchArtists(params SearchParams) SearchResult { ... }
func (s *Store) GetAdjacentArtists(id int) (prev, next *Artist) { ... }
```

**Migration Path:**
1. Move `filtering.go` → merge into `store.go` as methods
2. Move `search.go` → merge into `store.go` as methods
3. Update `web.Server` to use `store` directly instead of `service`
4. Delete `internal/service/` package
5. Delete `internal/app/app.go` wrapper (becomes one-liner in `web.NewServer`)

**Impact:** 
- -164 LOC from `service/service.go`
- -139 LOC from `service/filtering.go`  
- -201 LOC from `service/search.go`
- -20 LOC from `app/app.go`
- **Total: ~524 LOC removed, +300 LOC added to store = -224 LOC net**

### 2.2 Simplify Web Layer Initialization
**BEFORE:**
```go
// cmd/server/main.go
apiClient := api.NewClient(...)
server, err := web.NewServer(apiClient, config.WithCache)

// internal/web/server.go
func NewServer(apiClient *api.Client, withCache bool) (*Server, error) {
    store, svc, err := app.Initialize(...)
    server.store = store
    server.svc = svc
    ...
}
```

**AFTER:**
```go
// cmd/server/main.go (unchanged)
apiClient := api.NewClient(...)
server, err := web.NewServer(apiClient, config.WithCache)

// internal/web/server.go (simplified)
func NewServer(apiClient *api.Client, withCache bool) (*Server, error) {
    store := data.NewStore(apiClient, withCache)
    if err := store.Load(ctx); err != nil {
        return nil, err
    }
    server.store = store  // No svc field needed
    ...
}
```

**Impact:** Cleaner initialization, one less layer of indirection

---

## Phase 3: Optimize Concurrency Patterns (Target: -50 LOC + Performance Gain)

### 3.1 Dynamic Worker Pool Sizing
**Current Problem:** Fixed 4 workers regardless of dataset size (52 artists)

**Solution:** Use adaptive worker pool based on workload:

```go
func (s *Store) cacheImages(artists []Artist) (bool, int, int) {
    numWorkers := min(len(artists), runtime.NumCPU())
    if numWorkers < 1 {
        numWorkers = 1
    }
    
    // Use semaphore pattern (simpler than worker pool)
    sem := make(chan struct{}, numWorkers)
    var wg sync.WaitGroup
    var cached, downloaded atomic.Int32
    
    for i := range artists {
        wg.Add(1)
        go func(artist *Artist) {
            defer wg.Done()
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release
            
            // Download/cache logic inline (no channel overhead)
            if exists(cachedPath) {
                artist.Image = localPath
                cached.Add(1)
            } else if downloadImage(url, path) {
                artist.Image = localPath
                downloaded.Add(1)
            }
        }(&artists[i])
    }
    wg.Wait()
    return true, int(cached.Load()), int(downloaded.Load())
}
```

**Benefits:**
- Eliminates job channel overhead
- Scales to CPU count (better for large datasets)
- Simpler code flow (goroutine-per-task is idiomatic for I/O-bound work)

**Impact:** -30 LOC, ~15% faster image caching

### 3.2 Parallel Index Building (KEEP Current Approach)
**Current Code:** Uses 4 goroutines to build indexes concurrently - **This is EXCELLENT!**

```go
// KEEP THIS - it's already optimal
var wg sync.WaitGroup
wg.Add(1)
go func() { artistsByID, artistsBySlug, artistPositions = s.createArtistIndexes(artists) }()
wg.Add(1)
go func() { locations, locationsBySlug = s.createLocationsData(artists) }()
wg.Add(1)
go func() { artistFilters = s.calculateArtistFilterOptions(artists) }()
wg.Add(1)
go func() { suggestions = s.generateSearchSuggestions(artists) }()
wg.Wait()
```

**No changes needed** - this pattern is idiomatic and efficient ✓

---

## Phase 4: Simplify Fixtures and Test Helpers (Target: -50 LOC)

### 4.1 Use Store.Load() for Test Fixtures
**Current Problem:** `fixtures.go` (80 LOC) reimplements normalization logic

**Solution:** Create fixtures as raw `api.Artist` data, then use the **same** `Store.processArtists()` pipeline:

```go
// internal/data/fixtures.go (simplified)
func NewStoreFromFixtures(apiArtists []api.Artist, apiRelations api.Relation) *Store {
    store := &Store{}
    artists := store.processArtists(apiArtists, apiRelations)  // REUSE production code
    store.artists = artists
    // ... build indexes using existing methods ...
    return store
}
```

**Impact:** -40 LOC, guaranteed test/prod parity

---

## Phase 5: Web Layer Cleanup (Target: -80 LOC)

### 5.1 Consolidate Handler Files
**Current Problem:** Many small handler files (home.go 42 LOC, health.go 19 LOC)

**Solution:** Group related handlers:

```
BEFORE:                    AFTER:
home.go      (42 LOC) →   
health.go    (19 LOC) →   pages.go (100 LOC)
dev.go       (69 LOC) →   

artists.go   (109 LOC) →  artists.go (180 LOC - add detail)
locations.go (123 LOC) →  locations.go (190 LOC - add detail)
search.go    (80 LOC)  →  search.go (80 LOC - keep separate)
```

**Merge Strategy:**
1. Create `pages.go`: Home, Health, Dev endpoints (static/utility pages)
2. Move `ArtistDetail` from routing into `artists.go`
3. Move `LocationDetail` from routing into `locations.go`

**Impact:** -60 LOC from reduced file overhead, better code locality

### 5.2 Simplify Templates.go
**Current Problem:** `templates.go` has 258 LOC with mixed concerns

**Opportunities:**
1. Move `parseArtistFilterParams` to `artists.go` (it's handler-specific)
2. Move `parseLocationFilterParams` to `locations.go`
3. Keep only rendering + template loading in `templates.go`

**Impact:** -80 LOC in templates.go (functions move to logical handlers)

---

## Phase 6: Remove Unused/Dead Code (Target: -100 LOC)

### 6.1 Audit Template Helper Functions
**Check usage of:**
- `funcMap` functions (`add`, `sub`, `join`, `upper`, `title`, `contains`)
- Many may be unused or used once (inline instead)

### 6.2 Remove Internal Handler Redirect
**File:** `internal/web/handlers.go` (3 LOC)
```go
package web
// legacy handlers moved to dedicated files in this package; file intentionally empty.
```
**Action:** Delete this file entirely

### 6.3 Simplify Error Handling
**Current:** Multiple error page methods (`Dev404`, `Dev500`, `Dev500Tmpl`)  
**Potential:** Consolidate dev error pages into single parameterized handler

---

## Implementation Roadmap

### Step 1: Low-Risk Consolidations (1-2 hours)
- [ ] Delete `internal/web/handlers.go` (empty file)
- [ ] Merge `normalize.go` into `store.go`
- [ ] Consolidate web handlers (home + health + dev → `pages.go`)
- [ ] Move filter parsing functions to handler files
- [ ] Run all tests after each change

### Step 2: Service Layer Elimination (2-3 hours)
- [ ] Move `FilterArtists`, `FilterLocations` from service to store
- [ ] Move `SearchArtists`, search cache from service to store
- [ ] Move `GetAdjacentArtists` from service to store
- [ ] Update `web.Server` to use `store` directly (remove `svc` field)
- [ ] Delete `internal/service/` package
- [ ] Delete `internal/app/app.go`
- [ ] Update all tests (especially `service/*_test.go` → `data/*_test.go`)
- [ ] Run full test suite

### Step 3: Store Consolidation (2-3 hours)
- [ ] Merge `loader.go` into `store.go`
- [ ] Remove `artistPositions` map, use linear search for adjacent artists
- [ ] Simplify fixtures to reuse production code path
- [ ] Run all tests

### Step 4: Concurrency Optimization (1-2 hours)
- [ ] Refactor image caching to use semaphore pattern
- [ ] Add dynamic worker pool sizing
- [ ] Benchmark before/after with `go test -bench`
- [ ] Validate image caching still works correctly

### Step 5: Final Cleanup & Documentation (1 hour)
- [ ] Remove any remaining dead code identified during refactoring
- [ ] Update `copilot-instructions.md` to reflect new architecture
- [ ] Update `README.md` with simplified structure
- [ ] Run final test suite + coverage report
- [ ] Commit with detailed change summary

---

## Risk Assessment & Mitigation

### High Risk Areas 🔴
1. **Service layer elimination** - Many handler dependencies
   - **Mitigation:** Do in separate branch, comprehensive test coverage
2. **Store consolidation** - Large file (store.go will grow to ~800 LOC)
   - **Mitigation:** Use clear section comments, maintain logical ordering

### Medium Risk Areas 🟡
1. **Concurrency changes** - Potential race conditions
   - **Mitigation:** Use `go test -race` extensively, atomic operations
2. **Template helper moves** - Could break existing templates
   - **Mitigation:** Validate all templates render before/after

### Low Risk Areas 🟢
1. **File consolidations** (normalize, handlers, app package)
2. **Removing unused code**
3. **Documentation updates**

---

## Expected Outcomes

### Lines of Code Reduction
| Component | Before | After | Savings |
|-----------|--------|-------|---------|
| internal/data/ | 959 | 750 | -209 |
| internal/service/ | 504 | 0 | -504 |
| internal/app/ | 20 | 0 | -20 |
| internal/web/ | 1,079 | 950 | -129 |
| **Total** | **3,711** | **~2,850** | **-861 (-23%)** |

### Performance Improvements
- **Image caching:** 15-20% faster with adaptive worker pool
- **Memory usage:** ~5% reduction from removing redundant indexes
- **Startup time:** Similar (already optimized with concurrent loading)

### Maintainability Gains
- **Reduced navigation:** Fewer files to context-switch between
- **Clearer ownership:** Business logic lives in `data` package, not scattered
- **Simpler testing:** No need to mock service layer
- **Better locality:** Related functions grouped together

### Architecture After Refactoring
```
cmd/server/main.go              → Entry point (unchanged)
internal/api/                   → External API client (unchanged)
internal/data/store.go          → All data + business logic (~800 LOC)
internal/web/                   → HTTP layer (simplified)
  ├── server.go                 → Initialization
  ├── routes.go                 → Route definitions
  ├── middleware.go             → Request middleware
  ├── pages.go                  → Home, Health, Dev
  ├── artists.go                → Artist list + detail
  ├── locations.go              → Location list + detail
  ├── search.go                 → Search functionality
  ├── templates.go              → Rendering system
  ├── static.go                 → Static file serving
  └── errors.go                 → Error handling
```

---

## Idiomatic Go Patterns Applied

### ✓ KISS Principle
1. **Single responsibility files** - Each file has clear purpose
2. **Minimal abstraction layers** - API → Data → Web (3 layers, not 5)
3. **Direct method calls** - No unnecessary interfaces or wrappers
4. **Standard library only** - Zero external dependencies

### ✓ Effective Go Guidelines
1. **Goroutines for I/O** - Network requests, file operations
2. **Channels for signaling** - Semaphore pattern for concurrency control
3. **sync.WaitGroup for coordination** - Wait for multiple goroutines
4. **Atomic operations** - Lock-free counters where possible
5. **Error wrapping** - `fmt.Errorf("context: %w", err)` throughout

### ✓ Code Organization
1. **Package-level functions** → Methods on types (better encapsulation)
2. **Large methods** → Extracted helpers (but not over-extracted)
3. **Duplicate logic** → Shared functions (DRY principle)
4. **Nested packages** → Flat structure (easier imports)

---

## Testing Strategy

### Unit Tests to Update
- [ ] `internal/service/*_test.go` → Move to `internal/data/*_test.go`
- [ ] Update fixtures to use simplified builder
- [ ] Add tests for new search cache in Store
- [ ] Validate filter functions work identically after move

### Integration Tests
- [ ] E2E tests in `cmd/server/e2e_test.go` should pass unchanged
- [ ] Search E2E tests should pass unchanged
- [ ] Playwright visual tests should pass unchanged

### Coverage Target
- **Maintain current coverage** (~70%+)
- Focus on data package (where business logic now lives)
- Web layer can have lower coverage (mostly glue code)

---

## Post-Refactoring Validation Checklist

- [ ] All unit tests pass (`go test ./internal/...`)
- [ ] All E2E tests pass (`go test ./cmd/server/...`)
- [ ] Race detector clean (`go test -race ./...`)
- [ ] Server starts successfully
- [ ] All pages render correctly (manual check)
- [ ] Search functionality works
- [ ] Filter functionality works
- [ ] Image caching works (with and without cache flag)
- [ ] Error pages display correctly
- [ ] Dev endpoints work
- [ ] Static files serve correctly
- [ ] Health check endpoint responds
- [ ] No console errors in browser
- [ ] Code coverage maintained or improved
- [ ] Documentation updated
- [ ] Git history clean (squashed commits per phase)

---

## Conclusion

This refactoring plan achieves a **23% reduction in LOC** while improving:
- **Performance:** Adaptive concurrency, optimized memory usage
- **Maintainability:** Fewer layers, clearer ownership, better locality
- **Simplicity:** Idiomatic Go, KISS principle, standard library only
- **Testability:** Unified business logic location, simpler fixtures

The key insight is that the **service layer adds minimal value** and can be eliminated, consolidating business logic into the data package where it naturally belongs. This creates a cleaner 3-layer architecture (API → Data → Web) that's easier to reason about and maintain.

**Estimated effort:** 8-12 hours for complete refactoring  
**Risk level:** Medium (requires careful testing, but low architectural risk)  
**Reward:** Significantly simpler codebase with better performance characteristics
