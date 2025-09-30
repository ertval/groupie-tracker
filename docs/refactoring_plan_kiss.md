# Groupie Tracker Refactoring Plan (KISS-Oriented)

_Last updated: 2025-09-30_

> ✅ **Implementation note (2025-09-30):** The refactor described below is now complete. The project tree matches the "Target Architecture", global singletons have been removed, and new unit/integration tests have been added for the data pipeline, search service, and HTTP handlers. The document is retained for historical context.

## Goal

Restore a simple, idiomatic Go structure for the Groupie Tracker project by eliminating redundant logic, clarifying ownership of runtime state, and reshaping the folder layout so each package has one clear responsibility. The plan follows the KISS principle while keeping all existing user-facing behaviour intact.

---

## Current Pain Points

1. **Global mutable state everywhere**  
   `internal/data.go` and `internal/handlers.go` both declare package-level singletons (`store`, `dataStore`, `suggestions`, `templates`). This hides dependencies, makes tests order-dependent, and prevents parallel execution.

2. **Mixed concerns per file**  
   `internal/data.go` contains API DTOs, domain models, HTTP fetch logic, transformers, and helpers in one place. `internal/templates.go` mixes template helpers with middleware functions. `internal/handlers.go` combines router wiring, handlers, and environment probing.

3. **Duplicated / misleading utilities**  
   - `extractYearFromDate` and `extractYearFromAlbum` duplicate logic.  
   - `getRandomArtists` is deterministic but named "random".  
   - `hasField` always returns `true`, effectively dead code.  
   - Suggestion generation lowercases keys ad hoc instead of via one normalizer.

4. **Flattened package namespace**  
   Everything lives in package `data`, so HTTP handlers can reach inside the data layer without contracts. There is no clear distinction between API DTOs, internal models, search utilities, or HTTP concerns.

5. **Testing blind spots**  
   Tests (manual or automated) must rely on global state. Handlers cannot be spun up against a stub repository, and data transformations are hard to validate in isolation.

---

## Target Architecture

```
cmd/cli/main.go            # Entry point wires dependencies and starts server
internal/
  api/                     # Groupie API client + DTOs
  data/                    # Domain models + Store (loads, indexes, exposes getters)
  search/                  # Search + filter service (no globals)
  server/                  # HTTP server, handlers, middleware, template manager
  config/                  # Ports, base URLs, timeouts, feature toggles
```

Key principles:
- Each package exports a small constructor (`NewStore`, `NewService`, `NewServer`).
- All state sits on structs; no package-level mutable globals.
- Handlers depend on narrow interfaces (`ArtistReader`, `LocationReader`) to support testing and future backends.
- Templates and middleware live with the HTTP layer, not in data helpers.

---

## Refactoring Steps

### Phase 1 — Establish Baseline & Contracts
1. Add lightweight tests (or snapshots) capturing current outputs for: `/artists`, `/artists/{slug}`, `/locations/{slug}`, `/api/suggestions`.
2. Define minimal interfaces in `internal/data` for read access (e.g., `GetArtists() []Artist`, `GetArtist(slug string) (Artist, bool)`).
3. Replace `store` and `dataStore` globals with a single `Store` struct returned from `LoadData(context.Context)`; update callers to depend on the struct instance.

### Phase 2 — Package Realignment
1. Create `internal/api` with `Client` (wraps HTTP fetching) and move API DTOs (`APIArtist`, `APIRelationIndex`).
2. Split `internal/data.go` into:
   - `models.go`: `Artist`, `Location`, `Concert`, `AppStats`.
   - `store.go`: `Store` struct, `Load` method, index builders.
   - `transform.go`: `processArtists`, `createLocations`, `calculateStats`, slug/year helpers.
3. Move filter/search logic into `internal/search/` with a `Service` struct owning suggestion cache and helper methods (`Search`, `Suggest`, `Filter`).

### Phase 3 — Server Simplification
1. Introduce `server.Server` struct holding dependencies (`repo`, `search`, `templates`, `router`).
2. Turn each handler into a method (`func (s *Server) Home(w http.ResponseWriter, r *http.Request)`) so they access dependencies via the struct.
3. Relocate middleware into `internal/server/middleware.go`; apply within `NewServer` when constructing the mux.
4. Replace `renderTemplate` with a `TemplateManager` type that loads and executes templates. Remove unused helpers such as `hasField`.
5. Update `cmd/cli/main.go` to:
   ```go
   store := data.MustLoad(ctx, apiClient)
   search := search.NewService(store)
   srv := server.New(store, search, config.FromEnv())
   srv.Listen()
   ```

### Phase 4 — Cleanup & Enhancements
1. Rename misleading helpers (`getRandomArtists` → `pickFeaturedArtists`) or implement true randomness using `rand.Source` seeded once.
2. Consolidate string normalization into one helper (used by both search and suggestions) to remove ad hoc lowercasing.
3. Delete dead code (`hasField`) and verify template helpers match actual template usage.
4. Refresh documentation (`README.md`, `docs/`) with the new package layout and simplified runtime story.

### Phase 5 — Testing & Tooling
1. Write focused unit tests for `processArtists`, `createLocations`, `Search`, and handler responses using `httptest`.
2. Add integration test that spins up `server.Server` on an ephemeral port and checks key routes return 200/404/500 as expected.
3. Ensure `go test ./...` runs clean; wire this command into CI if not already present.

---

## Risks & Mitigations

| Risk | Mitigation |
| --- | --- |
| Behaviour drift after moving handlers into a struct | Capture baseline responses before refactor; keep template inputs identical. |
| Increased init time due to additional constructors | Reuse existing transformation helpers; avoid repeated computations in constructors. |
| Template mistakes after helper cleanup | Add compile-time template tests (`template.Must`) and run an integration smoke test. |
| Search suggestion regression | Provide unit tests using fixed artist fixtures to assert suggestion count and contents. |

---

## Definition of Done

- No package-level mutable globals remain; all dependencies are owned by structs created in `main`.
- Package structure matches the "Target Architecture" tree above.
- Tests cover data transformations, search/filter logic, and HTTP handlers; `go test ./...` passes.
- Documentation and README describe the new layout and simplified runtime story.
- Manual smoke test confirms templates render correctly and critical endpoints behave as before.
