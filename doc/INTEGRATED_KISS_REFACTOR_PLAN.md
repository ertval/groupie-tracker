# Groupie Tracker Integrated KISS Refactor Plan (2025-09-29)

## Executive Summary
This document unifies the recommendations from the C4 Refactoring Plan, KISS Refactor Plan, and Simplification Plan into a single roadmap focused on performance, simplicity, idiomatic Go, and the KISS principle. The strategy eliminates accidental complexity, tightens the HTTP layer, and optimizes data processing without altering public behavior or test surfaces.

## Guiding Principles
- **Prefer directness over abstraction**: keep only the interfaces consumers need and rely on concrete types elsewhere.
- **Compute once, reuse everywhere**: precompute expensive, deterministic artifacts at startup.
- **Keep handlers resilient**: centralized error handling, panic recovery, and no panics in hot paths.
- **Lean, readable Go**: no redundant comments, avoid reflection, and favor small, intention-revealing helpers.
- **Maintain TDD discipline**: write/adjust tests before implementation changes and run the full suite after each phase.

## Consolidated Pain Points
| Area | Key Issues | Impact |
|------|------------|--------|
| Service façade | `services.go` pass-through interfaces, redundant BaseTemplateData wrapper | Noise, extra allocations, cognitive load |
| Data lifecycle | Multi-pass artist processing, repeated location lookups, per-request suggestion generation | Startup cost, per-request CPU, memory churn |
| Filtering/Search | Recomputes country maps, lacks cached options, `/api/suggestions` disabled | Slower filters, missing audit endpoint |
| HTTP layer | Reflection helpers, scattered error handling, over-commented utilities | Harder to reason about, brittle |
| Configuration | Globals accessed ad hoc, inconsistent HTTP client usage | Harder to test/tune timeouts |

## Roadmap Overview
| Phase | Objective | Highlights |
|-------|-----------|-----------|
| 0 | Establish baselines & guardrails | Freeze API surface, capture current metrics, ensure gofmt/go test hygiene |
| 1 | Collapse redundant abstractions & restore endpoints | Remove service layer, streamline `Server`, re-enable `/api/suggestions`, centralize error responses |
| 2 | Cache deterministic data | Precompute search suggestions, filter options, and global stats; expose getters with defensive copies |
| 3 | Optimize repository pipeline & filtering | Merge data passes where safe, use maps for location creation, leverage precomputed `Countries`, reuse configured HTTP client |
| 4 | Simplify templates & HTTP utilities | Replace `BaseTemplateData`, remove reflection helpers, split template utilities/forms, prune redundant comments |
| 5 | Configuration & documentation polish (optional) | Consolidate config, document lean architecture, backlog low-risk niceties |

## Phase Details
### Phase 0 – Baseline & Guardrails
- Record current startup time, memory footprint during `LoadData`, and response latency for `/artists` & `/search`.
- Confirm all existing tests (`go test ./internal/...`, `./cmd/cli`, `./tests/...`) pass; set up continuous coverage tracking.
- Document API contracts and template expectations to prevent regressions during simplifications.

### Phase 1 – Structural Simplification
- Delete `internal/server/services.go`; update `Server` to hold `repo *data.Repository`, cached artifacts, and templates only.
- Inline repository calls within handlers; keep helper functions only where they provide real logic.
- Restore `/api/suggestions` routing and ensure panic-resistant handler wiring.
- Introduce a shared `handleError` helper to enforce consistent error responses and logging.
- Update server tests to reflect the new struct shape and cached fields.

### Phase 2 – Cached Derived Data
- Extend the repository to compute search suggestions, artist filter options, location filter options, and stats once post-load.
- Expose read-only accessors returning copies to avoid accidental mutation.
- Have `Server.New` (or equivalent) snapshot cached slices/structs; reuse them across handlers and templates.
- Adjust tests to verify caches are built exactly once and reused.

### Phase 3 – Repository & Filtering Optimizations
- Consolidate artist processing: merge redundant passes, compute slug maps, and location aggregates in one traversal.
- Replace `findArtistByID` loops with a prebuilt map; store `Country` on `Location` entities to skip repeated parsing.
- Refine filters to use `artist.Countries` directly and short-circuit on cheap checks first.
- Ensure `downloadImage` uses the repository’s configured `http.Client` and respects context cancellation.
- Add targeted unit benchmarks or microtests to confirm improvements where feasible.

### Phase 4 – Template & HTTP Layer Simplification
- Remove `BaseTemplateData`; provide inline structs per page with only required fields plus cached suggestions when needed.
- Split `templates.go` and `forms.go` from the current utilities, removing reflection-based helpers (`addSuggestionsToData`).
- Reduce comment noise to meaningful doc comments; keep templates relying on explicit data structs.
- Verify template inheritance (`base.tmpl`) still functions with the leaner data payloads.

### Phase 5 – Configuration & Documentation (Optional)
- Introduce a lightweight config struct piped into repository/server constructors, or centralize accessors within `internal/config`.
- Replace map-based stats with a typed struct if templates reference fixed keys.
- Update architecture documentation and README to describe the simplified flow and caching strategy.

## Architectural Adjustments
- Keep the standard-library-only constraint; avoid introducing external packages.
- Consider renaming `internal/server` to `internal/web` only after immediate wins land and tests pass.
- Maintain global repository/template variables for compatibility, but ensure they are initialized exactly once and nil-safe.

## Validation Strategy
1. Unit tests per package before/after each phase (`go test ./internal/data`, `./internal/server`).
2. Integration & audit suites after structural changes (`go test ./cmd/cli`, `go test ./tests/...`).
3. Manual smoke via `go run ./cmd/cli` for `/health`, `/artists`, `/artists/{slug}`, `/search`.
4. Optional: capture `pprof` snapshots pre/post phases 2–3 to quantify allocation reductions.

## Success Metrics
- Eliminate ~500 LOC of indirection/service boilerplate.
- Reduce per-request search suggestion generation from O(n) to O(1) via caching.
- Lower filter allocations by reusing precomputed country slices and avoiding temporary maps.
- Maintain or improve test coverage (baseline ~70% for `internal/data`).
- Keep startup time and memory usage steady or improved after pipeline optimizations.

## Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Hidden dependency on service interfaces | Refactor incrementally, run audit tests; provide temporary adapters if needed |
| Cache invalidation for future hot reloads | Encapsulate caches within repository; reset on reload hook if implemented later |
| Template regressions due to data shape changes | Update templates alongside handlers, rely on integration tests |
| Over-optimization reducing readability | Keep helpers small, document rationale, prefer clarity over micro-optimizations |

## Backlog & Stretch Goals
- Replace map-based stats with a typed struct for compile-time safety once templates are updated.
- Add lightweight benchmarks for critical repository methods to prevent regressions.
- Explore dynamic data reload support after core simplifications stabilize.
