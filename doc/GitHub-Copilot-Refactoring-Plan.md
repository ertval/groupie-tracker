# GitHub Copilot Refactoring Plan

## Objectives
- Collapse legacy compatibility layers while keeping the codebase idiomatic, testable, and easy to maintain.
- Preserve existing features (server-side filtering, suggestions, stats) with simpler, single-responsibility components.
- Leverage concurrency where it meaningfully lowers startup and request latency without overcomplicating control flow.
- Reduce duplication of data access logic so the web layer consumes pre-computed, read-only data in constant time.

## Current Pain Points
- **Triple layering (`Store ➜ Repository ➜ Service`)** duplicates APIs, requires redundant state (`Repository.syncFromStore`), and forces tests to reach through wrappers.
- **Derived data is scattered**: `Store`, `Repository`, and handlers each compute pieces of stats, filter options, and navigation hints.
- **Concurrency stops at fetch**: we already fetch artists/relations in parallel, but aggregation (indexing, statistics, cache priming) runs sequentially even though it operates on disjoint data.
- **Web layer carries caches** (suggestions, filter options, search cache) that would be cleaner and safer inside the data layer once data is immutable.
- **Testing hooks (`SetTestData`)** exist only for the compatibility wrapper and can be replaced with constructor helpers once the store API is streamlined.

## Final Architecture (Oct 2025)
1. **Data Package (`internal/data`)**
   - `Store` is the single exported type. `NewStore(apiClient, withCache)` plus `Store.Load(ctx)` encapsulate concurrent API fetches, image caching, and all derivations.
   - Read-only getters expose artists, locations, statistics, search suggestions, and filter option metadata.
   - Derived indexes include an `artistPositions` map that enables O(1) adjacency lookups for detail pages.

2. **Service Layer (`internal/service`)**
   - Separate package that wraps the store and adds business logic only: filtering, search, adjacency helpers, and a mutex-protected search cache.
   - Filtering uses precomputed metadata; search merges text queries with filters and caches simple lookups (50-entry LRU).
   - All concurrency and state live inside the store; the service is read-only after load.

3. **Web Layer (`internal/web`)**
   - `Server` receives its dependencies from `internal/app.Initialize`, which wires the store and service and performs the initial load with a startup timeout.
   - Handlers focus on HTTP concerns, parsing form data into typed params before calling the service.
   - Template compilation, middleware, and caching responsibilities remain in the web package; all data comes from the service/store.

## Completed Refactor Summary

### Data Loading & Store
- `internal/domain` was retired in favour of `internal/data`. The store now encapsulates all concurrency with a `sync.WaitGroup`-driven pipeline that fetches artists/relations in parallel and derives indexes, locations, suggestions, and stats concurrently.
- `NewStoreFromFixtures` mirrors the production pipeline, allowing tests in other packages to create fully-initialised stores without reaching into internals.
- Artist/location slices are immutable after load; maps (`byID`, `bySlug`, `artistPositions`, `locationsBySlug`) provide O(1) lookups.

### Derived Data & Caching
- Search suggestions, filter options, and statistics are computed during load and cached on the store.
- Image caching remains optional (4-worker pool). Cache status propagates through the store/service for logging and UI badges.
- Artist adjacency now relies on the precomputed `artistPositions` map instead of linear scans.

### Service & Search
- Filtering and search moved into the standalone `internal/service` package. The service is a thin façade with no duplicated state—just references to the store plus a mutex-protected search cache.
- Search combines text queries with filters and caches plain-text results (up to 50 entries) to avoid recomputing popular lookups.
- Filtering logic leverages cached metadata (`MemberCount`, `FirstAlbumYear`, `Countries`) so requests avoid runtime parsing.

### Web Layer
- `internal/app.Initialize` wires the store/service, performs the initial load with a timeout, and returns both to the web layer.
- `internal/web.Server` now stores both `store` and `svc`, keeping HTTP concerns isolated while relying on the service for data.
- Handlers remain split by concern and only handle form parsing + rendering; all data access goes through the service API.

### Testing & Tooling
- Service-level unit tests in `internal/service` cover filtering, search, and caching. Repository-based tests were removed with the compatibility layer.
- E2E tests (`cmd/server`) and smoke checks for `/health` continue to operate through `httptest.Server`.
- `go test ./...` is the canonical verification step and passes across the refactored tree.

### Documentation & Operations
- README now describes the store/service/web architecture and the new dependency wiring.
- Legacy references to the repository layer were removed; the refactor plan itself documents the final state for future contributors.
