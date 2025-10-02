# GitHub Copilot Simplification & Optimization Plan (Oct 2025)

## Guiding principles
- Embrace idiomatic Go constructs and the KISS principle: keep functions short, compose behavior with small helpers, and avoid bespoke abstractions.
- Prefer immutable, precomputed data flows and predictable concurrency over lazy recalculation or global state.
- Remove transitional/compatibility APIs so every layer depends on the narrowest surface area possible.
- Introduce concurrency only when it lowers latency or contention and can be reasoned about with simple synchronization primitives from the standard library.

## Key observations from current codebase audit
- **Boot flow coupling (`cmd/server`, `internal/app`)**: configuration is exported package-level state in `internal/config` and `NewServer` is responsible for wiring, logging, and HTTP server creation in one block, making alternative entry points, tests, or graceful shutdown harder.
- **API client**: thin wrapper without retry/backoff or shared transport – each call constructs raw `http.Client` requests; image downloads bypass the API client entirely and ignore context cancellation.
- **Data store (`internal/data`)**: `Store.loadData` mixes orchestration, transformation, and cache bookkeeping. Concurrency is present but ad-hoc (manual channels, inline goroutines) and error paths don’t cancel siblings. Several helpers return large value copies (`Artist`, `Location`) which increases GC churn.
- **Service layer**: still exposes legacy escape hatches (`Store()`), performs manual LRU bookkeeping, and duplicates filtering logic between search and filter paths.
- **Web layer**: handlers build anonymous structs on every render, middleware is globally applied, and form parsing utilities live in templates file. No graceful shutdown or request-scoped context usage past parsing.
- **Testing**: E2E tests spin up full servers repeatedly with duplicated fixture JSON. Unit coverage is decent but lacks targeted tests for concurrency/caching behavior.

## Refactoring roadmap

### Phase 1 – Configuration & startup hygiene
1. Replace `internal/config` globals with a typed `Config` struct and loader that reads from env with defaults. Pass `Config` through `main` → `web.NewServer`. Eliminate mutable package variables.
2. Split `web.NewServer` into builder steps: `newServerDependencies(cfg, apiClient)` (data/service wiring), `newTemplateRegistry(fs)` (compilation), and `newHTTPServer(cfg, handler)`. Return an `*http.Server` plus lightweight `Server` facade so tests can start/stop cleanly.
3. Implement graceful shutdown: expose `Server.Run(ctx)` that listens for context cancellation and calls `httpServer.Shutdown(ctx)`.

### Phase 2 – API + data ingestion pipeline
1. Introduce a shared `http.Transport` configured on the API client and inject it into the image downloader to reuse connections and honor timeouts.
2. Refactor `Store.Load` orchestration:
   - Extract `fetchArtistsAndRelations(ctx)` that runs the two API calls under a `sync.WaitGroup` with an error channel and early cancellation via `context.WithCancelCause`.
   - Replace manual anonymous structs with a small `type fetchResult[T any] struct { data T; err error }` helper to reduce duplication and keep zero-copy semantics.
3. Decompose `processArtists` into targeted stages (`hydrateBaseArtist`, `attachConcerts`, `sortArtists`) and ensure they operate on pointer slices to avoid repeated struct copying.
4. Rework index/filter precomputation as a single `buildIndexes(artists []*Artist) (indexes StoreIndexes, err error)` function returning a struct that groups all maps/slices. Populate the store atomically in one assignment to avoid partially-initialized states if a goroutine panics.
5. Extend image caching to accept context and limit outstanding downloads. Replace the current `atomic` counters with a channel-based worker pool that tracks results and errors; bubble errors up (e.g., disk full) instead of silently ignoring.

### Phase 3 – Service boundary tightening
1. Drop `Service.Store()` and any callers to enforce encapsulation. Tests should rely on public service methods or purpose-built helpers.
2. Replace custom search LRU with `container/list` + map helper in ~40 LOC, encapsulated inside a `type lruCache struct` with unit tests. Expose size tuning via config.
3. Consolidate artist filtering so both `FilterArtists` and `SearchArtists` call a single `applyArtistFilters([]*data.Artist, params)` helper that short-circuits early and works on pointers.
4. Add read-only accessor methods returning slices of pointers to avoid copying full structs for read-only flows (e.g., `Artists()` → `[]*data.Artist`). Keep exported value-returning variants for templates if needed but document the cost.

### Phase 4 – Web layer simplification
1. Move form parsing helpers from `templates.go` into a new `forms.go` file, keeping template compilation focused on rendering concerns.
2. Introduce typed view models for recurring pages (`ArtistsPage`, `LocationsPage`, etc.) in `internal/web/viewmodel.go`. This reduces repeated anonymous struct definitions and makes template evolution safer.
3. Wrap handlers with a common `withPageData` helper that injects site-wide fields (title, suggestions, stats) so each handler only fills the delta.
4. Replace manual path validation (`validateExactPath`) with mux patterns: use `http.StripPrefix` + explicit subrouters (`/artists/{slug}` equivalent with `http.ServeMux` by composing). Alternatively, embed the validation in helper `matchExactPath(r, "/artists")` returning bool to avoid extra render passes.
5. Ensure every handler receives the request context when calling service functions (e.g., `s.svc.FilterArtists` doesn’t currently accept a context; add optional context-aware variants where latency could grow, such as search).
6. Add graceful shutdown endpoint in dev mode (`/dev/shutdown`) that cancels the server context for manual testing.

### Phase 5 – Testing, tooling, and safety nets
1. Extract shared mock API setup into `tests/testserver/mock_api.go` with builders for different datasets; reuse across E2E tests to cut duplication and speed up changes.
2. Add focused unit tests for new concurrency helpers (fetch coordination, image cache worker pool, LRU eviction) to keep regressions cheap.
3. Wire `go test ./...` into CI with race detector (`-race`) at least on key packages (`internal/data`, `internal/service`). Update docs with command snippets.
4. Provide lightweight benchmarks (`internal/data/store_bench_test.go`) measuring load time with cache on/off to guard concurrency refactors.

## Concurrency & performance checkpoints
- Target under 250ms cold start for data load against local fixtures; track via test harness logging.
- Ensure image caching worker pool obeys `withCache` flag and respects configurable worker counts; default to min(4, runtime.NumCPU()).
- Propagate context cancellation through API fetch, image download, and any future long-lived operations so shutdown and timeouts behave predictably.

## Risk mitigation & rollout strategy
- Gate each phase behind small PRs with tests. Begin with configuration changes to avoid mass churn later.
- Maintain compatibility templates until view models land; introduce them behind feature flags (`enableViewModels` build tag) if needed.
- Document new configuration contract in README and include migration notes for existing scripts.

## Definition of done
- All compatibility shims (globals, `Service.Store()`, ad-hoc structs) removed.
- Store loading emits a single log line with timing and cache counts, backed by benchmarks.
- Web handlers consume typed view models and rely on shared helpers; duplicated form parsing eliminated.
- CI executes race-enabled tests; docs describe new architecture succinctly.
