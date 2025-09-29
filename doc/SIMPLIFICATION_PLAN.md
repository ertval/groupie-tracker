# Groupie Tracker – KISS Refactor Plan

## Context Snapshot
- **Scope reviewed**: `internal/data`, `internal/server`, `cmd/cli` (application code only; docs ignored).
- **Goal**: Remove redundancy, flatten unnecessary abstractions, and prefer idiomatic Go for maintainable performance.
- **Guiding principles**: Minimise moving parts, avoid recomputation, keep APIs narrow, and continue to honour existing tests/TDD workflow.

## Key Pain Points Discovered
1. **Service façade bloat** (`internal/server/services.go`)
   - Interfaces (`ArtistService`, `SearchService`, etc.) merely forward every call to `*data.Repository`.
   - Handlers already reach into `s.repo` directly for some operations (e.g., `Locations`), so the abstraction is leaky *and* redundant.
2. **Repeated heavy computations**
   - `NewBaseTemplateData` regenerates the full suggestion list on every handler call; `SuggestionsAPI` also regenerates then filters.
   - `Repository.matchesArtistFilters` rebuilds a country set from concerts even though `Artist.Countries` is pre-computed.
   - `GetArtistFilterOptions` re-parses countries from concerts instead of using `artist.Countries`.
3. **O(n²) data post-processing**
   - `createLocations` calls `findArtistByID` for every (location, artist) pair even though a full `artistsByID` map already exists logically.
4. **Dead/unapplied template helpers**
   - `internal/server/template_data.go` defines formatter structs/functions that are never referenced outside that file.
   - `addSuggestionsToData` (reflection helper) is unused.
5. **Missing/disabled route**
   - `/api/suggestions` required by audits is commented out in `routes.go`.
6. **Inconsistent caching utilities**
   - `downloadImage` uses `http.Get` instead of the repository’s configured `http.Client`, bypassing timeouts and tests.

## Refactor Roadmap (High → Low Priority)
1. **Collapse the service façade**
   - Remove `internal/server/services.go`.
   - Replace `Server` fields with direct `repo *data.Repository` and a precomputed suggestion cache.
   - Update handlers to call repository methods directly; keep method extraction only where it genuinely adds value.
   - Adjust tests to operate on the simplified `Server` struct.
2. **Cache expensive derived data once**
   - Compute and store search suggestions during `Repository.loadProcessedData`; expose as `GetSuggestions()`.
   - Have `Server` snapshot the values at startup and reuse them for `BaseTemplateData` and `/api/suggestions`.
   - Introduce lightweight filtering (case-insensitive substring) on the cached slice for `/api/suggestions`.
3. **Lean filter logic**
   - Update `matchesArtistFilters` to rely on `artist.Countries` instead of rebuilding maps per request.
   - Rework `GetArtistFilterOptions` to iterate `artist.Countries` directly, avoiding `extractCountryFromLocation`.
   - Extract a helper to test membership in a sorted slice (binary search) to keep lookups O(log n) without extra allocations.
4. **Location creation performance pass**
   - Build a local `map[int]data.Artist` once in `createLocations` (or pass in the repo’s eventual map).
   - While aggregating locations, capture and store the derived country value once (add `Country string` to `data.Location`).
   - Simplify `FilterLocations` to compare against the stored country instead of reparsing names.
5. **Remove unused template scaffolding**
   - Delete `Template*` structs and formatters unless templates start consuming them soon.
   - Drop `addSuggestionsToData` and associated reflection helpers.
   - Review templates to ensure they rely on simple, already-available fields before removing helpers.
6. **Re-enable `/api/suggestions` route**
   - Wire the handler back in via `router.HandleFunc`.
   - Ensure handler uses cached suggestions to avoid per-request recomputation.
7. **Harden cache downloads**
   - Swap `http.Get` with `r.apiClient.Do` inside `downloadImage` for consistent timeouts.
   - Thread a request context (respecting `LoadData`’s context) so image caching obeys cancellation.
8. **Opportunity: avoid linear `GetAdjacentArtists`**
   - Maintain `artistIndexByID` map during `loadProcessedData` to answer adjacency in O(1).
   - Keeps navigation cheap without precomputing neighbours.

## Structural Simplifications
- Keep the current package layout (`internal/data`, `internal/server`, `cmd/cli`) but reduce cross-package indirection.
- Limit `internal/server` to three focused files:
  1. `server.go` (construction + HTTP server)
  2. `handlers.go` (HTTP endpoints)
  3. `templates.go` (rendering + helpers) – merge `template_data.go` & `utils.go` after trimming dead code.
- Move pure utility helpers that don’t need repository state (e.g., `createSlug`, `normalizeLocation`) into a new `internal/data/strings.go` or keep them as private functions in the minimal files where they’re used.

## Test & Verification Checklist
- `go test ./internal/...`
- `go test ./cmd/cli -run Test` (existing e2e tests)
- `go test ./tests/...` (audit suite)
- Manual smoke: `go run ./cmd/cli` → hit `/health`, `/artists`, `/search`.

## Risk & Mitigation Notes
- **Suggestion caching**: ensure tests verify determinism; add a focused unit test on `Repository` to prove the cached slice is built once.
- **Handler simplification**: refactor incrementally and rely on existing tests (especially audit/e2e) to catch regressions.
- **Location country field**: update templates and tests expecting country data sourced from names.

## Next Steps
1. Execute façade removal + cached suggestions (biggest win, unlocks rest).
2. Optimize filtering logic and location creation for runtime savings.
3. Prune unused template helpers and consolidate server utilities.
4. Re-run full test suite and update docs outlining the leaner architecture.
