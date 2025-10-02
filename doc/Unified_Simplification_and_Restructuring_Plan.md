# Unified Simplification & Restructuring Implementation Plan

## Objectives
- Deliver a leaner, pointer-driven domain model that eliminates redundant data copies while keeping all templates and handlers behaviourally identical.
- Streamline the store loading, indexing, caching, filtering, and search pipelines to reduce allocations and improve startup/search responsiveness using only the Go standard library.
- Simplify the HTTP layer, configuration surface, and supporting tests without reintroducing additional services or JavaScript.
- Preserve full test coverage with incremental validation after each phase and ensure the final system passes `go test ./...`.

## Guiding Principles
- Standard library only; avoid introducing third-party dependencies.
- Prefer immutable, pointer-based datasets after `Store.Load()` completes; expose read-only helpers instead of mutating structures.
- Keep concurrency bounded (`runtime.NumCPU()` driven pools) and well-documented; target clarity over cleverness.
- Maintain compatibility for external behaviour (routes, templates, API contracts) while removing deprecated helper functions.
- Validate every structural change with focused unit tests before moving to the next phase.

## Phase 1 – Domain Data & Store Structure
1. **Reshape core collections**
   - Convert `Store.artists` and related indexes to pointer-backed collections (`[]*Artist`, `map[int]*Artist`, `map[string]*Artist`).
   - Introduce `artistIndex` / `locationIndex` structs embedded in `Store` to encapsulate list + lookup maps + positional metadata.
   - Update helper getters and call sites to rely on the new index helpers; delete redundant getter functions once callers are switched.
2. **Introduce lightweight summaries**
   - Add `ArtistSummary` (ID, Name, Slug, Image, MemberCount, ConcertCount) cached on each artist.
   - Update fan-out structs (e.g., `ArtistAtLocation`) to reference `*ArtistSummary`, presizing slices where possible.
3. **Normalize concert data**
   - Replace `DatesAtLocation` with a compact ledger (e.g., `map[string]ConcertSchedule`) storing sorted dates, country metadata, and counts.
   - Ensure concert country normalization and date sorting occur during load; keep accessors simple for templates.
4. **Tighten type safety and helpers**
   - Introduce typed IDs/slugs where helpful (e.g., `type ArtistID int`).
   - Move slug/country/year normalization helpers into a focused utility file to declutter `store.go`.
5. **Acceptance checks**
   - Existing unit tests updated for pointer semantics.
   - Templates render correctly using new summaries/ledgers (add targeted render tests).

## Phase 2 – Loading Pipeline & Caching Infrastructure
1. **Segment load stages**
   - Refactor `Store.loadData` into clear stages: `fetchRaw(ctx)`, `buildDataset(rawArtists, rawRelations)`, `finalizeIndexes(dataset)`.
   - Use a shared result struct to pass computed indexes, summaries, suggestions, and stats out of `finalizeIndexes`.
2. **Concurrent artist enrichment**
   - Implement a bounded worker pool (`min(runtime.NumCPU(), len(artists))`) to run `hydrateArtist` tasks that build ledgers, summaries, and derived stats in one pass.
   - Leverage `sync.WaitGroup` + channels; reuse buffers via `sync.Pool` only if profiling justifies it.
3. **Unified computation pool**
   - Collapse separate goroutines for statistics, filter metadata, and suggestions into a coordinated worker pool fed by jobs; results populate the new index structs.
4. **Image caching encapsulation**
   - Extract an `imageCache` type responsible for job preparation, locking, and eviction.
   - Move path preparation into a helper (e.g., `prepareImageJob`) before dispatching to workers.
5. **Search cache tightening**
   - Replace ad-hoc map/slice juggling with a dedicated `searchCache` type exposing `get`, `set`, `touch`, and `evict` methods with internal locking.
6. **Acceptance checks**
   - Benchmark startup vs. baseline to confirm no regression.
   - Unit tests for `searchCache` and `imageCache` concurrency behaviour.

## Phase 3 – Filtering & Search Optimisation
1. **Precomputed metadata**
   - Store `countriesSet` (and similar quick-look structures) on each artist during load to avoid per-request map allocations.
2. **Predicate-driven filtering**
   - Replace `matchesArtistFilters` with compiled predicates (`type ArtistPredicate func(*Artist) bool`) shared by `FilterArtists` and `SearchArtists`.
   - Decompose range/country/member checks into helpers (`matchYearRange`, `matchCountries`, etc.).
3. **Slice management & fast paths**
   - Hoist empty-filter checks to early returns; pre-size result slices.
   - For empty queries, return pre-indexed slices directly instead of scanning.
4. **Optional parallel scan**
   - For large result sets (e.g., loose filters), split filtering across two goroutines guarded by a heuristics threshold (e.g., dataset size > 40) and merge results deterministically.
5. **Suggestion generation cleanup**
   - Store raw suggestion text + description separately; push formatting (" - type") into templates.
   - Simplify suggestion filtering with in-place partitioning and pointer slices.
6. **Acceptance checks**
   - Update `internal/data` filter and search tests to cover new predicates.
   - Add benchmarks for hot paths (search with/without filters, suggestion filtering).

## Phase 4 – Web Layer Streamlining
1. **Page context consolidation**
   - Introduce `Page` / `PageContext` structs capturing shared fields (Title, ExtraCSS, Suggestions, filters, stats) and embed in specific page view models.
   - Cache global metadata (suggestions, filter options) on the server at startup.
2. **Form parsing helpers**
   - Extract reusable range/set parsing utilities reducing handler boilerplate.
   - Ensure helpers return validated parameter structs with error codes for bad input paths.
3. **Static asset handling**
   - Replace custom static handler with `http.StripPrefix("/static/", http.FileServer(...))`, wrapped by middleware that blocks dotfiles and sets headers.
4. **Handler organisation**
   - Move developer-only routes into `dev_handlers.go` (or similar) keeping production handlers concise.
   - Ensure `restrictMethod` accepts `http.Handler` to wrap both standard handlers and file server directly.
5. **Acceptance checks**
   - Update `internal/web` tests to cover new page structs, static asset delivery, and error handling.
   - Smoke-test critical routes (home, artists, locations, search, dev tools) using `httptest`.

## Phase 5 – Configuration & Startup Simplification
1. **Config struct**
   - Replace global config vars with `conf.Config` returned by `conf.Load()`; read env/defaults into the struct.
2. **Explicit dependencies**
   - Update `cmd/server/main.go` and `web.NewServer` (or new `web.NewApp`) to accept `conf.Config` and injected `*api.Client`.
3. **Graceful lifecycle**
   - Implement `App.Start(ctx context.Context)` (or equivalent) to wrap HTTP server startup/shutdown, honouring context cancellation for tests.
4. **Acceptance checks**
   - Adjust existing startup tests; add coverage for shutdown path.

## Phase 6 – Testing, Benchmarks & Quality Gates
1. **Fixture updates**
   - Refresh fixtures for pointer-based artists, summaries, and new ledgers.
2. **New unit tests**
   - Add suites for `searchCache`, `imageCache`, page context constructors, and compiled predicates.
3. **Benchmarks**
   - Introduce targeted benchmarks for store loading, filtering, and search concurrency paths.
4. **E2E adjustments**
   - Extend E2E tests to validate new static file handling and overall page responses.
5. **Quality gates**
   - After each phase, run `go test ./...` (or narrower scope) and record coverage deltas.
   - Maintain ≥70% coverage on `internal/data`; update coverage reports stored in `tests/` as needed.

## Execution Flow & Checkpoints
1. Complete Phases 1–3 sequentially; they build on the domain model changes.
2. Once data and search layers stabilise, proceed to Phase 4 (web layer) and Phase 5 (configuration) in parallel branches if needed, but merge only after full regression tests.
3. Run comprehensive test suites (unit + E2E) and benchmarks before declaring completion.
4. Document outcomes and LOC deltas in `doc/REFACTORING_SUMMARY_OCT_2025.md`.

## Deliverables & Reporting
- Updated Go source files per phase with inline documentation where concurrency is introduced.
- New/updated tests and benchmarks reflecting the streamlined architecture.
- Revised documentation (architecture summary, refactoring summary) capturing structural changes.
- Final verification log summarising test runs, benchmark results, and manual smoke findings.
