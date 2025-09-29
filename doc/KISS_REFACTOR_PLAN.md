# Groupie Tracker KISS Refactor Plan (2025-09-29)

> Goal: Reduce cognitive load, trim redundancy, remove over-engineered layers, and align with idiomatic Go & KISS while preserving functionality and tests.

## 1. Executive Summary
The codebase is clean and well-documented but *overly verbose* in several layers:
- Service interfaces (`services.go`) simply forward to `data.Repository` with 1:1 pass-through wrappers.
- Excessive comment blocks repeat obvious behavior, increasing maintenance cost.
- Some helper methods can be collapsed (e.g., `findArtistByID` only used once; can build a map once).
- Search + filter logic re-scans slices multiple times (can fuse steps for minor gains while keeping readability).
- Template suggestion generation happens on nearly every request (`NewBaseTemplateData`); could be cached.

Primary simplification strategy: **Flatten indirection**, **prune repetition**, **cache stable computations**, **shrink surface area**, and **prefer direct use of the repository inside handlers**.

Outcome targets:
- ~500–800 LOC reduction (mostly comments + wrappers).
- Fewer allocations & slice scans in hot paths (artists page, search, locations).
- Clearer mental model: `config`, `data`, `web` (rename `server`), `cmd`.

## 2. Current Architecture Assessment
| Concern | Observation | Impact | Priority |
|---------|-------------|--------|----------|
| Service interfaces | 5 interfaces + wrappers delegate directly | Noise, adds maintenance | High |
| Template suggestion regeneration | Regenerated per request (Home, Artists, etc.) | Unnecessary CPU | High |
| Comments verbosity | Many comments restate obvious Go semantics | Cognitive noise | Med |
| Data indexing | Artists + slug + ID maps OK; locations similar; `findArtistByID` linear | Minor inefficiency | Low |
| Filtering logic | Clear but allocates intermediate maps each call | Small cost | Low |
| Search suggestion building | Built once per request path using `GenerateAllSearchSuggestions` (O(N * M)) | Avoidable | High |
| Handler path validation | Manual path checks plus mux pattern overlap | Slight redundancy | Low |
| Range parsing utilities | Good, idiomatic | Keep | - |
| `cacheImages` | Serial download OK; could early-exit when disabled | Already early-exits | - |
| Global stats map | Fine; could wrap struct for type safety | Optional | Low |
| Normalization helpers | Adequate | Keep | - |

## 3. Key Simplification Principles Applied
1. Eliminate accidental abstractions (service facades).
2. Collapse one-time transformations into initialization step.
3. Prefer small, intention-revealing functions over giant comment blocks.
4. Cache deterministic expensive outputs (search suggestions, filter options).
5. Reduce reflection usage unless essential (remove `addSuggestionsToData`).
6. Narrow template func map to only functions actually used.
7. Keep test surface stable (public Repository + handlers) to minimize rewrites.

## 4. Proposed Target Structure
```
cmd/
  cli/main.go
internal/
  config/config.go
  data/
    models.go
    repository.go      (merge filters.go + search.go helpers in cohesive sections)
    filter.go          (optional if size remains large)
  web/                 (rename server -> web for clarity of HTTP layer)
    server.go
    handlers.go
    templates.go       (extract loadTemplates + render)
    middleware.go
    forms.go           (parseArtistFilterParams, parseLocationFilterParams, helpers)
```
Rationale:
- `services.go` removed.
- Split `utils.go` into `templates.go` + `forms.go` to clarify responsibilities.
- Keep repository cohesive; internal segmentation via region comments.
- Optional: Keep `search.go` separate if repository grows > ~800 LOC after merge.

## 5. Detailed Change List
### 5.1 Remove Service Layer
- Delete `internal/server/services.go`.
- Replace usages: in `server.go` store only `repo *data.Repository`.
- Update handlers to call `s.repo.Method()` directly.
- Remove interface fields from `Server` struct.
- Adjust tests to construct server the same way (should still pass; minimal diff).

### 5.2 Cache Stable Artifacts
- Add lazy-init (sync.Once) or eager cache after `LoadData`:
  - `searchSuggestions []data.SearchSuggestion`
  - `artistFilterOptions data.ArtistFilterOptions`
  - `locationFilterOptions data.LocationFilterOptions`
- Expose getters that return cached copies.
- Rebuild only if underlying data ever reloads (currently single load, so static).

### 5.3 Repository Consolidation Improvements
- Inline `findArtistByID` by using existing `artistsByID` map during locations creation (build locations after maps built or create a temporary map first).
- Fuse `processArtists` + `addConcertData` + `transformAPIArtists` into one pass (still readable) OR keep two passes but remove extra function if not reused.
- Simplify `convertCountriesMapToSlice` (inline small loop where used) to reduce jumping.

### 5.4 Reduce Redundant Comments
Rule of thumb: Keep comments for:
- Non-trivial algorithms
- External contract details
- Edge case reasoning
Remove comments that explain obvious struct fields or trivial standard library behavior.

### 5.5 Template & Rendering Simplification
- Move template functions + loading into `templates.go`.
- Remove reflection-based `addSuggestionsToData`: caller passes suggestions explicitly or layout pulls from cached global suggestions if field nil.
- Provide `s.suggestions()` accessor returning cached slice pointer.

### 5.6 Handlers Streamlining
- Remove manual `if r.URL.Path != "/x"` when mux path already precise except root (optional).
- Replace path trimming logic in `ArtistDetail` with checking route base (or keep for simplicity, low risk).
- Pre-sort artists by concert count once if that ordering is preferred for list view, or sort-on-demand with a copy to keep canonical alphabetical slice.

### 5.7 Filter Efficiency (Micro)
- In `FilterArtists`, short-circuit earlier by checking cheap conditions first (member count) before map-building countries.
- Pre-store member count on Artist (already derivable from len; micro gain—skip).
- Reuse `artist.Countries` without rebuilding countries map during filtering: replace per-call concert scan; you already compute `Countries` slice. (Currently filter rebuilds `artistCountries` map inside country filter branch; can just iterate Countries and test membership using a set built from params once.)

### 5.8 Search Efficiency & Simplicity
- In `matchesSearchQuery`: precompute lowercase of query once (already done) but avoid repeated `normalizeSearchQuery` calls on constant artist fields each iteration: store lowercase variant fields at load time (optional) OR just accept current cost (likely fine). For KISS: keep current unless profiling shows need.
- Convert `locationMatches` to simpler direct substring logic + small special cases; current function is acceptable.

### 5.9 Stats Type Safety (Optional)
- Replace `map[string]int` with:
```go
type Stats struct { TotalArtists, TotalMembers, TotalLocations, TotalConcerts, TotalCountries, CachedImages, DownloadedImages int }
```
- Return value semantics clearer, compile-time safety. Keep map only if templates rely on dynamic keys (they appear referenced via known constants—so struct is safe). Low priority; implement in later phase.

## 6. Phased Refactor Plan
### Phase 1 (Safe Structural Cleanup)
1. Remove `services.go` and update `Server` struct & constructor.
2. Add cached suggestions + filter options to `Server` after repository load.
3. Update handlers to use `s.repo` & cached getters.
4. Run tests; ensure no functional change.

### Phase 2 (Repository Simplification)
1. Inline `findArtistByID` by using temporary map at start of `createLocations`.
2. Simplify artist processing (merge small helpers where clarity improves).
3. Optimize country filter using precomputed `artist.Countries`.
4. Add cached `artistFilterOptions` & `locationFilterOptions`.

### Phase 3 (Comment & Docs Pruning)
1. Remove boilerplate comments that duplicate code.
2. Keep concise top-level doc comments per exported type/function.
3. Update README or architecture doc to reflect flattened design.

### Phase 4 (Template System Extraction)
1. Split `utils.go` into `templates.go` & `forms.go` for SRP.
2. Implement suggestion caching & remove reflection function.
3. Ensure all handlers explicitly pass suggestions or base template pulls from cache.

### Phase 5 (Optional Enhancements)
1. Replace stats map with struct.
2. Add basic benchmark (optional) to demonstrate improvement in `GenerateAllSearchSuggestions` (now O(1) retrieval).
3. Add shallow linter script (if allowed) – standard library only means manual script not needed; skip.

## 7. Risk & Mitigation
| Risk | Mitigation |
|------|------------|
| Test failures due to changed handler field assumptions | Keep handler signatures & templates stable; only internal field source changes |
| Cache invalidation if future dynamic reload desired | Encapsulate caches behind getters; if reload added later, reset fields and rebuild |
| Over-pruning comments reduces onboarding clarity | Keep a focused architecture doc + high-level package comment |
| Coupling handlers directly to repository limits future swapping | Repository already concrete; abstraction was not adding value; can reintroduce if needed |

## 8. Success Metrics
| Metric | Baseline | Target |
|--------|----------|--------|
| Lines of code (non-test) | Tally after extraction | -8% to -15% |
| Search suggestions generation per request | O(N) | O(1) cached |
| Artist filtering allocations per call | Map + slice | No map when filtering by countries only using existing slice |
| Time to full server init | ~current | ± same (no regression) |
| Cognitive complexity (subjective) | High comments density | Reduced noise |

## 9. Backward Compatibility
- API surface (HTTP routes, templates, form fields) unchanged.
- No change to environment variables or config semantics.
- Image cache behavior preserved.

## 10. Deferred (Out of Scope Now)
- Data reload / hot refresh
- Streaming or pagination for large artist sets
- Advanced search ranking/scoring
- Distributed cache / CDN for images

## 11. Implementation Order Checklist
- [ ] Phase 1 removal of service layer
- [ ] Phase 1 caching of suggestions & filter options
- [ ] Phase 2 repository simplifications
- [ ] Phase 2 country filter optimization
- [ ] Phase 3 comment pruning
- [ ] Phase 4 template system extraction
- [ ] Phase 5 optional stats struct

## 12. Sample Code Snippets (Illustrative)
### Cached Suggestions (Server)
```go
type Server struct {
  repo *data.Repository
  suggestions []data.SearchSuggestion
  artistFilterOpts data.ArtistFilterOptions
  locationFilterOpts data.LocationFilterOptions
}

func (s *Server) initCaches() {
  s.suggestions = s.repo.GenerateAllSearchSuggestions()
  s.artistFilterOpts = s.repo.GetArtistFilterOptions()
  s.locationFilterOpts = s.repo.GetLocationFilterOptions()
}

func (s *Server) Suggestions() []data.SearchSuggestion { return s.suggestions }
```

### Country Filter Optimization
```go
if len(params.Countries) > 0 {
  allowed := make(map[string]struct{}, len(params.Countries))
  for _, c := range params.Countries { allowed[c] = struct{}{} }
  match := false
  for _, c := range artist.Countries {
    if _, ok := allowed[c]; ok { match = true; break }
  }
  if !match { return false }
}
```

## 13. Go Style Alignment
- Avoid needless interfaces (Effective Go guidance: interfaces as consumers need them).
- Minimize reflection.
- Precompute; prefer data locality.
- Keep exported API small; unexport internals where not needed.

## 14. Rollout Strategy
1. Create a feature branch `refactor/kiss-simplification`.
2. Apply Phase 1 & run all tests (`go test ./...`).
3. Commit after each phase with clear messages.
4. Verify no template breakage via a manual server run.
5. Final PR: include LOC diff and reasoning referencing this plan.

## 15. Conclusion
The refactor focuses on surgical simplification—removing indirection and repeated work while preserving clarity. The system will be leaner, easier to navigate, and cheaper to evolve.

---
Prepared: 2025-09-29
