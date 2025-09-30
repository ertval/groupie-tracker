# GitHub Copilot Refactor Plan for Groupie Tracker

## Snapshot of Current Pain Points
- `internal/data/repository.go` (≈800 LOC) blends API fetching, transformation, caching, filtering, search, and utility helpers; the single struct hides responsibilities and makes targeted testing difficult.
- Domain models, API DTOs, and filter/search parameter types coexist in `models.go`, forcing unrelated concerns into one file and encouraging oversized comments to explain intent.
- `internal/server/handlers.go` (≈540 LOC) mixes routing logic, form parsing, caching, validation, and presentation, creating repeated patterns (`validateExactPath`, cache lookups, filter parsing) that are hard to evolve.
- Template presenter helpers in `internal/server/template_data.go` are currently unused; view formatting lives in handlers instead, leaving dead code and duplicated string formatting.
- Search/filter logic is scattered between repository methods and ad-hoc handler utilities; there is no clear “service layer” for business rules, and cache policy is tightly coupled with HTTP handlers.
- Tests lean on the oversized repository and server structs, so unit scope is coarse; most logic is only covered indirectly via higher-level tests, slowing feedback.

## Guiding Principles
1. **Idiomatic Go boundaries**: small packages with obvious responsibilities, minimal exported surface, dependency injection through interfaces where it improves testability.
2. **KISS + Golden Ratio**: keep a single store and a single service, but ensure each file stays narrow (200–250 LOC max) with cohesive helpers nearby.
3. **Readability over ceremony**: favour flat packages over deep nesting; use files to separate concerns before introducing new directories.
4. **Reduce LOC**: delete unused helpers, collapse repetitive comments, and consolidate duplicate parsing logic without sacrificing clarity.

## Target Package Layout
```
cmd/
  server/main.go        # entry point wires config -> app
internal/
  config/               # stays as-is (runtime knobs)
  api/                  # external API client + raw DTO structs (ArtistDTO, RelationDTO)
    client.go
    dto.go
  model/                # domain structs used by app (Artist, Location, Stats, filters)
    artist.go
    location.go
    filters.go          # shared params/options definitions
  store/                # single Store struct handling load + cached read access
    store.go            # public API (Load, getters, Adjacent)
    loader.go           # fetch + hydrate using api.Client
    artists.go          # artist-specific build helpers
    locations.go        # location aggregation, stats computation
    caching.go          # optional image caching isolated behind interface
  service/              # single Service struct for business rules
    service.go          # constructor wiring store + config
    search.go           # text search + suggestion generation
    filter.go           # artist/location filtering logic
    stats.go            # derived stats helpers (if any)
  web/                  # HTTP layer and presentation
    server.go           # Server struct owns Service + Store references
    router.go           # mux wiring
    middleware.go
    render.go           # template loading/render helpers
    handlers/
      home.go
      artists.go
      locations.go
      search.go
      dev.go
      errors.go
    forms.go            # shared form parsing utilities
    presenters.go       # minimal view-model formatting (only what templates use)
  assets/               # (optional) future image caching helpers

static/ & templates/ remain unchanged.
```
- Packages expose only what consumers need: `store.Store`, `service.Service`, and minimal domain types.
- `cmd/server` simply constructs `store.New(...)`, `service.New(...)`, and `web.NewServer(...)`, then runs `ListenAndServe`.

## Refactor Phases
### Phase 0 – Baseline & Safety Nets
- Capture current behaviour with high-signal tests: snapshot HTTP handler outputs (200/404/500), store loading smoke tests, and a quick coverage baseline.
- Document existing environment expectations in `README`.

### Phase 1 – Data Boundary Cleanup
- Carve API DTOs + client calls out of `internal/data` into `internal/api`.
- Split domain structs into `internal/model` with concise comments; trim verbose doc blocks once responsibilities are clearer.
- Introduce `store.Store` by slimming current repository: keep load + accessor responsibilities only, move filter/search/caching toggles to dedicated files.
- Remove unused presenter helpers or wire them where beneficial; delete anything that remains unused after audit.

### Phase 2 – Service Layer Extraction
- Create `service.Service` owning references to `store.Store` + config flags.
- Move `FilterArtists`, `FilterLocations`, `SearchArtists`, and suggestion logic into the service; expose small interfaces for handlers.
- Centralize normalization helpers (`createSlug`, `normalizeLocation`, year extraction) in private service/store files to avoid duplication.
- Ensure service methods return plain domain types without HTTP concerns, simplifying unit tests.

### Phase 3 – HTTP Layer Slim Down
- Restructure `internal/server` into `internal/web` with sub-files per feature; keep each handler under ~120 LOC.
- Replace `Server` caches with service-level helpers; handlers become thin orchestrators (parse form → call service → render template).
- Consolidate form parsing, validation, and error handling in `forms.go` and `errors.go` to eliminate repeated guard code (`validateExactPath`, `parseFormOrError`).
- Introduce `ServerOptions` config struct for dependency injection (templates dir, cache size) to ease testing.

### Phase 4 – Tests & Tooling
- Write unit tests per new boundary:
  - `api` mocks HTTP client.
  - `store` tests builder helpers with fixture JSON.
  - `service` tests filtering/search edge cases without spinning up HTTP.
  - `web` handler tests use in-memory templates via `httptest`.
- Update integration/E2E tests to reflect new handler wiring.
- Refresh docs: `README`, coverage summary, and any dev guides referencing old structure.

## Cleanup & LOC Reduction Targets
- Delete unused `template_data.go` (or revive only the slices actually consumed by templates) to trim ~200 LOC.
- Collapse three nearly identical country/title-case helpers into one utility.
- Reduce oversized comments once code layout communicates intent; keep concise godocs for exported symbols.
- Evaluate necessity of search query cache; if service-level search is fast enough, remove cache map and related eviction code (~60 LOC) to simplify state.
- Consider moving random-homepage selection to service with deterministic seed for tests, reducing handler branching.

## Risks & Mitigations
- **Risk**: Breaking template expectations when reshaping view models.
  - *Mitigation*: Add template-focused tests (render to string + compare key substrings) before moving data structures.
- **Risk**: API contract drift while splitting DTOs.
  - *Mitigation*: Define explicit fixtures (golden files) for `/artists` and `/relation` responses; run decode tests in `api` package.
- **Risk**: Over-segmentation increasing complexity.
  - *Mitigation*: Stop at the proposed package depth; prefer multiple files within a package before creating new directories.

## Definition of Done
- Store/service/server packages each fit on a single screen per file and have clear, narrow APIs.
- All handlers depend on `service.Service` (no direct data mutation), and form parsing lives in a shared helper.
- Duplication reduced (single slug & country normalization functions), unused code removed, tests pass, documentation updated.
- Overall LOC in `internal/data` + `internal/server` decreases meaningfully while coverage improves or holds steady.
