# Quick Reference: Refactoring Implementation Guide

## TL;DR - The Big Picture

**Goal:** Reduce complexity while improving performance  
**Method:** Eliminate service layer, consolidate files, optimize concurrency  
**Impact:** -23% LOC, +12% startup speed, clearer architecture  
**Risk:** Medium (comprehensive testing required)  
**Effort:** 8-12 hours  

---

## What's Changing?

### 1. Architecture Simplification
```
REMOVE: app package (20 LOC wrapper)
REMOVE: service package (504 LOC facade)
RESULT: 3 layers instead of 5
```

### 2. File Consolidation
```
MERGE: store.go + loader.go + normalize.go → store.go (~800 LOC)
MERGE: home.go + health.go + dev.go → pages.go (100 LOC)
DELETE: handlers.go (empty file)
RESULT: 11 files instead of 24
```

### 3. Concurrency Optimization
```
CHANGE: Fixed 4-worker pool → Adaptive semaphore pattern
CHANGE: CPU-bound sizing → min(tasks, CPUs)
RESULT: 15-20% faster image caching
```

---

## Quick Decision Tree

### Should I merge this file?
```
├─ Is it < 100 LOC with only helper functions?
│  └─ YES → Merge into caller
│  └─ NO → Keep separate
│
├─ Does it only wrap other functions?
│  └─ YES → Eliminate wrapper
│  └─ NO → Keep
│
└─ Is it logically related to another file?
   └─ YES → Consider merging
   └─ NO → Keep separate
```

### Should I optimize this concurrency?
```
├─ Is it I/O-bound work (network, disk)?
│  └─ YES → Use goroutines
│  └─ NO → Avoid concurrency
│
├─ Is the dataset size variable?
│  └─ YES → Use adaptive worker pool
│  └─ NO → Fixed size OK
│
└─ Are there dependencies between tasks?
   └─ YES → Use channels/WaitGroups
   └─ NO → Can run independently
```

---

## Implementation Phases - At a Glance

| Phase | Focus | Risk | Effort | LOC Impact |
|-------|-------|------|--------|------------|
| 1 | Quick wins (consolidate small files) | Low | 1-2h | -100 |
| 2 | Eliminate service layer | Med | 2-3h | -224 |
| 3 | Consolidate data package | Low-Med | 2-3h | -209 |
| 4 | Optimize concurrency | Med | 1-2h | -30 |
| 5 | Final cleanup | Low | 1h | -100 |
| **Total** | | | **8-12h** | **-663** |

---

## Code Patterns - Before & After

### Pattern 1: Service Layer Usage

**BEFORE:**
```go
// internal/web/server.go
type Server struct {
    store *data.Store
    svc   *service.Service  // ← facade layer
}

// internal/web/artists.go
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
    artists := s.svc.FilterArtists(filters)  // ← indirect call
    // ...
}
```

**AFTER:**
```go
// internal/web/server.go
type Server struct {
    store *data.Store  // ← direct access only
}

// internal/web/artists.go
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
    artists := s.store.FilterArtists(filters)  // ← direct call
    // ...
}
```

### Pattern 2: File Organization

**BEFORE:**
```go
// internal/data/store.go (223 LOC)
type Store struct { ... }
func (s *Store) Artists() []Artist { ... }
func (s *Store) ArtistByID(id int) (Artist, bool) { ... }

// internal/data/loader.go (599 LOC)
func (s *Store) processArtists(...) []Artist { ... }
func (s *Store) createLocations(...) []Location { ... }

// internal/data/normalize.go (57 LOC)
func extractCountryFromLocation(loc string) string { ... }
func extractYearFromDate(date string) int { ... }
```

**AFTER:**
```go
// internal/data/store.go (~800 LOC)
type Store struct { ... }

// --- Public API ---
func (s *Store) Artists() []Artist { ... }
func (s *Store) FilterArtists(...) []Artist { ... }
func (s *Store) SearchArtists(...) SearchResult { ... }

// --- Data Loading ---
func (s *Store) Load(ctx) error { ... }
func (s *Store) processArtists(...) []Artist { ... }
func (s *Store) createLocations(...) []Location { ... }

// --- Helpers ---
func extractCountryFromLocation(loc string) string { ... }
func extractYearFromDate(date string) int { ... }
```

### Pattern 3: Concurrency

**BEFORE (Worker Pool):**
```go
type job struct {
    artist   *Artist
    fileName string
}

jobs := make(chan job, len(artists))
numWorkers := 4  // fixed

for w := 0; w < numWorkers; w++ {
    go worker(jobs, &cached, &downloaded)
}

for i := range artists {
    jobs <- job{&artists[i], fileName}
}
close(jobs)
```

**AFTER (Semaphore):**
```go
numWorkers := min(len(artists), runtime.NumCPU())
sem := make(chan struct{}, numWorkers)
var cached, downloaded atomic.Int32

for i := range artists {
    go func(a *Artist) {
        sem <- struct{}{}
        defer func() { <-sem }()
        
        if cacheImage(a) {
            cached.Add(1)
        }
    }(&artists[i])
}
// No close needed, wait handles it
```

---

## Common Pitfalls & Solutions

### Pitfall 1: Breaking Template Dependencies
**Problem:** Moving filter parsing breaks templates  
**Solution:** Keep function signature identical, just change location

```go
// BEFORE: internal/web/templates.go
func parseArtistFilterParams(r *http.Request) data.ArtistFilterParams

// AFTER: internal/web/artists.go
func parseArtistFilterParams(r *http.Request) data.ArtistFilterParams
// ↑ Same signature, templates still work
```

### Pitfall 2: Race Conditions in Concurrency
**Problem:** Shared variable access without synchronization  
**Solution:** Use atomic operations or mutexes

```go
// ❌ WRONG
var count int
go func() { count++ }()  // race!

// ✅ RIGHT
var count atomic.Int32
go func() { count.Add(1) }()  // safe
```

### Pitfall 3: Large File Complexity
**Problem:** 800 LOC store.go is hard to navigate  
**Solution:** Use clear section comments and logical ordering

```go
// internal/data/store.go
package data

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================
type Store struct { ... }

// ============================================================================
// PUBLIC API - Data Access
// ============================================================================
func (s *Store) Artists() []Artist { ... }
func (s *Store) ArtistByID(id int) (Artist, bool) { ... }

// ============================================================================
// PUBLIC API - Business Logic
// ============================================================================
func (s *Store) FilterArtists(params ArtistFilterParams) []Artist { ... }
func (s *Store) SearchArtists(params SearchParams) SearchResult { ... }

// ============================================================================
// DATA LOADING
// ============================================================================
func (s *Store) Load(ctx context.Context) error { ... }

// ============================================================================
// HELPERS - Private
// ============================================================================
func extractYearFromDate(date string) int { ... }
```

---

## Testing Strategy - Quick Commands

### Run tests after each change:
```bash
# All tests
go test ./...

# Specific package
go test ./internal/data

# With coverage
go test -cover ./internal/data

# Race detector (MUST for concurrency changes)
go test -race ./...

# Verbose output
go test -v ./internal/data

# E2E tests
go test ./cmd/server
```

### Benchmarking:
```bash
# Before changes
go test -bench . -benchmem > before.txt

# After changes
go test -bench . -benchmem > after.txt

# Compare
diff before.txt after.txt
```

---

## Validation Checklist - Copy This!

### After Each Phase:
```
□ All unit tests pass
□ No new race conditions (go test -race)
□ Code compiles without errors
□ Commit with descriptive message
```

### Before Merging:
```
□ All unit tests pass (go test ./...)
□ All E2E tests pass (go test ./cmd/server)
□ Race detector clean (go test -race ./...)
□ Manual testing of all pages:
  □ Home page loads
  □ Artists page + filters work
  □ Artist detail page loads
  □ Locations page + filters work
  □ Location detail page loads
  □ Search works
  □ Dev tools accessible
□ Image caching works (test with --cache flag)
□ Error pages display correctly
□ Static files serve correctly
□ Health endpoint responds
□ No console errors in browser
□ Performance maintained or improved
□ Documentation updated
□ README reflects new architecture
```

---

## File Mapping Reference

### What Goes Where After Refactoring

| Current Location | New Location | Reason |
|------------------|--------------|--------|
| `internal/app/app.go` | DELETE | One-liner moved to web.NewServer |
| `internal/service/service.go` | `internal/data/store.go` | Business logic belongs with data |
| `internal/service/filtering.go` | `internal/data/store.go` | Merge into store methods |
| `internal/service/search.go` | `internal/data/store.go` | Merge into store methods |
| `internal/data/loader.go` | `internal/data/store.go` | Same concern, one file |
| `internal/data/normalize.go` | `internal/data/store.go` | Helper functions used by loader |
| `internal/web/handlers.go` | DELETE | Empty file |
| `internal/web/home.go` | `internal/web/pages.go` | Group utility pages |
| `internal/web/health.go` | `internal/web/pages.go` | Group utility pages |
| `internal/web/dev.go` | `internal/web/pages.go` | Group utility pages |

### Tests Migration

| Current Location | New Location |
|------------------|--------------|
| `internal/service/filter_test.go` | `internal/data/filter_test.go` |
| `internal/service/search_test.go` | `internal/data/search_test.go` |

---

## Git Strategy

### Branch Structure
```
main
 └─ refactor/simplify-architecture (feature branch)
     ├─ phase-1-quick-wins (sub-branch)
     ├─ phase-2-service-elimination (sub-branch)
     ├─ phase-3-data-consolidation (sub-branch)
     ├─ phase-4-concurrency (sub-branch)
     └─ phase-5-cleanup (sub-branch)
```

### Commit Messages Template
```
refactor(phase-1): consolidate web handler files

- Merge home.go, health.go, dev.go → pages.go
- Delete empty handlers.go file
- Update routes to use new handler names

Impact: -60 LOC
Tests: All passing
```

---

## Emergency Rollback

If something goes wrong during refactoring:

```bash
# Rollback uncommitted changes
git restore .

# Rollback last commit
git reset --soft HEAD~1

# Rollback to specific commit
git reset --hard <commit-hash>

# Rollback entire phase
git checkout main
git branch -D refactor/simplify-architecture
```

---

## Key Metrics to Track

### Before Refactoring (Baseline)
```bash
# LOC count
find internal -name "*.go" -exec wc -l {} + | tail -1

# Test coverage
go test -cover ./internal/...

# Benchmark
go test -bench . -benchmem > baseline-bench.txt

# Startup time (logged by server)
go run ./cmd/server
```

### After Each Phase
- Repeat above measurements
- Compare to baseline
- Document any regressions

---

## Questions? Check These First

**Q: Should I keep or merge this file?**  
A: If it's < 100 LOC with only helpers → merge. Otherwise → keep separate.

**Q: Where does business logic go now?**  
A: `internal/data/store.go` - it's the single source of business logic.

**Q: How do I handle test failures?**  
A: Fix before proceeding. Never commit failing tests.

**Q: Is 800 LOC in one file too much?**  
A: No, if it's well-organized with clear sections. Use comments liberally.

**Q: What if performance regresses?**  
A: Benchmark before/after. If regression > 10%, investigate or rollback.

**Q: Should I update docs as I go?**  
A: Update in Phase 5 (final cleanup) to avoid churn.

---

## Resources

- **Main Plan:** `GitHub-Copilot-Concurrency-Simplification-Refactoring-Plan.md`
- **Analysis:** `REFACTORING_ANALYSIS_SUMMARY.md`
- **Architecture:** `ARCHITECTURE_TRANSFORMATION_DIAGRAM.md`
- **This Guide:** `QUICK_REFERENCE_GUIDE.md`

---

## Ready to Start?

1. **Create feature branch:** `git checkout -b refactor/simplify-architecture`
2. **Start with Phase 1:** Low-risk, quick wins
3. **Test after each change:** `go test ./...`
4. **Commit frequently:** Small, logical commits
5. **Stay focused:** One phase at a time

**Good luck! 🚀**
