# GitHub Copilot Refactoring Plan v3
**Date:** 2025-10-01

## 🎯 Objectives
- Simplify the Groupie Tracker codebase while preserving all current behaviour.
- Apply idiomatic Go and the KISS principle to every layer.
- Introduce concurrency only where it yields measurable wins (API ingestion, image caching, expensive precomputations).
- Maintain the existing three-layer separation (API → Domain → Web) but reduce friction between layers.

---

## 🔍 Current Pain Points
- **Monolithic repository logic:** `internal/domain/repository.go` (~500 LOC) interleaves API mapping, caching, indexing, statistics, filtering, and search. This makes it hard to reason about and test in isolation.
- **Duplicated API models:** `internal/domain/models.go` still mirrors API structs (`APIArtist`, `APIRelation*`) despite the dedicated `internal/api` package already exporting their canonical definition.
- **Sequential data pipeline:** `Repository.LoadData` performs network calls, transformation, and image caching serially. Slow API responses or large image sets block the entire boot sequence.
- **Mixed concerns in web layer:** `internal/web/handlers.go` (~530 LOC) intermixes routing validation, form parsing, caching decisions, and template orchestration in single methods.
- **Ad-hoc caching:** Search caching, suggestion caching, and filter option caching live as loose maps without eviction semantics or observability hooks.
- **Test fragility:** Unit tests rely on custom setters like `SetTestData` instead of testing the pipeline through clear seams (store/service interfaces), making refactors risky.

---

## 🧭 Guiding Principles
1. **Idiomatic Go first:** favour small packages, explicit constructors, context-aware APIs, and narrow interfaces.
2. **KISS:** prefer straightforward data flows over abstraction factories; encode invariants in types rather than comments.
3. **Concurrency with purpose:** introduce goroutines only for IO-bound or CPU-heavy stages where profiling indicates benefit. Guard shared state via channels or scoped mutexes.
4. **Golden ratio of structure:** keep the three core packages but split oversized files into cohesive siblings (e.g., `loader.go`, `filters.go`, `search.go`) instead of spawning deeply nested packages.
5. **Test-driven evolution:** every structural change ships with focused unit tests and at least one end-to-end regression test.

---

## 🏛️ Target Architecture Overview
```
cmd/server/main.go
    ↓ (DI)
internal/web        ─────► request handling, template orchestration
    ↓ (read-only access)
internal/domain     ─────► Store (data) + Services (business operations)
    ↓ (API boundary)
internal/api        ─────► HTTP client + raw payload models
```
- **Store-Service split inside `domain`:** keep a single package but introduce two structs:
  - `Store`: owns immutable slices/maps, indexes, statistics. Only `Load(ctx)` mutates internal state.
  - `Service`: wraps a `*Store` and supplies filtering, search, suggestions, adjacency helpers. Methods become thin, readable, and concurrency-safe.
- **Centralised caching types:** replace ad-hoc maps with reusable `cache.Map[K, V]` using generics and bounded size policies.

---

## ⚙️ Concurrency Enhancements
1. **Parallel API downloads:** use `errgroup.Group` in `Store.Load` to fetch artists and relations concurrently; share a context with cancellation to fail fast.
2. **Bounded image caching:** swap the serial loop in `cacheImages` for a worker pool (`max(4, runtime.NumCPU()/2)`) that streams download jobs over a channel, reporting totals via atomic counters.
3. **Concurrent index building:** while images download, build countries, concerts, and location aggregates in parallel goroutines that feed results into the store via channels. Synchronise with `errgroup`/`WaitGroup` to keep flow explicit.
4. **Precomputations on demand:** lazily compute search suggestions and filter options using `sync.Once` to avoid upfront cost but guarantee thread-safety.

---

## 🧩 Package-Level Refactors
### internal/api
- Keep as-is, but add context-aware helpers (`Fetch[T any](ctx, path)`) to remove duplication between artist & relation calls.
- Add request logging hooks (optional functional options pattern) for observability during refactors.

### internal/domain
1. **Restructure files:**
   - `store.go`: definition of `Store`, constructor, `Load`, accessors.
   - `loader.go`: API ingestion & mapping logic (now concurrent).
   - `cache.go`: image caching worker pool.
   - `locations.go`: location aggregation helpers.
   - `filters.go`, `search.go`: trimmed, pure functions.
2. **Remove duplicate API structs:** depend directly on `api.Artist`/`api.Relation`.
3. **Encapsulate maps:** expose read-only views via slices; avoid leaking mutable references.
4. **Introduce `Service`:**
   ```go
   type Service struct { store *Store }
   func NewService(store *Store) *Service
   func (s *Service) Artists(ctx context.Context, filters ArtistFilterParams) ([]Artist, error)
   ```
   where filtering/search functions become methods on `Service`, keeping `Store` write-free after load.
5. **Adopt functional options for `Load`:** allow toggling cache, worker count, and future metrics without parameter bloat.
6. **Instrument with lightweight metrics hooks (interfaces) to ease future observability.**

### internal/web
1. **Split handlers:** move artist, location, search, and utility handlers into dedicated files (e.g., `handler_artists.go`).
2. **Service injection:** `Server` now holds `*domain.Service` instead of talking directly to the `Store`. Template data structs shrink.
3. **Middleware cleanup:** convert middleware chain to small, composable functions with clear signatures; extract shared error handling.
4. **Form parsing helpers:** centralise parsing/validation in `forms.go`, returning typed filter structs.
5. **Template loading:** move template globbing into `templates/registry.go` with explicit template names; add checksum logging to detect stale templates.

### templates & static assets
- No semantic changes, but document required fields per template; align with new service return shapes if fields change name/order.

### config & startup
- Replace `Server.ListenAndServe` wrapper with direct `http.Server` usage plus a lightweight `boot.Report()` that prints links and cache stats (fulfilling prior “bakingInfo” request from prompt history).
- Consolidate configuration (env + defaults) via `config.Load()` returning a struct instead of globals.

---

## 🧪 Testing & Tooling
- **Unit coverage:** introduce table-driven tests for new concurrency code (mocking API client and image downloader via interfaces).
- **Race detector:** add `go test -race ./...` to CI once concurrency lands.
- **Benchmarks:** add micro-benchmarks for filtering/search to ensure refactors don’t regress performance.
- **E2E harness:** reuse existing `cmd/server/e2e_test.go`, but update to run against the new service layer seams (inject fake store data without relying on `SetTestData`).

---

## 🗺️ Implementation Roadmap
1. **Foundations (Day 1):**
   - Extract duplicate API structs, update imports.
   - Introduce `Store` struct and migrate repository fields incrementally.
2. **Concurrent loader (Day 2):**
   - Implement `Load(ctx)` using `errgroup` and worker pool.
   - Add targeted unit tests + benchmarks; run race detector.
3. **Service layer & handler rewiring (Day 3):**
   - Add `Service`, rehome filtering/search.
   - Refactor handlers to consume service outputs.
4. **Caching overhaul (Day 4):**
   - Introduce generic bounded cache, replace search cache maps.
   - Wire metrics/logging for hit/miss counts.
5. **Polish & docs (Day 5):**
   - Update README, diagrams, developer docs.
   - Run full `go test ./...`, `go test -race ./...`, and regenerate coverage summary.

Each phase ends with commits + updated `todo.md`, ensuring incremental, reviewable progress.

---

## ⚠️ Risks & Mitigations
- **Concurrency bugs:** guard shared state via channels/immutable slices; run race detector nightly.
- **Template drift:** create snapshot tests that render templates with fixture data verifying required keys.
- **API contract changes:** keep `api.Client` thin and mockable; integration tests hit a local `httptest.Server` to detect schema drift early.

---

## ✅ Expected Outcomes
- Load times reduced (parallel downloads + caching pool).
- Easier navigation of domain logic thanks to store/service split and smaller files.
- Clearer separation of concerns in the web layer, enabling future feature work without touching unrelated handlers.
- Stronger test harness covering concurrency and business rules, supporting confident iteration.
