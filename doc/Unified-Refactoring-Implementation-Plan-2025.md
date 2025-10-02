# Unified Refactoring & Simplification Implementation Plan (October 2025)

## Overview
- **Mission**: Reduce production LOC by ~12–15% while preserving behaviour, clarity, and the three-layer architecture (API → data store → web).
- **Constraints**: Standard library only, zero JavaScript, immutable store after load, maintain ≥70% test coverage, avoid reintroducing service layers.
- **Performance Targets**: ~10–15% faster startup, ≥20% faster filtering, no material regression elsewhere.
- **Operating Model**: All steps executed by a single automated agent; keep commits logically grouped per phase. Run `go test ./...` after each major milestone.

## Success Criteria
- LOC reduction ≥12% without sacrificing readability.
- All tests (unit + E2E) pass with `go test ./...` on Go 1.24.3.
- Search, filter, and page rendering behave identically to pre-refactor baselines.
- Image cache, search cache, and adaptive concurrency remain standard-library implementations.

## Phase 0 — Baseline & Tooling Prep
1. Capture `main` branch baseline: LOC (`cloc` optional), `go test ./...`, load manual smoke (home, artists, locations).
2. Read existing docs in `doc/` (especially refactoring summaries) to confirm historical decisions.
3. Draft branch naming convention per phase (e.g., `refactor/domain-ledger`).
4. Ensure environment uses Windows + bash-friendly commands as needed.

## Phase 1 — Domain Model Simplification
**Objectives**: Flatten duplicate fields, adopt pointer-backed collections, introduce concert ledgers, and normalise stats.

1. Update `internal/data/models.go`:
   - Replace pointer-based filter params with zero-value semantics (e.g., `CreationYearMin int` where `0` means "unset").
   - Flatten `AppStats` into a single struct (remove type alias).
   - Introduce `type ArtistID int`, `type LocationSlug string`, and `ConcertLedger` for per-location data.
   - Remove redundant scalars (`MemberCount`, `ConcertCount`) in favour of accessors (e.g., `func (a *Artist) ConcertCount() int`).
2. Switch store collections to pointer slices (`[]*Artist`, `[]*Location`) and group indexes into dedicated structs (`artistIndex`, `locationIndex`).
3. Replace `DatesAtLocation` with `map[LocationSlug]ConcertLedger`; embed pre-sorted dates and derived country.
4. Precompute and store per-artist country sets for fast filter checks.
5. Delete duplicate accessors (`GetArtistFilterOptions`, etc.). Update all call sites immediately, including templates, handlers, and tests.
6. Maintain JSON/API compatibility by updating marshal tags if required; run targeted tests (`internal/data/data_test.go`).

## Phase 2 — Data Loading & Caching Pipeline
**Objectives**: Clarify load stages, boost concurrency (standard library only), encapsulate caches.

1. Refactor `Store.Load` into three explicit stages:
   - `fetchRaw(ctx)` — parallel API fetch for artists and relations with channels.
   - `buildDataset(rawArtists, rawRelations)` — hydrate artists, build concert ledgers, populate stats.
   - `finaliseIndexes(dataset)` — create indexes, suggestions, filter metadata.
2. Introduce bounded worker pool (`min(runtime.NumCPU(), len(artists))`) to hydrate artists concurrently; reuse `sync.WaitGroup` and channels only.
3. Encapsulate image caching logic in a dedicated `imageCache` type. Keep adaptive worker pool (no third-party packages).
4. Replace manual search cache maps with a `searchCache` struct exposing `get`/`set`/`evict` while retaining LRU behaviour (50-entry limit).
5. Move helper functions (`createSlug`, `extractCountryFromLocation`, `extractYearFromDate`) into a focused `normalise.go` to declutter `store.go`.
6. Ensure all maps/slices become immutable post-`Load()`; document via comments where necessary.
7. Re-run data-layer tests and add new unit tests for `ConcertLedger`, `searchCache`, and `imageCache` helpers.

## Phase 3 — Filtering & Search Refinement
**Objectives**: Remove duplication, favour compiled predicates, and integrate caching with zero-value filters.

1. Introduce `type ArtistPredicate func(*Artist) bool`; create a builder that consumes `ArtistFilterParams` and precomputes lookup sets (member counts, countries).
2. Refactor `FilterArtists` and `SearchArtists` to reuse the predicate, avoiding double passes when combining search + filters.
3. Implement helper utilities (`matchRange`, `containsInt`, `hasIntersection`) once and share across artist/location filters.
4. Optimise empty-query searches: return the base artist list directly when no filters apply.
5. Add lightweight indexes (`artistByCountry`, `artistByMemberCount`) built at startup; apply only when corresponding filters are present (start from smallest slice for intersection).
6. Streamline suggestion generation: store raw text + metadata, push formatting to templates, and ensure `normalizedText` is cached for comparisons.
7. Maintain LRU search cache update logic (`updateCache` helper) and ensure zero-value filter semantics are honoured.
8. Extend `internal/data/filters_test.go` and `searches.go` tests to cover edge cases (empty filters, overlapping sets, wide-open searches).

## Phase 4 — Web Layer Consolidation
**Objectives**: Collapse `App` → `Server`, simplify routing, share page contexts, and standardise form parsing.

1. Rename/replace `App` with a single `Server` struct in `internal/web/server.go`; constructor `NewServer(apiClient, withCache)` loads the store, compiles templates, and constructs the `http.Server`.
2. Adjust `cmd/server/main.go` to use the new `Server` API; ensure tests in `internal/web/web_test.go` adapt to the new struct.
3. Introduce routing helpers inside `routes.go` (`get`, `post`, `getPost`) wrapping `restrictMethod` to shrink boilerplate.
4. Cache commonly reused data (`Suggestions`, `FilterOptions`) on the server instance during startup.
5. Create a shared `PageContext` struct plus specific view models (`ArtistsPage`, `LocationsPage`, etc.) that embed it, replacing ad-hoc maps/structs in handlers.
6. Migrate developer-only endpoints into `dev_handlers.go` (optional but recommended) to declutter core handlers.
7. Replace manual static file handling with `http.StripPrefix` + `http.FileServer`, wrapped by a `denyDotfiles` middleware to retain safety.

## Phase 5 — Form & Template Utilities
**Objectives**: Centralise form parsing, align templates, and keep zero-JS interactions.

1. Add a reusable form parser in `internal/web/templates.go` (e.g., `type formParser struct {...}`) with helpers `int`, `intSlice`, `stringSlice` supporting zero-value filters.
2. Update `parseArtistFilterParams`/`parseLocationFilterParams` to use the parser and the new struct fields (`CreationYearMin`, etc.).
3. Review templates (`templates/*.tmpl`) and adjust bindings to new accessors and contexts (e.g., `{{.Artist.ConcertCount}}` using methods instead of fields).
4. Confirm no JavaScript dependencies are introduced; all interactions remain POST forms.

## Phase 6 — Configuration & Startup Polish
**Objectives**: Make configuration explicit and support graceful shutdown.

1. Refactor `internal/conf` to expose a `Config` struct returned by `Load()` (reads env, applies defaults).
2. Update `cmd/server/main.go` to obtain config, instantiate API client, and call `server.ListenAndServe()`; handle shutdown signals using context.
3. Maintain default values (`config.DefaultPort`, cache flag) and ensure tests can override via helper constructors.

## Phase 7 — Testing, Benchmarks, and Quality Gates
1. Update fixtures in `internal/data/fixtures.go` to match new domain structs.
2. Expand data-layer tests for `ConcertLedger`, predicate builder, search cache, and pointer-based slices.
3. Add benchmarks for `FilterArtists` and `SearchArtists` (with and without filters) to validate performance gains.
4. Run E2E tests under `cmd/server` confirming primary pages, search, filters, and static assets.
5. Regenerate coverage reports (`go test -cover ./...`) and update summaries in `tests/` if part of workflow.
6. Document improvements in `doc/REFACTORING_SUMMARY_OCT_2025.md` once plan is executed.

## Rollout & Risk Management
- **Incremental Delivery**: Ship each phase in its own PR; include test evidence and before/after metrics.
- **Risk Mitigations**:
  - Domain changes risk template regressions → add snapshot tests via `html/template` execution.
  - Concurrency adjustments risk races → run `go test -race ./internal/data`.
  - Cache refactors risk stale data → include targeted tests ensuring eviction and hit logic.
- **Fallback Strategy**: Maintain feature branches per phase; revert quickly if regressions appear.

## Quick Reference Checklist (per Phase)
- [ ] Update code & tests according to phase goals.
- [ ] Run `go test ./...` (and `-race` for data layer changes).
- [ ] Perform manual smoke via local server as needed.
- [ ] Capture LOC delta (`git diff --stat`).
- [ ] Update documentation or logs of changes.

## Post-Implementation
- Compile performance findings into `doc/REFACTORING_ANALYSIS_SUMMARY.md`.
- Share LOC and benchmark improvements with stakeholders.
- Identify leftover nice-to-haves (e.g., optional concurrent search path behind guard) as future backlog items.
