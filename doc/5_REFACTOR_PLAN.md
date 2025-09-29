# Refactor Plan — Groupie Tracker

This document summarizes the refactor plan produced by an automated code review. The goal is to simplify code paths, remove redundancy, and apply idiomatic Go + KISS principles while keeping the current architecture intact.

## Summary of findings

- `internal/server/template_data.go` contains template DTOs and formatting helpers that are not referenced elsewhere; remove to reduce maintenance.
- `Server` holds both `httpServer` and an exported `Handler` field; this is redundant. Simplify to a single source of truth for the handler.
- `Repository` stores full `Artist` values in multiple maps leading to memory duplication and extra allocations. Convert to pointer-based storage and introduce an `artistIndex map[int]int` for O(1) adjacency lookups.
- `GetAdjacentArtists` currently scans the artists slice linearly; replace with index-based lookup using the `artistIndex` map.
- Duplicate helpers: `isEmptyFilter` (in `internal/data/search.go`) and `isEmptyArtistFilters` (in `internal/server/handlers.go`) do the same work; centralize one helper.
- `searchCache` eviction strategy is a naive full/half-clear mechanism. Replace with either a simple FIFO eviction or remove caching until a profiling-backed strategy is chosen.
- `repository.go` is large and mixes multiple responsibilities. Split into multiple files: `repository_core.go`, `repository_images.go`, `repository_locations.go`, `repository_helpers.go`.

## Concrete refactor steps

1. Remove `internal/server/template_data.go` and any references.
2. Convert repository maps to store `*Artist` instead of `Artist` values.
   - Update `loadProcessedData`, `createLocations`, and any other code that reads maps accordingly.
   - Introduce `artistIndex map[int]int` mapping artist ID → index in `artists` slice.
3. Implement index-based `GetAdjacentArtists` using `artistIndex` for O(1) lookups.
4. Move utility functions to clearly named files:
   - `repository_helpers.go` — `createSlug`, `normalizeLocation`, `extractYearFromDate`, `extractCountryFromLocation` (make package-private where appropriate)
   - `repository_images.go` — image caching functions
   - `repository_locations.go` — `createLocations` and related logic
5. Consolidate duplicate filter-empty checks into `internal/data/search.go` or a small `internal/data/helpers.go` helper and use it from server code.
6. Simplify server cache logic: either implement a FIFO list for eviction or remove the cache and rely on repository search being fast; add benchmarks to justify later re-introduction.
7. After each step, run `go test ./...` and fix failing tests before moving on.

## Validation & quality gates

- Build: `go build ./...` — must succeed
- Unit tests: `go test ./internal/...` — all tests should pass
- Smoke test: start server locally with `go run ./cmd/cli` and hit `/health`

## Notes and rationale

- Pointer-based maps reduce copies and allocations (Go is more efficient with pointers for shared objects that have slices/maps inside).
- KISS: avoid premature optimizations such as complex LRU caches without profiling.
- Small, focused files are easier to unit test and reason about than a monolithic `repository.go`.

## Next steps

1. Convert small, low-risk items first (remove unused `template_data.go`, consolidate empty-filter helpers).
2. Convert repository to pointer maps in a single commit, run tests, and iterate if necessary.
3. Split repository into multiple files.
4. Re-run full test suite and update documentation.

---

Generated: 2025-09-29
