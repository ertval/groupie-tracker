# Phase 5: Code Polish & Documentation

## Overview
This phase focuses on cleaning up the codebase by removing redundant comments, normalizing naming conventions, reviewing package responsibilities, updating documentation, and ensuring code quality standards.

## Step 5.1: Remove Redundant Comments
**Goal:** Keep only rationale-level comments.

### Sub-steps:
1. **Audit all comments:**
   ```bash
   grep -r "// " internal/
   ```

2. **Remove obvious comments:**
   ```go
   // BAD: Remove these
   // Get artist by ID
   func (s *Store) GetArtist(id int) (*Artist, error)

   // GOOD: Keep these
   // artistsByCountry builds a reverse index from country to artists.
   // This is precomputed to avoid O(n) scans on each filter operation.
   func artistsByCountry(artists []Artist) map[string][]int
   ```

3. **Keep comments for:**
   - Non-obvious design decisions
   - Performance considerations
   - Edge cases and gotchas
   - Public API documentation

4. **Update package-level docs:**
   ```go
   // Package data provides the core domain models and data access layer
   // for the Groupie Tracker application.
   package data
   ```

5. **Run `gofmt`:** `gofmt -w .`

## Step 5.2: Normalize Naming Conventions
**Goal:** Consistent, idiomatic Go names.

### Sub-steps:
1. **Remove Get prefixes:**
   ```go
   // Before
   GetArtist()
   GetArtists()
   GetLocation()

   // After
   Artist()           // Single item by ID
   ArtistByID()       // Explicit lookup
   ArtistBySlug()     // Alternative lookup
   Artists()          // Multiple items
   AllArtists()       // All items
   ```

2. **Standardize method names:**
   - Use `ByID`, `BySlug` for lookups
   - Use `All` prefix for full collections
   - Use `Filter` prefix for filtered collections
   - Use `Search` for search operations

3. **Review and update:**
   ```bash
   # Find all Get* methods
   grep -r "func.*Get[A-Z]" internal/

   # Update each one
   ```

4. **Update all callers:**
   - Use find/replace carefully
   - Run tests after each batch

5. **Run tests:** `go test ./...`

## Step 5.3: Package Responsibility Review
**Goal:** Ensure each package has clear, focused responsibility.

### Sub-steps:
1. **Review package structure:**
   ```
   internal/
   ├── api/       - External API client
   ├── data/      - Domain models, catalog, store, filters
   ├── view/      - View models for templates
   └── web/       - HTTP handlers, middleware, routing
   ```

2. **Verify package independence:**
   - `api` should not import `web` or `view`
   - `data` should not import `web` or `view`
   - `view` can import `data`
   - `web` can import `data` and `view`

3. **Move misplaced code:**
   - Identify code in wrong package
   - Move to appropriate location
   - Update imports

4. **Check for circular dependencies:**
   ```bash
   go list -f '{{ join .Deps "\n" }}' ./...
   ```

5. **Run tests:** `go test ./...`

## Step 5.4: Update Documentation
**Goal:** Reflect new architecture in docs.

### Sub-steps:
1. **Update README.md:**
   - Project structure section
   - Architecture overview
   - Build/run instructions

2. **Create/update ARCHITECTURE.md:**
   ```markdown
   # Architecture

   ## Overview
   This application follows a clean, idiomatic Go architecture...

   ## Package Structure
   - `internal/api`: External API client
   - `internal/data`: Core domain and data access
   - `internal/view`: View models
   - `internal/web`: HTTP layer

   ## Data Flow
   1. API client fetches raw data
   2. Catalog normalizes and indexes
   3. Store provides filtered/searched access
   4. Handlers build view models
   5. Templates render HTML
   ```

3. **Create migration guide:**
   ```markdown
   # Migration Guide

   ## Changed APIs
   - `store.GetArtist()` → `store.Artist()`
   - `artist.ConcertCount` → `artist.ConcertCount()`
   ...
   ```

4. **Update doc/ folder:**
   - Add summary of refactoring
   - Link to this implementation plan
   - Archive old plans

5. **Review and commit:**
   ```bash
   git add doc/
   git commit -m "docs: update architecture documentation"
   ```

## Step 5.5: Code Quality Check
**Goal:** Ensure code quality standards.

### Sub-steps:
1. **Run gofmt:**
   ```bash
   gofmt -w .
   ```

2. **Run go vet:**
   ```bash
   go vet ./...
   ```

3. **Run golint (if available):**
   ```bash
   golint ./...
   ```

4. **Check for common issues:**
   - Unused variables
   - Unused imports
   - Shadowed variables
   - Error handling

5. **Run all tests:** `go test ./... -v`