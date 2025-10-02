# GitHub Copilot Restructuring Plan 2025

## Goals
- Trim production LOC by ~12–15% without sacrificing clarity or behaviour.
- Keep the architecture strictly three layers while tightening responsibilities inside each package.
- Prefer straightforward, idiomatic Go; avoid clever abstractions that increase mental load.
- Add concurrency only where it demonstrably reduces wall-clock work (startup and heavy searches) while keeping code approachable.
- Retire compatibility helpers and duplicated APIs that linger from previous refactors.

## Current Pain Points
- `internal/data.Store` mixes structural data, indexes, caching, and helpers in large functions that repeat the same loops (concert aggregation, country extraction, filter prep).
- The domain structs carry redundant views of the same data (for example `Members` + `MemberCount`, `Concerts` + `DatesAtLocation`), forcing extra bookkeeping every time we transform artists.
- Filter/search pipelines re-parse parameters on every call and linearly scan the full dataset even when we only need a subset.
- Handlers build similar view models and repeatedly fetch global data (suggestions, filter options) instead of using a common page context.
- Compatibility helpers (`GetArtistFilterOptions`, `GenerateAllSearchSuggestions`) duplicate newer APIs and can be removed once callers switch.

## Phase 1 – Core Domain Model Simplification
1. **Normalise concert data once.**
   - Replace `Artist.DatesAtLocation map[string][]string` plus `Concerts []Concert` with a single `map[string]ConcertLedger` where `ConcertLedger` holds sorted dates, total count, and derived country. This keeps all per-location data together and eliminates the extra `Countries []string` + manual loops.
   - Compute canonical `ConcertWindow` (earliest/latest year) for both artists and locations during build time and store it in a lightweight struct instead of scattering `EarliestYear`/`LatestYear` ints.
2. **Minimise duplicated scalar fields.**
   - Drop `MemberCount` and derive it lazily via `len(Members)` where needed; the dataset is small, and we can micro-optimise by memoising inside the new ledger when required.
   - Remove `ConcertCount` from `Artist` once the ledger exists; expose an accessor `func (Artist) ConcertCount() int` so templates remain simple without duplicating state.
3. **Tighten type safety.**
   - Introduce `type ArtistID int` and `type LocationSlug string` to make map keys explicit, reducing accidental string/int mismatches during refactors.
   - Move helper functions (`createSlug`, `extractCountryFromLocation`, `extractYearFromDate`) into a small `normalise.go` to keep the primary store file focused on orchestration.

Expected LOC delta: **≈ -180** (removing duplicate fields, redundant loops, and helper code from handlers/templates).

## Phase 2 – Store Loading & Indexing Pipeline
1. **Introduce build stages.**
   - Factor the current `loadData` into `fetchRaw(ctx)`, `buildDataset(rawArtists, rawRelations)`, and `finaliseIndexes(dataset)` functions for clarity and testability.
   - Each stage should return immutable structs (`dataset`, `indexes`) to keep mutation local and enable targeted tests.
2. **Concurrent artist enrichment.**
   - After fetching raw data, spin up a bounded worker pool (size `min(runtime.NumCPU(), len(artists))`) to run `hydrateArtist` in parallel. Each worker will:
     - Build the concert ledger from relations.
     - Populate derived stats (first album year, concert window).
   - Use `sync.Pool` for temporary buffers (date slices) to keep allocations low but keep the worker logic straightforward.
3. **Shared parallel helper.**
   - Extract the repeated `wg.Add/Done` pattern used for building indexes into a reusable `parallel(tasks ...func())` helper inside `internal/data` so Stage 4 reads clearer and removes ~25 duplicated lines.
4. **Kill compatibility APIs.**
   - Remove `GetArtistFilterOptions`, `GetLocationFilterOptions`, and `GenerateAllSearchSuggestions` in favour of `ArtistFilterOptions`, `LocationFilterOptions`, and `Suggestions` (update callers in `handlers.go`).
   - Collapse the `searchCache` accessors into a single `cache.Search` helper to shrink the cache code by ~40 LOC.

Expected LOC delta: **≈ -220** (mainly from deduplicated pipelines and cache helpers).

## Phase 3 – Filtering & Search Optimisation
1. **Compile filters.**
   - Replace `matchesArtistFilters` with a builder that returns a predicate `type ArtistPredicate func(*Artist) bool`. The predicate should close over precomputed sets (countries, member counts) and range bounds, eliminating repeated map allocations per artist.
   - Reuse the same predicate in `FilterArtists` and `SearchArtists`, removing the redundant second pass when combining search + filters.
2. **Introduce lightweight indexes.**
   - Precompute `artistByCountry map[string][]*Artist` and `artistByMemberCount map[int][]*Artist` during startup for fast lookups when only a subset is needed. Use intersection logic to keep it simple: start from the smallest index slice to minimise scans.
   - Add a tiny cache for range-only searches (e.g., creation year) by storing sorted slices and using binary search instead of full scans.
3. **Streamlined suggestion filtering.**
   - Store suggestions as `[]*SearchSuggestion` with the `normalizedText` prepared upfront. Replace the triple-slice accumulation in `filterSearchSuggestions` with a stable in-place partition (single pass) to cut the function roughly in half.
4. **Optional concurrency for heavy searches.**
   - For wide-open requests (empty query + loose filters), run the predicate application using two goroutines over halves of the dataset and merge results. Keep the implementation behind a `if len(artists) > 40` guard to avoid overhead for tiny slices.

Expected LOC delta: **≈ -160** (less branching logic, shared predicates, smaller suggestion filter).

## Phase 4 – Web Layer Simplification
1. **Shared page context.**
   - Introduce `type PageContext struct { Title string; ExtraCSS string; Suggestions []data.SearchSuggestion; ... }` plus helper constructors (`NewHomeContext`, `NewArtistListContext`, etc.). Let handlers focus on pulling data from the store and delegating to the constructor.
   - Move repeated `suggestions := app.store.Suggestions()` and `filterOptions := ...` blocks into those constructors.
2. **Form parsing helpers.**
   - Replace the bespoke `parseArtistFilterParams`/`parseLocationFilterParams` with a generic `extractRange` + `extractSet` utility. This reduces the two functions to declarative lists and trims ~40 LOC.
3. **Static assets handler.**
   - Swap the manual path validation in `StaticFiles` for `http.StripPrefix` + `http.FileServer`, then wrap it with a custom `denyDotfiles` middleware to keep security tight but reduce LOC by ~30.
4. **Retire redundant JSON endpoints.**
   - `SuggestionsAPI` can simply marshal through `json.Marshal` and write once; reuse a shared `writeJSON(w, status, payload)` helper used elsewhere to centralise error handling.

Expected LOC delta: **≈ -140**.

## Phase 5 – Tests, Benchmarks, and Tooling
1. Update fixtures to the new domain structs and add unit tests for `ConcertLedger` and compiled predicates.
2. Add benchmark tests for `SearchArtists` with/without concurrency to ensure the new path earns its keep.
3. Simplify existing table-driven tests by introducing helper assertions (`assertArtistNames`, `assertLocationSlugs`) to reduce boilerplate.
4. Expand E2E tests to cover the leaner static asset serving and ensure headers stay correct.

Expected LOC delta: **≈ -60** (tests tighten up despite some new coverage).

## Rollout Checklist
- Implement domain model changes first and update fixtures/templates before touching handlers.
- Convert handlers to the new context helpers and delete compatibility functions in the same commit to avoid dangling callers.
- Regenerate coverage report (`go test -cover ./...`) after each major phase to ensure parity.
- Update `doc/REFACTORING_SUMMARY_OCT_2025.md` once the plan is executed to keep historical tracking accurate.

## Risks & Mitigations
- **Template impact**: Changing artist/location field names can break templates. Mitigate by providing accessors (`func (Artist) ConcertCount() int`) and unit tests with `html/template` execution.
- **Concurrency complexity**: Keep worker pools bounded and encapsulated (`hydrateArtists` returns a slice and error). Document the reasoning inline to preserve KISS.
- **Index maintenance**: Ensure indexes rebuild during `Store.Load` only; mark slices/maps as immutable and expose read-only getters to prevent accidental modification.

Following this roadmap should shave **~760 LOC** overall, keep the code idiomatic, and position the project for future features without reintroducing unnecessary layers.
