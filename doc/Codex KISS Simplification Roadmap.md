# GitHub Copilot KISS Simplification Roadmap

## Guiding principles
- Prefer straight-line, readable logic over concurrency where the dataset is small (≈50 artists) and latency is dominated by the third-party API.
- Keep exported types lean; promote helper functions or methods that derive computed data instead of caching redundant fields.
- Normalize on idiomatic Go defaults: zero values as "unset", value types where possible, and small packages with clear responsibility boundaries.
- Push normalization work to the load phase, so handlers simply read already-clean data without re-parsing strings.

## Phase 1 – Simplify the data model and loading pipeline
1. **Collapse redundant artist fields**
   - Replace `Artist.DatesAtLocation`, `ConcertCount`, `Countries`, and `MemberCount` with helper methods that derive from a single `[]Concert` slice.
   - Represent concerts with parsed dates (`time.Time`) and a typed `LocationSlug` to avoid repeated string munging.
   - Store `FirstAlbum` alongside a parsed `FirstAlbumYear` struct returned by helper methods rather than keeping duplicate integer caches.
2. **Streamline location aggregation**
   - Generate `Location` data directly from the normalized concert list using small helper functions.
   - Retain a single source of truth for artist aggregates (no parallel maps of counts); compute display data (`ArtistCount`, `TotalConcerts`, year ranges) in one pass.
3. **Refactor `Store` to smaller collaborators**
   - Introduce a lightweight `Catalog` struct that owns the slices/maps and exposes accessors; make `Store` a façade wiring together the catalog, filters, and search index.
   - Convert the current goroutine-heavy `loadData` into sequential steps with clear helper functions: `fetch`, `normalize`, `buildCatalog`, `buildViews`.
   - Keep image caching optional behind a simple interface so the main load path stays linear.
4. **Tighten type usage and visibility**
   - Use unexported helpers for slug generation and country extraction; only export what the web layer actually needs.
   - Replace package-level regex creation with a `sync.Once` or precompiled variable scoped near use to avoid re-compilation but keep code readable.

## Phase 2 – Rework filter types and evaluation
1. **Define explicit range and set helpers**
   - Replace pointer-based range parameters with an idiomatic struct: `type IntRange struct { From, To int }` where zero values mean "open".
   - Wrap multi-select filters in a small `StringSet`/`IntSet` helper to remove per-request map allocations.
2. **Move filter logic onto methods**
   - Attach `Match` methods to the filter structs (`ArtistFilters.Match(artist)` and `LocationFilters.Match(location)`) to encapsulate the AND logic and remove free functions.
   - Expose a single `FilterArtists(filters ArtistFilters)` that iterates once and returns results without extra allocations or resorting.
3. **Precompute filter metadata from normalized data**
   - Rebuild `ArtistFilterOptions` and `LocationFilterOptions` using the simplified model, ensuring min/max computations come from the same pass that builds the catalog.
   - Keep options generation in a separate helper so tests can target it directly.

## Phase 3 – Simplify search and suggestion handling
1. **Drop the bespoke LRU cache**
   - Remove the `searchCache`/`searchOrder` bookkeeping; the dataset is tiny, so a direct scan keeps code simpler and predictable.
   - If caching is still desirable, wrap it behind an interface and consider `sync.Map` with capped size or a standard library-friendly alternative.
2. **Build a dedicated search index**
   - Precompute a `map[ArtistID]Tokens` with normalized name/member/location strings at load time.
   - Implement search as a simple token match over this index, reusing the filter `Match` helpers for the post-filter step.
3. **Unify suggestion generation**
   - Represent suggestions with a small struct that stores both display text and normalized tokens to avoid recomputing inside `filterSearchSuggestions`.
   - Deduplicate suggestion text formatting ("name - type") into a helper, and ensure URLs are generated in one place.

## Phase 4 – Trim HTTP layer verbosity
1. **Introduce shared view models**
   - Create a `view` package or unexported structs that bundle common template data (title, extra assets, suggestions) to avoid redeclaring anonymous structs in each handler.
   - Centralize suggestion and stats retrieval in a helper, so handlers focus on request-specific data.
2. **Reduce form parsing repetition**
   - Keep `parseArtistFilterParams`/`parseLocationFilterParams`, but rewrite them to use the new filter structs directly and return `ArtistFilters`/`LocationFilters` without intermediate copies.
   - Provide small helpers for request validation (`requireMethod`, `requirePath`) so the handlers read like high-level scripts.
3. **Clean up templating utilities**
   - Collapse template lookup errors into a single path; ensure template compilation happens once during `NewApp` with minimal logging.
   - Consider extracting `render` into a tiny adapter that always writes common headers, reducing branching inside the helper.
4. **Clarify routing and middleware**
   - Replace the method restriction logic with a reusable middleware that sets the `Allow` header and responds early, minimizing inline duplication.
   - Document the middleware order near its definition and keep the chain straight-line.

## Phase 5 – Testing and rollout
1. **Update unit tests to the new model**
   - Adjust `internal/data` tests to use the simplified structs and helper methods (e.g., call `artist.ConcertCount()` instead of accessing a field).
   - Add focused tests for the new `Match` methods and search index construction.
2. **Add lightweight integration coverage**
   - Extend existing web handler tests to assert that view models still populate correctly after the data layer slimming.
   - Include regression tests for edge cases such as artists without concerts, empty filters, and unknown slugs.
3. **Migration checkpoints**
   - Land the data model changes first (Phase 1), ensure all tests pass, then proceed to filters (Phase 2) and search (Phase 3).
   - Defer HTTP polish (Phase 4) until data/search layers stabilize to minimize churn in the templates.

## Success criteria
- Data structures expose a single source of truth with helper methods for derived fields; no duplicated caches or parallel maps.
- Search and filter flows become short, easily readable functions without bespoke caching layers or repeated allocations.
- HTTP handlers rely on small, composable helpers so each route reads as a concise script aligned with the KISS principle.
- Existing automated tests remain green, and new tests cover the streamlined helpers to guard against regressions.
