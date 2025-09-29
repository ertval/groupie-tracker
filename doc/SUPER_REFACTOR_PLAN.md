# Groupie Tracker Super Refactor Plan

## Executive Summary
- Streamline the application into an idiomatic, standard-library-only Go web service while protecting audit invariants (Queen members, Gorillaz first album date, Travis Scott locations, Foo Fighters member count).
- Merge the structural simplifications proposed in `REFACTORING_PLAN.md` with the incremental hardening tactics from `5_REFACTOR_PLAN.md` to yield a phased roadmap that balances risk reduction and architectural payoff.
- Embrace test-driven development: every change lands with failing tests first, then implementation, keeping `go test ./internal/...` and targeted e2e suites green at every checkpoint.

## Non-Negotiable Constraints
- Go 1.24.3, standard library only (no third-party modules); keep `go.mod` dependency-free.
- No JavaScript; all UX improvements occur through HTML templates and server-side form handling.
- Panic-safe handlers, centralized error templates, and resilient startup flows.
- Serve as audit reference: keep `/health`, `/artists`, `/artists/{slug}`, `/locations`, `/locations/{slug}`, `/search`, and `/api/suggestions` endpoints compliant.

## Guiding Principles
1. **KISS + Go idioms** — prefer functions, slices, and maps over heavy structs or service layers.
2. **Progressive simplification** — harden current code paths before large structural changes to avoid regressions.
3. **Single responsibility** — each file owns one concern (data loading, filtering, search, handlers, templates).
4. **Strategic caching only** — templates and search suggestions remain cached; other caches require profiling proof.
5. **Continuous validation** — unit, integration, and audit tests run at the end of every phase; failures get fixed before moving forward.

## Phase 0 – Baseline & Safety Net (Day 0-1)
### Objectives
- Capture the current behaviour, metrics, and weak spots before touching architecture.
- Establish tooling for fast feedback.

### Actions
- Run and document `go test ./...`, `go test ./internal/...`, and `go test ./tests/...`; note flaky suites.
- Snapshot key metrics (coverage summary, response timing via lightweight curl/smoke test on `/health`).
- Inventory caches, map usages, and template helpers to inform downstream work.
- Open tracking issues for each major refactor epic (Data Layer, Templates, Server) to stage PRs.

### Deliverables
- Baseline test report saved to `tests/E2E_TEST_AND_COVERAGE_REPORT.md` addendum.
- Refactor kanban or issue list with owners and acceptance criteria.

## Phase 1 – Data Layer Evolution (Week 1-2)
Two sub-phases ensure a safe migration from the existing repository to the lean `DataStore` envisioned in the refactor plan.

### Phase 1A: Repository Hardening (Week 1)
- Convert repository maps to store `*Artist`; maintain a parallel `artistIndex map[int]int` for O(1) adjacency lookups.
- Split `internal/data/repository.go` into focused files (`repository_core.go`, `repository_locations.go`, `repository_images.go`, `repository_helpers.go`) without altering behaviour.
- Remove duplicate helpers by consolidating empty-filter detection in `internal/data/search.go` and reusing it from server handlers.
- Revisit `searchCache`: either implement a documented FIFO eviction (if profiling shows benefit) or remove the cache pending future evidence.
- Write regression tests for pointer-based storage, adjacency lookup, and helper reuse.

### Phase 1B: Transition to Simplified Data Store (Week 2)
- Introduce `internal/data/store.go` exposing `LoadData(ctx) (*DataStore, error)` that returns `Artists`, `Locations`, and derived filter/search scaffolding.
- Gradually migrate repository consumers to the `DataStore`, keeping both layers temporarily until tests pass.
- Remove redundant caches (navigation links, query caches, image caching) once the simplified store delivers equivalent behaviour.
- Update models to remain concise (~100 lines) and relocate filter/search data structures to their dedicated files.
- Ensure `filters.go` and `search.go` operate on plain slices supplied by the new store.

### Exit Criteria
- All data operations reference the new `DataStore`; legacy repository constructs deleted or shimmed only for tests.
- Data tests (`internal/data/*_test.go`) green with additional coverage for pointer storage and store migration.

## Phase 2 – Package Flattening & Configuration Simplification (Week 2-3)
### Goals
- Align directory layout with the simplified data layer while retaining clarity.
- Reduce config indirection for easier testing.

### Actions
- Collapse `internal/data` and `internal/server` granularity into the target structure:
  - `internal/data.go` (models + load)
  - `internal/filters.go`
  - `internal/search.go`
  - `internal/handlers.go`
  - `internal/templates.go`
- Retain subdirectories temporarily if needed, but add build tags or TODOs to track pending merges.
- Move configuration defaults closer to consumers (handlers) while keeping overrides centralized in `internal/config` for tests.
- Update imports, module documentation, and gofmt/goimports across the codebase.

### Exit Criteria
- `internal/` matches the target five-file structure (modulo ongoing tests).
- Builds and tests succeed without circular imports or orphaned helpers.

## Phase 3 – Template & Presentation Streamlining (Week 3)
### Objectives
- Eliminate template DTO duplication and push formatting logic to templates and helper funcs.

### Actions
- Delete `internal/server/template_data.go`; replace template structs with domain models in handlers.
- Expand `template.FuncMap` with helpers (`join`, `len`, `pluralize`, `formatDate`, etc.) so templates can derive display strings directly.
- Update templates (`templates/*.tmpl`) to rely on domain data (`{{len .Artist.Members}}`) and new helpers instead of preformatted strings.
- Ensure `templates` compile at startup; maintain panic-safe rendering with descriptive error pages.
- Refresh CSS only if needed to accommodate template data changes (no JS additions).

### Tests & Validation
- Extend template-related tests (possibly via golden files) to confirm expected HTML snippets for artist details and search results.
- Manual smoke test: fetch `/artists`, `/artists/{slug}`, `/search` to confirm formatting and filter UX.

## Phase 4 – Server & Caching Simplification (Week 3-4)
### Goals
- Collapse the server struct into a lean handler module while preserving middleware and recovery logic.

### Actions
- Merge `server.go`, `handlers.go`, `routes.go`, `middleware.go`, and `utils.go` into `internal/handlers.go` (core HTTP) and `internal/templates.go` (template compilation & helpers).
- Replace the `Server` dependency-injection struct with package-level state that caches only templates and search suggestions; access `DataStore` directly.
- Ensure middleware (panic recovery, logging, security headers) survives the merge, ideally as plain functions wrapping `http.Handler`.
- Update the CLI entrypoint (`cmd/cli/main.go`) to use the new simplified server start function.
- Remove defunct caches (search query cache, navigation cache) if not already dropped in Phase 1B.

### Validation
- `go run ./cmd/cli` followed by manual `/health` and `/artists` checks.
- Add or update tests in `internal/server` (now `internal/handlers.go`) to cover routing and middleware behaviour.

## Phase 5 – Performance, Testing & Launch (Week 4)
### Objectives
- Confirm the refactor maintains or improves performance and passes audits.

### Actions
- Run full suites: `go test ./internal/...`, `go test ./tests/...`, and `go test ./cmd/...`.
- Execute coverage report (`go test -cover ./internal/...`) and compare to baseline.
- Perform lightweight benchmarks or timing harness around search/filter functions (optional but recommended).
- Update documentation (`README.md`, `doc/*`) to reflect new architecture, commands, and file layout.
- Prepare release notes summarizing behavioural changes, new helper functions, and removal of caches.

### Exit Criteria
- All automated tests pass; manual smoke tests confirmed.
- Documentation updated; repo ready for PR/merge.

## Cross-Cutting Enhancements
- Centralize reusable helpers (slug creation, location normalization) in `internal/data/helpers.go` or similar.
- Replace brittle date parsing with well-tested helpers (unit tests covering API edge cases).
- Maintain audit invariants via dedicated regression tests under `tests/audit_test.go`.
- Track technical debt removed (line count reduction, deleted files) to showcase ROI.

## Risk & Mitigation Matrix
| Risk | Impact | Mitigation |
| --- | --- | --- |
| Data regressions from repository → DataStore migration | High | Maintain dual-layer adapters during Phase 1B, use golden-data tests and API fixtures. |
| Template breakage after DTO removal | Medium | Incrementally migrate templates with feature flags/tests; leverage snapshot testing. |
| Cache removal triggers performance regressions | Medium | Benchmark before/after; reintroduce minimal caching only where data proves necessary. |
| Large PRs become unreviewable | Medium | Slice work into phase-specific branches, keep PRs under ~400 lines, provide migration docs. |
| Test flakiness blocks progress | Low | Quarantine flaky tests early (Phase 0) and stabilize before major refactors. |

## Completion Checklist
- [ ] Baseline metrics captured and stored in docs/tests.
- [ ] Repository hardened with pointer storage and slim files.
- [ ] `DataStore` replaces repository usage across handlers.
- [ ] Package structure flattened; imports updated.
- [ ] Templates consume domain models via helper functions.
- [ ] Server started via simplified handlers module with strategic caching only.
- [ ] Full test suite green with updated documentation.

## Appendix – Mapping Original Plans to Phases
| Source Item | Superplan Phase |
| --- | --- |
| Remove `template_data.go` (doc/5) | Phase 3 |
| Convert repository maps to pointers + indexing (doc/5) | Phase 1A |
| Split repository into focused files (doc/5) | Phase 1A |
| Replace repository with `DataStore` (REFACTORING_PLAN) | Phase 1B |
| Flatten package structure (REFACTORING_PLAN) | Phase 2 |
| Simplify config usage (REFACTORING_PLAN) | Phase 2 |
| Eliminate over-engineered caching (REFACTORING_PLAN & doc/5) | Phases 1B & 4 |
| Merge server files, simplify handlers (REFACTORING_PLAN) | Phase 4 |
| Add template helpers, use domain models (REFACTORING_PLAN) | Phase 3 |
| Continuous testing & documentation (both) | Phases 0 & 5 |
