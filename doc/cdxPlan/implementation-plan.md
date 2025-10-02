# Step-by-Step Implementation Plan

This document converts the "GitHub Copilot Consolidated Idiomatic Go Simplification Plan" into execution-ready steps. Each phase lists the exact sequence of actions, suggested owners, and validation gates.

---

## Phase 1 – Data Model & Loading Overhaul

1. **Inventory current data structures**
   - Review `internal/data/models.go`, `store.go`, and related structs for cached fields and redundant indexes.
   - Capture a before-and-after diagram in `doc/` to track the simplification.
2. **Refactor `Artist` model**
   - Remove cached fields (`ConcertCount`, `MemberCount`, etc.) and ensure zero-value safety.
   - Introduce computed helpers: `MemberCount()`, `ConcertCount()`, `FirstAlbumYear()`, `Countries()`, `Slug()`.
   - Update all call sites (filters, handlers, templates, tests) to use the new helpers.
3. **Refactor `Location` model**
   - Drop `ArtistAtLocation` and cached aggregates.
   - Add helper methods (`ArtistCount()`, `TotalConcerts()`, `YearRange()`, `Slug()`, `Country()`) and migrate usages.
4. **Normalize concert data**
   - Parse concert dates into `time.Time` once, keeping raw strings if needed for display.
   - Normalize location slugs and countries via dedicated helper functions.
5. **Introduce the catalog component**
   - Create a lightweight catalog that owns slices/maps; adjust `Store` to orchestrate filters/search using catalog data.
   - Remove persistent map indexes unless profiling proves they are required.
6. **Linearize the loading pipeline**
   - Rewrite `loadData` to follow fetch → normalize → build catalog → derive views.
   - Remove goroutines/worker pools; retain optional image caching behind a thin interface.
7. **Optimize helper initialization**
   - Precompile regexes or similar utilities with `sync.Once` or package-level variables.
8. **Run regression tests and lint**
   - Execute `go test ./...` and `go vet ./...`.
   - Update docs/tests to reflect model changes.

## Phase 2 – Filters & Options Simplification

1. **Design predicate abstractions**
   - Define `type FilterFunc func(*Artist) bool` (and location equivalent) in the data layer.
   - Document expected semantics (AND-composition, no side effects).
2. **Implement reusable filter builders**
   - Create functions like `CreationYearBetween`, `HasMemberCount`, `InCountries`, `PlayedInYear`.
   - Add unit tests covering typical and edge cases (empty data, zero members).
3. **Introduce range and set helpers**
   - Add `RangeFilter[T]` or `IntRange` types plus reusable string/int set utilities.
   - Ensure allocations are minimized; benchmark if necessary.
4. **Refactor filter structs**
   - Keep `ArtistFilters`/`LocationFilters` with `Match()` that internally composes predicate lists.
   - Update parsing code to populate predicates instead of fields directly.
5. **Regenerate filter option metadata**
   - Compute min/max years, member counts, and countries during catalog build.
   - Cache results in the catalog and expose read-only getters.
6. **Simplify parsing flow**
   - Replace pointer-heavy parsing with direct predicate construction.
   - Ensure handlers use shared parsing helpers.
7. **Validate end-to-end**
   - Run focused tests (`internal/data/data_test.go`, `internal/web/web_test.go`).
   - Confirm UI renders expected filter ranges.

## Phase 3 – Search & Suggestion Refactor

1. **Analyze current search stack**
   - Identify legacy types (`SearchParams`, `SearchResult`, caching layers) slated for removal.
   - Document migration strategy for references across server and templates.
2. **Build normalized token index**
   - During load, generate tokens for artist/location names, members, countries, and locations.
   - Store index in catalog for direct scans; eliminate bespoke LRU caching.
3. **Unify search matching logic**
   - Implement helper like `matchArtistTokens(artist *Artist, query string)`.
   - Ensure case- and accent-insensitive comparisons using normalized runes.
4. **Streamline `Store.Search`**
   - Update to scan the token index sequentially; remove mutex/order bookkeeping.
   - Support simple ranking (exact match > prefix > substring).
5. **Revamp suggestions**
   - Choose between client-side JSON index or slim server-side list; document decision.
   - Implement minimal suggestion formatter that outputs text + URL.
6. **Deprecate legacy APIs**
   - Remove unused search structs and endpoints once new solution is wired.
   - Update handlers/templates to consume new search/suggestion data.
7. **Test & benchmark**
   - Write unit tests for tokenization and search ranking.
   - Benchmark search throughput to ensure sub-millisecond responses.

## Phase 4 – Web Layer & Template Simplification

1. **Define shared view models**
   - Create reusable page structs (`Page`, `ListPage`, etc.) in `internal/web` or a new `view` package.
   - Update handlers to populate these structs instead of anonymous maps.
2. **Create request helpers**
   - Implement helpers for method validation, filter parsing, and error responses.
   - Ensure middleware sets `Allow` headers automatically for restricted methods.
3. **Move business logic to data layer**
   - Strip sorting/filtering out of handlers; rely on catalog/store helpers.
4. **Centralize template compilation**
   - Compile templates once at startup with consistent logging and error handling.
   - Expose a single render helper responsible for headers and error paths.
5. **Refresh templates**
   - Update `.tmpl` files to consume new view models and helper functions.
   - Remove redundant comments and ensure consistent naming.
6. **Regression testing**
   - Run integration and Playwright smoke tests (where available).
   - Manually verify key pages (`/artists`, `/locations`, search results).

## Phase 5 – Code Polish & Documentation

1. **Audit naming and comments**
   - Remove redundant comments; keep rationale-level notes only.
   - Normalize naming (drop `Get` prefixes, standardize `BySlug`, `ByID`, etc.).
2. **Tighten helper scope**
   - Ensure helper packages are narrowly scoped and colocated with consumers.
   - Move slug/country utilities near data catalog if not already done.
3. **Documentation updates**
   - Update `README.md` to reflect simplified workflows and search UX.
   - Summarize refactors in existing `doc/` plans; add changelog entry if needed.
4. **Optional CLI tooling**
   - If helpful, build a tiny CLI (`cmd/catalogdump`) to output normalized data for debugging.
5. **Quality sweep**
   - Run `gofmt`, `go vet`, `staticcheck` (if available), and the full test suite.
   - Capture before/after metrics if performance changed.

## Phase 6 – Testing & Validation

1. **Expand unit coverage**
   - Update data tests for new helper methods (`ConcertCount()`, etc.).
   - Add tests for filter predicates, token indexing, range/set helpers.
2. **Update integration tests**
   - Refresh handler tests to assert new view models and flows.
   - Ensure edge cases: artists without concerts, unknown slugs, empty filters.
3. **Run end-to-end suites**
   - Execute Playwright/visual/E2E tests; update fixtures as needed.
   - Verify deployment scripts or CI configs reflect new packages.
4. **Quality gates**
   - Standardize on `go test ./...`, `go vet ./...`, lint, and smoke `go run ./cmd/server`.
   - Automate in CI; document steps in `tests/` or `doc/`.
5. **Sign-off and rollout**
   - Review changes with stakeholders; ensure rollout notes (feature flags, backout plan) are ready.
   - Prepare final summary for maintainers and archive validation evidence.

---

## Cross-Phase Milestones

- **Change log**: Track significant refactors per phase.
- **CI automation**: Ensure pipelines execute lint/tests after each phase.
- **Monitoring**: Collect load/search metrics before and after major changes.
- **Communication**: Share progress updates weekly with links to merged PRs and docs.
