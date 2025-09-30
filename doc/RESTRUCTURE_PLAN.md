# Groupie Tracker Restructuring Plan

## Purpose
- Reduce package sprawl and mega-files while keeping the codebase approachable for new contributors.
- Align with idiomatic Go: narrow packages, predictable dependency flow, and clear boundaries between domain, transport, and infrastructure.
- Preserve the current behaviour (API consumption, caching, HTML rendering) while making incremental delivery straightforward.

## Design Tenets
- **Keep It Simple**: Fewer layers, no abstract factories, and standard library only.
- **Package Cohesion**: Each package owns one concept (domain types, data access, web transport, orchestration).
- **Dependency Direction**: `cmd` → `internal/app` → domain/services → infrastructure (`repo`, `web`) → views/assets.
- **Test First**: Every move is accompanied by relocating or adding the matching `*_test.go` alongside the code.
- **Incremental Delivery**: Reorganize in slices (domain types first, then repository, etc.) to avoid big-bang rewrites.

## Target Layout Snapshot
```
cmd/
  server/
    main.go           // Bootstraps the HTTP server via internal/app
  tools/
    refresh-data.go   // (Optional) CLI helper to warm caches or run diagnostics
internal/
  app/
    bootstrap.go      // Load config, wire dependencies, expose Run()
    shutdown.go       // Graceful shutdown helpers
  config/             // Unchanged: centralised runtime configuration
  domain/
    artist.go         // Core structs + behaviour-free helpers
    location.go
    filters.go        // Filter/search parameter structs + validation
    search.go         // Suggestion types, value objects
  repo/
    repository.go     // Data loading orchestration & read-only accessors
    loader.go         // API fetch + decoding
    cache.go          // Image caching (kept private to repo)
  service/
    artist.go         // Filter logic over repo data
    location.go       // Location filtering/aggregation helpers
    search.go         // Search orchestration + caching policy
  web/
    server.go         // HTTP server construction (net/http only)
    router.go         // Routes wired to handler methods
    render.go         // Template execution + helper funcs
    handlers/
      home.go
      artists.go
      artist_detail.go
      locations.go
      location_detail.go
      search.go
      health.go
      dev.go
      errors.go
    middleware/
      logging.go
      recovery.go
      security.go
views/
  templates/          // Existing templates moved out of internal
  funcs.go            // Shared template funcs registered by web/render.go
static/               // Unchanged assets (served by web/static handler)
tests/
  e2e/...
```

## Package Roles
- **`internal/app`**: Minimal orchestration. Builds a `Container` struct holding `Repo`, services, and `web.Server`. Exposes `Run()` and `Shutdown()` for `cmd/server`.
- **`internal/domain`**: Pure types + validation helpers. Zero IO, zero exports beyond structs/functions used by other packages. Keeps models near filter/search params to avoid leaking repository internals.
- **`internal/repo`**: Retains current repository semantics but pruned into smaller files (fetch, transform, cache). Only exports interfaces needed by services (`Repository` interface with methods like `Artists()`, `ArtistBySlug`, etc.).
- **`internal/service`**: Stateless helpers built on top of the repository interface. Provides filter/search routines so handlers no longer reach into repo internals.
- **`internal/web`**: Owns HTTP specifics. Handlers are methods on a light `Handler` struct that receives service interfaces in its constructor. Rendering moves to `render.go` for reuse.
- **`views`**: Templates and helper funcs live outside `internal` to allow future tooling (e.g., `html/template` parsing in tests) without import cycles. `internal/web` loads from this path at startup.

## Incremental Migration Steps
1. **Establish Domain Package**
   - Move `internal/data/models.go` contents into `internal/domain` (split by concern).
   - Update imports in tests and other packages.
   - Run `go test ./internal/data ./internal/server` to confirm no behaviour change.

2. **Carve Out Repository Package**
   - Rename `internal/data` → `internal/repo`.
   - Split large `repository.go` into `repository.go`, `loader.go`, `transform.go`, `cache.go` keeping functions private unless needed externally.
   - Introduce an exported interface `type Store interface { Artists() []domain.Artist; ... }` consumed by services/handlers.

3. **Introduce Services Layer**
   - Move filter/search logic from `filters.go`/`search.go` into `internal/service` where each file exposes small stateless constructors (e.g., `NewArtistService(store Store)`).
   - Tests move alongside to `internal/service/*_test.go` reusing existing cases.

4. **Rebuild Web Package**
   - Create `internal/web` with `server.go`, `router.go`, and subfolders `handlers`, `middleware`.
   - Convert existing handler functions into methods on `handlers.Handler` receiving explicit dependencies (`ArtistService`, `SearchService`, etc.).
   - Extract template helpers to `web/render.go` and relocate template parsing.

5. **Slim `internal/server` → `internal/app`**
   - Replace the old `Server` struct with an `app.Container` that wires repo, services, and `web.Server`.
   - `cmd/cli/main.go` moves to `cmd/server/main.go` and calls `app.Run()`.

6. **Move Templates and Static Assets**
   - Relocate `templates/` to `views/templates` and update loader paths.
   - Keep `static` in place; expose a simple `http.FileServer` from `internal/web` (already standard library compliant).

7. **Test & Cleanup**
   - Update package paths in integration/end-to-end tests.
   - Run `go test ./...` ensuring parity.
   - Remove deprecated files and verify docs/diagrams align with new layout.

## Rationale Highlights
- **Smaller Files**: Handlers drop from 500+ LOC to focused per-endpoint files.
- **Explicit Dependencies**: Services receive interfaces rather than reaching into global state, easing testing.
- **Template Isolation**: Rendering concerns stay in one place; adding a new page becomes "create template + handler".
- **Command Separation**: Future CLIs (data refresh, smoke tests) can live under `cmd/` without touching web code.

## Testing Strategy
- Maintain existing table-driven tests while updating imports.
- Add lightweight service tests covering cache usage and filter edge cases after relocation.
- Keep audit/e2e suite intact by reusing the same HTTP surface.

## Risk Mitigation
- Execute steps in order, committing after each major move to keep diffs reviewable.
- Use git move/rename to preserve history, preventing re-review of unchanged logic.
- Document path updates in `README.md` and developer docs after the final step.

## Follow-Up Opportunities
- Introduce an in-memory interface for repository to support future mocks.
- Consider a background refresher (cron) once repo is isolated.
- Add benchmarks in `internal/service` to watch for regressions in filter/search performance.
