# Phase 3: Search & Suggestion Refactor

## Overview
This phase focuses on removing complex search infrastructure (LRU caches, mutexes) and implementing a simple, direct search using normalized token indexes built during catalog creation.

## Step 3.1: Remove Heavy Search Infrastructure
**Goal:** Eliminate LRU cache, mutexes, and complex bookkeeping.

### Sub-steps:
1. **Identify current search components:**
   - List all search-related types
   - Document current flow
   - Note what to keep vs. remove

2. **Remove LRU cache:**
   - Delete cache implementation
   - Remove cache-related mutexes
   - Simplify to direct search

3. **Remove order bookkeeping:**
   - Delete search result ordering logic
   - Simplify to sorted slice return

4. **Clean up SearchParams and SearchResult types:**
   - Evaluate if still needed
   - Simplify or remove

5. **Run tests:** `go test ./internal/data/...`

## Step 3.2: Build Normalized Token Index
**Goal:** Create simple search index during catalog build.

### Sub-steps:
1. **Create search index types:**
   ```go
   type SearchIndex struct {
       artistTokens   map[int][]string  // artistID -> normalized tokens
       locationTokens map[string][]string // locationSlug -> normalized tokens
   }
   ```

2. **Create token normalization function:**
   ```go
   func normalizeTokens(text string) []string {
       // Lowercase, split, remove special chars
       // Return deduplicated tokens
   }
   ```

3. **Build index during Catalog.Build():**
   ```go
   func (c *Catalog) buildSearchIndex() *SearchIndex {
       index := &SearchIndex{
           artistTokens:   make(map[int][]string),
           locationTokens: make(map[string][]string),
       }

       for _, a := range c.Artists {
           tokens := normalizeTokens(a.Name)
           tokens = append(tokens, normalizeTokens(strings.Join(a.Members, " "))...)
           // Add other searchable fields
           index.artistTokens[a.ID] = tokens
       }

       // Similar for locations

       return index
   }
   ```

4. **Add search index to Catalog:**
   ```go
   type Catalog struct {
       // ... existing fields ...
       searchIndex *SearchIndex
   }
   ```

5. **Run tests:** `go test ./internal/data/...`

## Step 3.3: Implement Direct Search
**Goal:** Simple, direct search using the token index.

### Sub-steps:
1. **Create unified search method:**
   ```go
   func (s *Store) Search(query string) SearchResults {
       normalized := normalizeTokens(query)

       artists := s.searchArtists(normalized)
       locations := s.searchLocations(normalized)

       return SearchResults{
           Artists:   artists,
           Locations: locations,
       }
   }
   ```

2. **Implement token matching:**
   ```go
   func (s *Store) searchArtists(queryTokens []string) []Artist {
       var results []Artist

       for _, artist := range s.catalog.Artists {
           if matchesTokens(s.catalog.searchIndex.artistTokens[artist.ID], queryTokens) {
               results = append(results, artist)
           }
       }

       return results
   }

   func matchesTokens(docTokens, queryTokens []string) bool {
       // Check if any query token matches any doc token
       for _, qt := range queryTokens {
           for _, dt := range docTokens {
               if strings.Contains(dt, qt) {
                   return true
               }
           }
       }
       return false
   }
   ```

3. **Add relevance sorting:**
   ```go
   func sortByRelevance(results []Artist, query string) {
       sort.Slice(results, func(i, j int) bool {
           // Exact name match first
           // Then prefix match
           // Then contains
           // Then alphabetical
       })
   }
   ```

4. **Update handlers:**
   - Replace old search calls with new `Store.Search()`
   - Simplify result handling

5. **Run tests:** `go test ./internal/data/...`

## Step 3.4: Refactor Suggestions
**Goal:** Simplify suggestion generation.

### Sub-steps:
1. **Decide on approach:**
   - **Option A:** Client-side JSON index (recommended for small datasets)
   - **Option B:** Simple server-side endpoint

2. **Option A: Client-Side Implementation:**
   ```go
   // Create endpoint for search index
   func (h *Handlers) handleSearchIndex(w http.ResponseWriter, r *http.Request) {
       index := h.store.GetSearchIndex()
       json.NewEncoder(w).Encode(index)
   }

   // Client-side JavaScript handles filtering/suggestions
   ```

3. **Option B: Server-Side Implementation:**
   ```go
   func (s *Store) GetSuggestions(query string, limit int) []Suggestion {
       // Simple prefix/contains matching
       // Return top N results with formatted text/URL
   }

   type Suggestion struct {
       Text string
       URL  string
       Type string // "artist" or "location"
   }
   ```

4. **Create suggestion formatting helpers:**
   ```go
   func formatArtistSuggestion(a *Artist) Suggestion
   func formatLocationSuggestion(l *Location) Suggestion
   ```

5. **Update web handlers:**
   - Remove complex suggestion infrastructure
   - Use new simple approach

6. **Run tests:** `go test ./internal/web/...`

## Step 3.5: API Cleanup
**Goal:** Remove legacy search types.

### Sub-steps:
1. **Audit usage of SearchParams:**
   - Find all references
   - Plan replacement

2. **Audit usage of SearchResult:**
   - Find all references
   - Plan replacement

3. **Remove or simplify types:**
   - Delete if no longer needed
   - Simplify to basic structs if still useful

4. **Update all callers:**
   - Use new simpler types
   - Update tests

5. **Run tests:** `go test ./...`