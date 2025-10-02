# GitHub Copilot Simplification Blueprint

## Goals
- Cut duplication in domain structs first, then flow improvements outward through the store and HTTP layers.
- Retain the current server behaviour and test surface while shaving unnecessary LOC and allocations.
- Prefer standard-library concurrency primitives only, keeping the design understandable for future contributors.

## 1. Core Domain Data (internal/data/models.go, store.go)
1. Replace heavy value copies of `data.Artist` with pointer-backed collections:
   - Change `Store.artists` to `[]*Artist` and update helper indexes to hold pointers to avoid copying large structs on every lookup.
   - Drop redundant getters (`GetArtistFilterOptions`, `GetLocationFilterOptions`) once call sites switch to the pointer collections.
2. Introduce lean reference types for fan-out data:
   - Add `type ArtistSummary struct { ID int; Name, Slug, Image string; MemberCount, ConcertCount int }` stored on each `Artist` and reuse in places where we only need display data.
   - Refactor `ArtistAtLocation` to wrap `*ArtistSummary` instead of the entire `Artist`; pre-size the slice to reduce allocations during `createLocations`.
3. Collapse scattered Store fields into grouped structs to clarify responsibilities and shrink struct definitions:
   - Create `type artistIndex struct { list []*Artist; byID map[int]*Artist; bySlug map[string]*Artist; positions map[int]int }` and similar `locationIndex`.
   - Embed these in `Store` so exported helpers delegate to the index structs, reducing repeated field handling code.
4. Trim rarely used payloads from `Artist`:
   - Replace `DatesAtLocation map[string][]string` with a compact `map[string]ConcertSchedule` containing pre-sorted dates and country metadata.
   - Keep `Concerts` but ensure concert country normalization happens once during load (store on each `Concert`).

## 2. Loading Pipeline & Concurrency (internal/data/store.go, cache.go)
1. Wrap API fetches in a tiny helper to remove duplicated channel struct literals and return early on error.
2. After raw data arrives, run artist enrichment through a bounded worker pool (`runtime.NumCPU()` workers, jobs channel of artists) so `transformAPIArtists` and `addConcertData` execute together per artist without extra passes.
3. Consolidate Stage 4 computations into a single goroutine pool that feeds a results struct; reuse one `sync.WaitGroup` but collapse seven shared variables into the new index structs, reducing boilerplate assignments.
4. While building locations, record stats (members, concerts, countries) incrementally so `calculateStats` no longer re-walks the artist slice.
5. Keep the adaptive image caching worker but move file-path preparation into a `prepareImageJob` helper and expose a `type imageCache struct` to encapsulate mutex + atomic logic.
6. Replace the hand-rolled search cache maps with a dedicated `searchCache` type (`get`, `set`, `touch`, `evict`) that hides locking and order-slice juggling.

## 3. Filtering & Search (internal/data/filters.go, searches.go)
1. Precompute and store a `countriesSet map[string]struct{}` on each `Artist` during load so `matchesArtistFilters` just checks membership without rebuilding maps per request.
2. Hoist repeated `len(slice) == 0` checks into early returns and pre-size the result slice with `make` (capacity = len input) so the loops stay lean.
3. Split `matchesArtistFilters` into small helpers (`matchYearRange`, `matchCountries`) to remove nested condition blocks and reuse from search filtering.
4. Normalize suggestion generation by pushing the " - type" formatting into templates; store raw suggestion text and description separately to cut string concatenation LOC in `generateSearchSuggestions`.
5. In `SearchArtists`, pivot from linear scanning when the query is empty to returning the pre-indexed slice directly, and for non-empty queries reuse a shared lowercase buffer to avoid allocating per comparison.

## 4. Web Layer Streamlining (internal/web/*.go)
1. Cache search suggestions and filter metadata on the `App` struct at startup (`app.suggestions`, `app.artistFilters`, `app.locationFilters`) so handlers stop re-fetching from the store on every request.
2. Introduce a `type Page struct { Title, ExtraCSS string; Suggestions []data.SearchSuggestion }` and embed it in typed view-model structs (`ArtistsPage`, `LocationsPage`, etc.) to eliminate repeated anonymous structs and field lists.
3. Replace the custom `StaticFiles` handler with `http.FileServer(http.Dir("static"))` wrapped by `http.StripPrefix`, keeping only the favicon special-case; this removes the manual filesystem checks.
4. Have `restrictMethod` accept `http.Handler` so we can wrap `http.FileServer` directly and reduce closure boilerplate.
5. Move developer-only handlers into their own file (e.g., `internal/web/dev_handlers.go`) generated from small helper functions to shrink the main `handlers.go` and make production routes easier to scan.
6. Extract form parsing into reusable helpers that return parameter objects and error codes, allowing handler bodies to stay under ~30 LOC each.

## 5. Configuration & Startup (cmd/server/main.go, internal/conf/conf.go)
1. Replace mutable package-level vars with a `conf.Config` struct produced by `conf.Load()` (reads env, applies defaults). Pass the struct into `web.NewApp` to make dependencies explicit and simplify test overrides.
2. Provide a single `App.Start(ctx context.Context)` method that honours context cancellation for graceful shutdown in tests and production launches, removing the need for callers to manually compare against `http.ErrServerClosed`.

## 6. Testing & Guardrails
1. Update store-related tests (`internal/data/data_test.go`, `internal/data/filters_test.go`, search tests) to work with pointer slices and new helper types; add benchmarks for filter/search under the new worker pipeline.
2. Extend web handler tests to verify the shared page structs populate required template fields and that static assets are served by the new file server.
3. Add targeted unit tests for the new `searchCache` and `imageCache` helpers to lock in concurrency behaviour.

## 7. Execution Order
1. Implement domain model restructuring plus new index structs (Section 1).
2. Refactor loading pipeline and caches (Section 2).
3. Adjust filtering/search logic (Section 3) and update associated tests.
4. Apply web layer changes (Section 4) alongside configuration tweaks (Section 5).
5. Run full `go test ./...` and a quick manual smoke (home, artists, locations) before shipping.
