# GitHub Copilot Refactoring Plan

## Objectives
- Collapse legacy compatibility layers while keeping the codebase idiomatic, testable, and easy to maintain.
- Preserve existing features (server-side filtering, suggestions, stats) with simpler, single-responsibility components.
- Leverage concurrency where it meaningfully lowers startup and request latency without overcomplicating control flow.
- Reduce duplication of data access logic so the web layer consumes pre-computed, read-only data in constant time.

## Current Pain Points
- **Triple layering (`Store ➜ Repository ➜ Service`)** duplicates APIs, requires redundant state (`Repository.syncFromStore`), and forces tests to reach through wrappers.
- **Derived data is scattered**: `Store`, `Repository`, and handlers each compute pieces of stats, filter options, and navigation hints.
- **Concurrency stops at fetch**: we already fetch artists/relations in parallel, but aggregation (indexing, statistics, cache priming) runs sequentially even though it operates on disjoint data.
- **Web layer carries caches** (suggestions, filter options, search cache) that would be cleaner and safer inside the data layer once data is immutable.
- **Testing hooks (`SetTestData`)** exist only for the compatibility wrapper and can be replaced with constructor helpers once the store API is streamlined.

## Target Architecture
1. **Data Package (`internal/data`)**
   - Single exported type `Store` that owns the immutable dataset post-load.
   - Public constructor `data.Load(ctx, client, config)` returning `*Store` to encapsulate concurrency, image caching, and derivations.
   - Store exposes read-only getters (`Artists()`, `ArtistBySlug()`, `Locations()`, `Stats()`, `Suggestions()`, `ArtistFilters()`, `LocationFilters()`).
   - Internal files (`fetch.go`, `transform.go`, `derive.go`, `cache.go`) keep responsibilities small while living in one package.

2. **Service Layer (`internal/service`)**
   - Thin façade focused on business operations that require orchestration (filtering, search, adjacency helpers).
   - Methods operate on the pre-computed store data without mutating it; filtering/search logic moves here so handlers call `service.FilterArtists(...)` instead of touching store internals.
   - No duplicated state—service holds a pointer to the store and nothing else.

3. **Web Layer (`internal/web`)**
   - `Server` holds a `*service.Service` and keeps HTTP concerns only.
   - Template caches, filter option caches, and suggestion caches are provided by service/store and no longer re-computed in handlers.
   - Handlers split by concern (`home.go`, `artists.go`, `locations.go`, `search.go`) to keep files under ~200 LOC and follow KISS.

## Planned Refactor Steps

### 1. Data Loading & Store Simplification
- Move `internal/domain` to `internal/data`; delete `repository.go` and `service.go`.
- Fold `Store.loadData`, `processArtists`, `buildLocations`, `computeStats`, `cacheImages` into cohesive private helpers under the new package.
- Replace manual channels with a `sync.WaitGroup`/`sync.Mutex` pattern:
  - Fetch artists and relations concurrently (keep existing pattern).
  - Kick off goroutines to build indexes, locations, stats, filter/suggestion caches once raw artists are available. Use `WaitGroup` to block `Load` until all derivations finish.
- Return immutable slices (`[]Artist`, `[]Location`) that share backing arrays with the store but document them as read-only. Where mutation is unavoidable (tests), provide dedicated builders instead of exported setters.
- Remove `SetTestData`; introduce `data.NewStoreFromFixtures(artists, locations)` compiled under `_test.go` for unit tests.

### 2. Derived Data & Caching Inside the Store
- Move search suggestion generation and filter option computation from the web layer into `Store` during load. Store the results as slices/maps for O(1) access.
- Replace the ad-hoc search result cache with a generic, lock-protected ring buffer inside service (bounded map + eviction). This keeps cache logic close to the feature but away from handlers.
- Calculate adjacency (prev/next artist) once by storing artists sorted by name and building an index map (`map[int]int`), enabling O(1) navigation without re-scanning.

### 3. Filtering & Search Streamlining
- Relocate `filtering.go` and `search.go` logic into the new service package; eliminate duplicated helper functions by:
  - Sharing normalization utilities via an unexported `strings.go` file.
  - Reusing pre-computed country/album year metadata produced by the store to avoid re-parsing on every request.
- Shorten filter matching by precomputing lookup tables (e.g., `artist.AlbumYear`, `artist.MemberCount`) during load to reduce branching inside request-time loops.
- Introduce small goroutines to parallelize expensive filter operations when inputs are large (e.g., split artist slice into N chunks for filtering/search using a worker pool). Use buffered channels and a `WaitGroup`, ensuring we bail out early when parameters are trivial (keeps KISS while still boosting worst-case performance).

### 4. Web Layer Cleanup
- Update `Server` to receive `(store *data.Store, svc *service.Service)` from a new `app.Initialize()` helper that wires dependencies, calls `store.Load(ctx)` once, and logs startup metrics.
- Remove path validation duplication by extracting helpers into `router.go` and reusing method guards.
- Simplify error handling: centralize template missing/error logging inside `render` and ensure 500 responses use the error template buffer.
- Ensure handlers reject paths like `/artists/foo/bar` by checking `strings.Count` or using a router-level constraint before lookup.
- Keep templates untouched but adapt data structs to match new service outputs (e.g., `Filters: svc.ArtistFilters()` returning pre-built DTOs).

### 5. Testing & Tooling
- Rewrite domain tests to target `data.Store` directly; provide fixture builders and table-driven cases for filtering/search that cover edge cases with the new precomputed fields.
- Update web handler tests to use the new service/store wiring via a thin `testApp()` helper.
- Refresh E2E tests so they assert on simplified responses and 404 handling for malformed artist URLs.
- Run `go test ./...` and capture updated coverage into `tests/final_coverage.html` (replace previous artifact).

### 6. Documentation & Operations
- Document the new architecture in `README.md` and `doc/github-copilot-refactoring-plan-v4.md`, highlighting the two-layer data/service approach and concurrency model.
- Update startup logs: replace `ListenAndServe` wrapper with direct server start preceded by a `logStartupInfo(store, port)` helper that prints cached image counts and clickable URL.
- Remove obsolete files (`internal/domain/repository.go`, `internal/domain/service.go`, legacy docs mentioning compatibility mode`).

## Phased Execution
1. **Phase 0 – Prep**: Rename package directories, adjust module imports, and ensure tests compile with the new package paths (temporarily re-export old names to avoid large diffs).
2. **Phase 1 – Store Rewrite**: Implement new `data.Store`, concurrency for derivations, and fixture helpers; update unit tests to target the new API.
3. **Phase 2 – Service & Filtering**: Create lean service layer, migrate filtering/search logic, and add parallel filtering for large datasets.
4. **Phase 3 – Web Layer Update**: Swap in the new service/store, prune caches from handlers, and split handler files by concern.
5. **Phase 4 – Cleanup & Docs**: Remove compatibility remnants, regenerate coverage artifacts, and refresh documentation.

Each phase should end with `go test ./...` to keep the refactor safe and incremental.
