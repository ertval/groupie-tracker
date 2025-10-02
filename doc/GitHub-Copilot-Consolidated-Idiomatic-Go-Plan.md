# GitHub Copilot Consolidated Idiomatic Go Simplification Plan

## Guiding Principles
- Favor KISS and idiomatic Go: lean structs, zero-value defaults, and straightforward logic.
- Maintain a single source of truth; compute derived values via helpers.
- Default to sequential execution; introduce concurrency only when measurement proves it worthwhile.
- Normalize and clean data during the load phase so request handlers remain thin.
- Comment intent, not mechanics; keep naming concise and consistent.

## Phase 1 – Data Model & Loading Overhaul
- **Artist struct cleanup**
  - Remove cached fields such as `ConcertCount`, `MemberCount`, `Countries`, `FirstAlbumYear`, and `DatesAtLocation`.
  - Store concerts in a single `[]Concert` slice and expose helpers like `MemberCount()`, `ConcertCount()`, `FirstAlbumYear()`, `Countries()`, and `Slug()`.
  - Parse concert dates once into `time.Time` and normalize location slugs with typed wrappers.
- **Location streamlining**
  - Drop the `ArtistAtLocation` wrapper alongside cached aggregates (`ArtistCount`, `TotalConcerts`, `EarliestYear`, `LatestYear`).
  - Maintain a `[]Concert` slice and provide methods such as `ArtistCount()`, `TotalConcerts()`, `YearRange()`, `Slug()`, and `Country()`.
- **Store refactor**
  - Introduce a lightweight catalog component that owns normalized slices/maps while `Store` orchestrates filters and search.
  - Collapse persistent map indexes; prefer slice scans or build-on-demand maps.
- **Loading pipeline**
  - Restructure `loadData` into sequential stages: fetch → normalize → build catalog → build views.
  - Remove goroutine fan-out and worker pools; keep optional image caching behind a narrow interface.
  - Precompile regular expressions or similar helpers once with `sync.Once` or package-level variables.
- **Type hygiene**
  - Keep helper utilities (slug, country extraction) unexported and colocated with use.
  - Ensure structs are zero-value friendly and follow idiomatic Go naming.

## Phase 2 – Filters & Options Simplification
- **Functional filtering approach**
  - Implement predicate-based helpers (`type FilterFunc func(*Artist) bool`) and equivalents for locations.
  - Provide reusable filter builders (`CreationYearBetween`, `HasMemberCount`, `InCountries`, etc.) that compose with AND semantics.
- **Range and set helpers**
  - Introduce `RangeFilter[T]`/`IntRange` types plus reusable string/int set utilities to avoid per-request allocations.
- **Filter structs and match methods**
  - Retain optional explicit `ArtistFilters`/`LocationFilters` structs exposing `Match()` while internally relying on predicates.
- **Filter options metadata**
  - Regenerate filter option data (min/max years, member counts, countries) during catalog build using normalized inputs.
- **Parsing simplification**
  - Replace pointer-heavy parsing with direct construction of filter predicates and eliminate duplicate parsers where possible.

## Phase 3 – Search & Suggestion Refactor
- **Search flow**
  - Remove bespoke LRU caching, mutexes, and order bookkeeping.
  - Build a normalized token index per artist/location during load and scan it directly in `Store.Search`.
  - Consolidate matching logic into a single helper (e.g., `contains(*Artist, query string)`).
- **Suggestions**
  - Deprecate the heavy suggestion infrastructure in favor of either:
    - A minimal JSON search index (names, members, countries, locations) served to the client for in-browser filtering, or
    - A slim server-side list with normalized tokens and unified relevance sorting.
  - Centralize suggestion text/URL formatting helpers.
- **API cleanup**
  - Remove legacy types such as `SearchParams` and `SearchResult` once slices plus predicate-based APIs suffice.

## Phase 4 – Web Layer & Template Simplification
- **Shared view models**
  - Create reusable page structs (e.g., `Page{Title, Description, Data, Assets}`) or a `view` package to eliminate repetitive anonymous structs.
  - Centralize retrieval of common metrics and suggestions.
- **Handler slimming**
  - Push business logic (sorting, filtering) into the data layer.
  - Add reusable request helpers (`requireMethod`, `parseFilters`, `respondError`) for consistent handler flow.
  - Clarify middleware ordering and reuse a method-restriction middleware that sets `Allow` automatically.
- **Template utilities**
  - Compile templates once during application setup with concise logging.
  - Keep the render helper responsible for headers and error handling in a single path.

## Phase 5 – Code Polish & Documentation
- Remove redundant comments; retain only rationale-level notes.
- Normalize naming conventions (drop `Get` prefixes, prefer `ByID`, `BySlug`, `Adjacent`, `Search`).
- Keep helper packages narrowly scoped with clear responsibilities.
- Update README/license notes if workflow or search UX changes.
- Summarize key refactors within `doc/` for future reference.

## Phase 6 – Testing & Validation
- **Unit tests**
  - Update data tests to exercise new helper methods (`artist.ConcertCount()` etc.).
  - Add coverage for filter predicates, `Match` logic, token indexing, and range/set helpers.
- **Integration and E2E**
  - Refresh handler tests to assert new view model shapes and simplified flows.
  - Cover edge cases: artists without concerts, empty filters, unknown slugs, minimal dataset.
- **Quality gates**
  - Run `gofmt`, `go vet`, and `go test` continuously.
  - Benchmark load/search briefly to confirm sub-millisecond results and document findings.

## Rollout Strategy
1. Land data model simplifications and adjust consumers/tests.
2. Introduce the new filter framework and regenerate option metadata.
3. Replace search and suggestion layers once filters stabilize.
4. Simplify handlers/templates and remove residual comments or naming inconsistencies.
5. Execute full test suite and smoke-test via `go run ./cmd/server/` before release.

## Adjacent Improvements
- Add a small CLI to dump the normalized catalog for debugging if helpful.
- Gate new search/suggestion behavior with environment flags during rollout.
- Maintain ongoing refactoring notes in `doc/` to align with existing strategy documentation.
