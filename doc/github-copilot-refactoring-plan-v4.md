# GitHub Copilot Refactoring & Optimization Plan (v4)

## 1. Objectives
Simplify and optimize the codebase while preserving all current end-user functionality (artists, locations, search, filtering, detail pages, stats, caching, error handling) using idiomatic Go and KISS principles.

Key goals:
- Reduce cognitive load & LOC (remove duplication / unnecessary helper layering)
- Keep clear separation of concerns (API fetch → data load/transform → read‑only queries → HTTP layer)
- Introduce bounded, low-risk concurrency where it measurably helps (parallel API fetch & image caching)
- Preserve deterministic behavior, testability, and current templates (no JS additions)
- Avoid premature micro-packaging; strike balance between modularity and simplicity

## 2. Current State Summary
Layered architecture (api → domain → web) is already reasonably clean. Main complexity hotspots:
- `repository.go` mixes: orchestration, transformation, enrichment, image caching, slug/location normalization, stats aggregation.
- Filtering & search logic overlaps in normalization and country/year extraction utilities.
- Handlers include small validation helpers repeated (path validation, filter emptiness) and inline data shaping logic.
- Some functions operate sequentially but could safely parallelize (API fetches, image downloads) with simple fan-out control.
- Tests rely on repository being deterministic: we must retain single-pass load & immutable slices/maps after init.

## 3. Guiding Principles
1. Single initialization pipeline → immutable data structures (no post-load mutation except via clearly marked test hooks).
2. Keep public surface small: expose only getters + purposeful query helpers (Search, Filters, Adjacent lookups).
3. Utilities that are transformation-only become unexported pure helpers (facilitates unit testing via table tests still).
4. Concurrency only in startup path & image caching; no per-request goroutines unless they eliminate clear latency (not needed now since all data is in-memory reads ~O(1)).
5. No generic abstractions unless needed by two or more call sites.

## 4. Proposed Structural Adjustments
| Area | Change | Rationale |
|------|--------|-----------|
| domain/repository.go | Split into focused files: `load.go` (LoadData + concurrency), `transform.go` (artist/location building), `cache.go` (image caching), `stats.go` (AppStats aggregation), keep `repository.go` (struct + constructors + getters) | Improves readability without over-fragmenting |
| domain/filtering.go & search.go | Introduce `query.go` consolidating shared normalization/util funcs (country extraction, year parsing, normalize search) | Removes duplication, centralizes parsing invariants |
| domain/models.go | Leave as-is (already cohesive) | Simplicity |
| web/handlers.go | Extract small reusable helpers (path validation already exists; keep). Optional: move cache-related Search logic into repository-level `SearchWithCache` if we want to centralize (but might keep to avoid expanding repo responsibilities). | Avoid overfat handlers, keep repo single-responsibility |
| api package | Already lean; keep intact. | Adequate |
| doc folder | Add v4 plan (this doc). Remove obsolete older phase plans if superseded (retain one historical summary file). | Declutter |

## 5. Concurrency Plan
Phase into Load pipeline only (repository init):
1. Parallel API fetches: fetch artists & relations concurrently with `errgroup.Group` (bounded: 2 goroutines).
2. Parallel image caching: after artists transformed & concerts enriched, spawn worker pool (e.g., size = min(8, runtime.NumCPU()*2)) to download uncached images. Collect results via channel; update counters atomically (or aggregate per worker then reduce). Mutate only the image field on the slice (safe because exclusive during init).
3. Keep deterministic ordering: after all downloads, final sort/reslug unaffected.
4. Ensure context cancellation propagates (pass ctx to HTTP GET with `http.NewRequestWithContext`).

No concurrency for per-request reads (maps/slices are immutable). Search/filter already O(n) with small dataset—parallelization would add overhead and complexity.

## 6. Simplifications & Optimizations
- Replace manual loops constructing multiple maps with single pass reducers where possible.
- Hoist regex compilation for slug creation to package level (avoid recompilation).
- Deduplicate `normalizeSearchQuery`, `normalizeLocation`, and country capitalization into shared helpers.
- Collapse `convertCountriesMapToSlice` + direct map iteration into inline local helper where only used once (or keep if reused post-split—decide after split).
- Inline trivial wrappers (e.g., `processArtists` could become sequence in `loadArtists` for clarity) if it reduces cognitive hops.
- Introduce small value objects? Not needed; current structs are fine.

## 7. Public API Surface After Refactor (Repository)
- Constructor: `NewRepository(client *api.Client, withCache bool)`
- Initialization: `Load(ctx context.Context) error` (rename from LoadData)
- Getters: `Artists() []Artist`, `ArtistByID(id int) (Artist,bool)`, `ArtistBySlug(slug string) (Artist,bool)`, `Locations() []Location`, `LocationBySlug(slug string) (Location,bool)`, `Stats() AppStats`, `AdjacentArtists(id int) (prev,next *Artist)`
- Queries: `FilterArtists(params ArtistFilterParams) []Artist`, `FilterLocations(params LocationFilterParams) []Location`, `SearchArtists(params SearchParams) SearchResult`, `GenerateAllSearchSuggestions() []SearchSuggestion`
(All names shortened for readability; handler layer adapts.)

Backward compatibility: handlers updated in one atomic commit—tests adjusted accordingly.

## 8. Migration Steps (Phased Execution)
Phase 1: Mechanical Split
- Copy existing logic into new files (`repository.go`, `load.go`, `transform.go`, `cache.go`, `stats.go`, `query_utils.go`).
- Adjust function receivers & visibility (lowercase for internal helpers).
- Run tests (expect no behavioral change). Commit.

Phase 2: Concurrency Introduction
- Introduce `errgroup` for API fetch.
- Implement image download worker pool (benchmark optional). Ensure deterministic final slice ordering (only mutate `.Image`).
- Add unit tests for image caching (mock HTTP server). Commit.

Phase 3: Helper Deduplication & Naming Cleanup
- Consolidate normalization & parsing functions.
- Rename public methods to concise forms (add deprecated wrappers if necessary for transitional test update; remove after test adjustment). Commit.

Phase 4: Search/Filter Minor Optimizations
- Precompute lowercased artist & member names arrays to slightly accelerate search (optional; micro). Only if profiling shows >5% improvement; otherwise skip to keep simplicity.

Phase 5: Documentation & Cleanup
- Remove superseded plan docs (retain summary file).
- Update `README.md` architecture section reflecting new file breakdown & concurrency notes.
- Add `CONCURRENCY.md` short rationale (optional if README section concise).

## 9. Risk & Mitigation
| Risk | Mitigation |
|------|-----------|
| Data races during concurrent image caching | Perform before exposing repo; slice not shared until repo fields assigned. No goroutines after publish. |
| Increased complexity from over-splitting | Limit to 5 focused files; avoid deeper package nesting. |
| Test fragility due to renamed methods | Batch rename + update tests same commit. |
| Unbounded goroutines on large dataset | Fixed worker pool with queue channel + context cancel. |
| Non-deterministic image assignment order | Only field mutated is `Image`; final sort uses Name; stable. |

## 10. Acceptance Criteria
- All existing functionality & templates operate identically (snapshot tests pass).
- Test suite green with equal or higher coverage (target ≥ previous coverage baseline).
- Repository load time decreased (qualitative log timing; optional micro-bench for fetch+cache path).
- No data races (run with `go test -race ./...`).
- LOC in domain package reduced (excluding added concurrency code overhead) OR cognitive complexity reduced (fewer very large files >500 LOC).

## 11. Deferred / Explicitly Not Doing
- No premature generics or interface abstraction for repositories.
- No per-request goroutine search parallelization (dataset small). Could revisit if dataset grows 10x.
- No caching layer beyond current in-memory structures.

## 12. Quick Pseudocode (Concurrency Sections)
```
// Parallel fetch
var artistsData []api.Artist
var relationsData api.Relation
g, ctx := errgroup.WithContext(ctx)

g.Go(func() error { var err error; artistsData, err = client.FetchArtists(ctx); return err })
g.Go(func() error { var err error; relationsData, err = client.FetchRelations(ctx); return err })
if err := g.Wait(); err != nil { return err }

// Image worker pool
jobs := make(chan *Artist)
wg := sync.WaitGroup{}
for i := 0; i < workers; i++ {
  wg.Add(1)
  go func(){ defer wg.Done(); for a := range jobs { downloadAndSet(a) } }()
}
for i := range artists { jobs <- &artists[i] }
close(jobs); wg.Wait()
```

## 13. Post-Refactor Review Checklist
- grep for old names: `grep -R "GetArtists("` → ensure updated.
- run: `go test -race ./...`
- verify log startup timings & concurrency path.
- manual smoke: home, artists, artist detail, search, locations, location detail, dev endpoints.

## 14. Summary
This plan keeps the existing clean architecture while reducing monolithic file size, consolidating parsing logic, and adding safe, bounded concurrency to the initialization path—achieving a pragmatic balance between simplicity and performance without over-engineering.

---
Generated by GitHub Copilot Refactoring Assistant (v4 plan).
