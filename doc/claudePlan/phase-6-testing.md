# Phase 6: Testing & Validation

## Overview
This phase focuses on comprehensive testing and validation of all the refactored components, including unit tests, integration tests, edge case testing, and performance benchmarking to ensure the new architecture works correctly and efficiently.

## Step 6.1: Update Data Layer Tests
**Goal:** Test new helper methods and catalog.

### Sub-steps:
1. **Test Artist helper methods:**
   ```go
   func TestArtist_MemberCount(t *testing.T) {
       tests := []struct{
           name     string
           artist   Artist
           expected int
       }{
           {"no members", Artist{}, 0},
           {"one member", Artist{Members: []string{"John"}}, 1},
           {"multiple members", Artist{Members: []string{"John", "Paul"}}, 2},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got := tt.artist.MemberCount()
               if got != tt.expected {
                   t.Errorf("got %d, want %d", got, tt.expected)
               }
           })
       }
   }

   // Similar tests for:
   // - ConcertCount()
   // - FirstAlbumYear()
   // - Countries()
   // - Slug()
   ```

2. **Test Location helper methods:**
   ```go
   func TestLocation_ArtistCount(t *testing.T)
   func TestLocation_TotalConcerts(t *testing.T)
   func TestLocation_YearRange(t *testing.T)
   func TestLocation_Country(t *testing.T)
   ```

3. **Test Catalog:**
   ```go
   func TestCatalog_Build(t *testing.T)
   func TestCatalog_ArtistByID(t *testing.T)
   func TestCatalog_ArtistBySlug(t *testing.T)
   func TestCatalog_LocationBySlug(t *testing.T)
   ```

4. **Run data tests:**
   ```bash
   go test ./internal/data/... -v -cover
   ```

## Step 6.2: Test Filter Framework
**Goal:** Comprehensive filter test coverage.

### Sub-steps:
1. **Test filter builders:**
   ```go
   func TestCreationYearBetween(t *testing.T)
   func TestHasMemberCount(t *testing.T)
   func TestInCountries(t *testing.T)
   func TestFirstAlbumYearBetween(t *testing.T)
   ```

2. **Test range and set helpers:**
   ```go
   func TestIntRange_Contains(t *testing.T)
   func TestIntRange_IsZero(t *testing.T)
   func TestStringSet_Contains(t *testing.T)
   func TestIntSet_Contains(t *testing.T)
   ```

3. **Test filter Match methods:**
   ```go
   func TestArtistFilters_Match(t *testing.T) {
       tests := []struct{
           name    string
           filters ArtistFilters
           artist  Artist
           want    bool
       }{
           {
               name: "empty filters match all",
               filters: ArtistFilters{},
               artist: Artist{ID: 1, Name: "Test"},
               want: true,
           },
           {
               name: "creation year filter",
               filters: ArtistFilters{
                   CreationYear: IntRange{Min: 2000, Max: 2010},
               },
               artist: Artist{ID: 1, CreationDate: 2005},
               want: true,
           },
           // More test cases
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got := tt.filters.Match(&tt.artist)
               if got != tt.want {
                   t.Errorf("got %v, want %v", got, tt.want)
               }
           })
       }
   }
   ```

4. **Test filter composition:**
   ```go
   func TestAndFilters(t *testing.T)
   ```

5. **Run filter tests:**
   ```bash
   go test ./internal/data/ -run TestFilter -v
   ```

## Step 6.3: Test Search and Token Index
**Goal:** Validate search functionality.

### Sub-steps:
1. **Test token normalization:**
   ```go
   func TestNormalizeTokens(t *testing.T) {
       tests := []struct{
           input    string
           expected []string
       }{
           {"The Beatles", []string{"the", "beatles"}},
           {"AC/DC", []string{"ac", "dc"}},
           {"Maroon 5", []string{"maroon", "5"}},
       }

       for _, tt := range tests {
           t.Run(tt.input, func(t *testing.T) {
               got := normalizeTokens(tt.input)
               if !reflect.DeepEqual(got, tt.expected) {
                   t.Errorf("got %v, want %v", got, tt.expected)
               }
           })
       }
   }
   ```

2. **Test token matching:**
   ```go
   func TestMatchesTokens(t *testing.T)
   ```

3. **Test search index building:**
   ```go
   func TestCatalog_BuildSearchIndex(t *testing.T)
   ```

4. **Test search method:**
   ```go
   func TestStore_Search(t *testing.T) {
       // Create test catalog with known artists
       // Search for various queries
       // Assert correct results returned
   }
   ```

5. **Run search tests:**
   ```bash
   go test ./internal/data/ -run TestSearch -v
   ```

## Step 6.4: Update Handler Tests
**Goal:** Test new view models and handler flow.

### Sub-steps:
1. **Test view model creation:**
   ```go
   func TestNewHomePage(t *testing.T)
   func TestNewArtistListPage(t *testing.T)
   func TestNewArtistDetailPage(t *testing.T)
   ```

2. **Test handler helpers:**
   ```go
   func TestRequireMethod(t *testing.T)
   func TestParseFilters(t *testing.T)
   func TestRespondJSON(t *testing.T)
   ```

3. **Test handlers:**
   ```go
   func TestHandleHome(t *testing.T) {
       store := setupTestStore()
       templates := setupTestTemplates()
       handlers := web.NewHandlers(store, templates)

       req := httptest.NewRequest("GET", "/", nil)
       w := httptest.NewRecorder()

       handlers.handleHome(w, req)

       if w.Code != http.StatusOK {
           t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
       }
   }

   // Similar tests for all handlers
   ```

4. **Test middleware:**
   ```go
   func TestMethodOnly(t *testing.T)
   func TestLoggingMiddleware(t *testing.T)
   func TestRecoveryMiddleware(t *testing.T)
   ```

5. **Run web tests:**
   ```bash
   go test ./internal/web/... -v -cover
   ```

## Step 6.5: Edge Case Testing
**Goal:** Cover edge cases and error paths.

### Sub-steps:
1. **Test empty/zero-value inputs:**
   ```go
   func TestArtist_EmptyConcerts(t *testing.T)
   func TestFilters_EmptyFilters(t *testing.T)
   func TestSearch_EmptyQuery(t *testing.T)
   ```

2. **Test not found scenarios:**
   ```go
   func TestStore_ArtistNotFound(t *testing.T)
   func TestStore_LocationNotFound(t *testing.T)
   ```

3. **Test invalid inputs:**
   ```go
   func TestParseFilters_InvalidYear(t *testing.T)
   func TestParseFilters_InvalidMemberCount(t *testing.T)
   ```

4. **Test minimal dataset:**
   ```go
   func TestCatalog_SingleArtist(t *testing.T)
   func TestCatalog_NoLocations(t *testing.T)
   ```

5. **Run all tests:**
   ```bash
   go test ./... -v -cover
   ```

## Step 6.6: Integration and E2E Tests
**Goal:** Test full application flow.

### Sub-steps:
1. **Update integration tests:**
   ```go
   // In tests/integration_test.go

   func TestFullDataLoad(t *testing.T) {
       // Test complete data loading pipeline
   }

   func TestFilterAndSearch(t *testing.T) {
       // Test filter + search combination
   }
   ```

2. **Update E2E tests:**
   ```go
   // In tests/e2e_test.go

   func TestHomePage(t *testing.T)
   func TestArtistList(t *testing.T)
   func TestArtistDetail(t *testing.T)
   func TestSearch(t *testing.T)
   ```

3. **Test complete user flows:**
   - Home → Artists → Detail
   - Home → Search → Results
   - Artists → Filter → Results
   - Artists → Detail → Adjacent

4. **Run integration tests:**
   ```bash
   go test ./tests/... -v
   ```

5. **Check coverage:**
   ```bash
   go test ./... -cover -coverprofile=coverage.out
   go tool cover -html=coverage.out -o coverage.html
   ```

## Step 6.7: Performance Benchmarking
**Goal:** Ensure performance is acceptable.

### Sub-steps:
1. **Create benchmark tests:**
   ```go
   // In internal/data/benchmark_test.go

   func BenchmarkCatalogBuild(b *testing.B) {
       for i := 0; i < b.N; i++ {
           catalog := NewCatalog()
           // Add test data
           catalog.Build()
       }
   }

   func BenchmarkSearch(b *testing.B) {
       store := setupTestStore()
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           store.Search("beatles")
       }
   }

   func BenchmarkFilterArtists(b *testing.B) {
       store := setupTestStore()
       filters := ArtistFilters{
           CreationYear: IntRange{Min: 2000, Max: 2010},
       }
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           store.FilterArtists(filters)
       }
   }
   ```

2. **Run benchmarks:**
   ```bash
   go test ./internal/data/ -bench=. -benchmem
   ```

3. **Document results:**
   ```markdown
   # Performance Benchmarks

   ## Data Loading
   - Catalog build: X ms
   - Full data load: Y ms

   ## Search
   - Simple search: X μs
   - Complex search: Y μs

   ## Filtering
   - Single filter: X μs
   - Multiple filters: Y μs

   All operations are sub-millisecond on typical hardware.
   ```

4. **Set performance baselines:**
   - Document acceptable thresholds
   - Create regression tests if needed

5. **Commit results:**
   ```bash
   git add doc/PERFORMANCE_BENCHMARKS.md
   git commit -m "docs: add performance benchmark results"
   ```