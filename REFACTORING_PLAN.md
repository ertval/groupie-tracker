# Groupie Tracker Refactoring Plan

## Executive Summary

After analyzing the codebase, I've identified significant opportunities to simplify the architecture while maintaining functionality. The current implementation shows over-engineering patterns that violate KISS principles and Go idioms. This plan focuses on reducing complexity, eliminating redundancy, and improving maintainability.

## Current Issues Analysis

### 1. Over-Engineered Data Layer (⚠️ HIGH PRIORITY)

**Problem**: Repository pattern with excessive abstraction
- Single repository managing ALL operations (violates SRP)
- 785-line repository.go file with mixed responsibilities  
- Complex caching mechanisms with limited benefit
- Unnecessary service layer patterns for a simple web app

**Example**: Repository manages API fetching, image caching, slug generation, navigation, filtering, and search

### 2. Template Data Structure Duplication (⚠️ HIGH PRIORITY)

**Problem**: Parallel template-specific structs
- `template_data.go` (270 lines) duplicates domain models
- `TemplateArtist`, `TemplateLocation` mirror `Artist`, `Location`
- Pre-formatting logic belongs in templates, not Go code
- Violates DRY principle across 15+ duplicate fields

**Example**: Both `Artist.Countries []string` and `TemplateArtist.CountriesText string` exist

### 3. Excessive Configuration Complexity (🔸 MEDIUM PRIORITY)

**Problem**: Over-abstracted configuration
- Global config variables make testing harder
- Configuration scattered across multiple patterns
- Simple web app doesn't need enterprise-level config management

### 4. Inefficient Caching Strategy (🔸 MEDIUM PRIORITY)

**Problem**: Complex caching implementation with poor cache-to-benefit ratio
- Over-engineered search query cache with manual eviction logic
- Image caching adds complexity for external URL performance gains
- Pre-computed navigation links using excessive memory
- Filter options cached despite being static data that's cheap to compute

### 5. Package Structure Over-Engineering (🔸 MEDIUM PRIORITY)

**Problem**: Unnecessary package separation
- `internal/server` split into 6 files for simple HTTP handlers
- `cmd/cli` and `cmd/testapi` for basic main functions
- Template data helpers scattered across multiple files

## Refactoring Strategy: KISS Principles Applied

### Phase 1: Simplify Data Layer (Week 1)

#### 1.1 Eliminate Repository Pattern
```go
// BEFORE: Over-engineered repository
type Repository struct {
    // 15+ fields managing everything
    artists []Artist
    artistsByID map[int]Artist
    // ... excessive abstractions
}

// AFTER: Simple data store  
type DataStore struct {
    Artists   []Artist   `json:"artists"`
    Locations []Location `json:"locations"`
}
```

**Actions**:
- [ ] Replace `Repository` with simple `DataStore` struct
- [ ] Move API fetching to dedicated `LoadData(ctx) (*DataStore, error)` function
- [ ] Keep strategic caching: templates and search suggestions only
- [ ] Remove complex caching (query cache, navigation links)
- [ ] Remove service layer patterns

**Files to refactor**:
- `internal/data/repository.go` → `internal/data/store.go` (reduce from 785 to ~200 lines)
- Keep `internal/data/filters.go` as separate `filters.go` (~150 lines)
- Keep `internal/data/search.go` as separate `search.go` (~100 lines)
- Simplify `internal/data/models.go` (reduce to ~100 lines)

#### 1.2 Flatten Package Structure
```go
// BEFORE: Over-abstracted packages
internal/
├── config/config.go
├── data/
│   ├── repository.go (785 lines)
│   ├── models.go (283 lines)  
│   ├── filters.go (467 lines)
│   ├── search.go
│   └── *_test.go
└── server/
    ├── server.go
    ├── handlers.go (542 lines)
    ├── routes.go
    ├── middleware.go
    ├── utils.go (316 lines)
    └── template_data.go (270 lines)

// AFTER: Simplified structure
internal/
├── data.go        (~200 lines: models + API loading)
├── filters.go     (~150 lines: filtering logic)
├── search.go      (~100 lines: search functionality)
├── handlers.go    (~350 lines: HTTP handlers)
└── templates.go   (~100 lines: template utilities)
```

**Actions**:
- [ ] Keep data operations separate: `data.go`, `filters.go`, `search.go`
- [ ] Merge `internal/server/*` into `handlers.go`, `templates.go`
- [ ] Move config constants to main package or handlers
- [ ] Consolidate all template logic

### Phase 2: Eliminate Template Duplication (Week 1)

#### 2.1 Remove Template-Specific Structs
```go
// BEFORE: Parallel template structs (270 lines duplication)
type TemplateArtist struct {
    Name string
    // ... 15 duplicate fields
    MemberCountText  string  // "4 members" 
    CountriesText    string  // "USA, UK"
    ConcertCountText string  // "12 concerts"
}

// AFTER: Use domain models + template functions
// Template: {{.Artist.Members | len}} member{{if ne (len .Artist.Members) 1}}s{{end}}
```

**Actions**:
- [ ] Delete entire `template_data.go` file
- [ ] Use existing template functions: `{{join .Countries ", "}}`
- [ ] Add missing template helpers: `pluralize`, `len`
- [ ] Pass domain models directly to templates

#### 2.2 Simplify Template Functions
```go
// BEFORE: Complex custom formatting in Go
func toTitleCase(s string) string { /* 15+ lines */ }
func formatMemberCount(count int) string { /* custom logic */ }

// AFTER: Simple template functions
funcMap := template.FuncMap{
    "join":      strings.Join,
    "len":       func(s interface{}) int { /* reflection */ },
    "pluralize": func(count int, singular, plural string) string {
        if count == 1 { return singular }
        return plural
    },
}
```

### Phase 3: Simplify Server Architecture (Week 2)

#### 3.1 Consolidate Server Package
```go
// BEFORE: 6 files in server package
server/
├── server.go (155 lines)
├── handlers.go (542 lines) 
├── routes.go
├── middleware.go
├── utils.go (316 lines)
└── template_data.go (270 lines)

// AFTER: 2 files maximum
├── handlers.go (~350 lines total)
└── templates.go (~100 lines)
```

**Actions**:
- [ ] Merge all handlers into single file (they're simple CRUD)
- [ ] Move template loading to `templates.go`
- [ ] Eliminate server struct dependency injection pattern
- [ ] Use package-level variables for strategic caching only

#### 3.2 Simplify HTTP Layer with Strategic Caching
```go
// BEFORE: Complex dependency injection
type Server struct {
    repo *data.Repository
    templates map[string]*template.Template
    suggestions []data.SearchSuggestion
    // ... 10+ cached fields
}

// AFTER: Package-level simplicity with strategic caching
var (
    dataStore   *DataStore
    templates   map[string]*template.Template  // Cache: templates (expensive to compile)
    suggestions []SearchSuggestion              // Cache: search suggestions (expensive to generate)
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
    artists := dataStore.Artists
    // ... simple handler logic using cached suggestions
}
```

### Phase 4: Optimize Caching Strategy (Week 2)

#### 4.1 Implement Strategic Caching Only
```go
// REMOVE: Over-engineered caching
s.searchCache = make(map[string][]data.Artist)         // Query result cache (poor hit rate)
s.artistFilterOpts = s.repo.GetArtistFilterOptions()  // Static data cache (cheap to compute)
s.navigationCache = ...                                // Pre-computed navigation (memory waste)

// KEEP: Strategic caching (high value)
var (
    templates   map[string]*template.Template  // Templates: expensive to compile
    suggestions []SearchSuggestion             // Search suggestions: expensive to generate
)

// REPLACE: Compute cheap operations on-demand
func SearchHandler(w http.ResponseWriter, r *http.Request) {
    // Use cached suggestions for autocomplete
    allSuggestions := suggestions
    
    // Compute filter options on-demand (small dataset, simple computation)
    filterOptions := computeFilterOptions(dataStore.Artists)
    
    results := search.FindArtists(query, dataStore.Artists)  // Direct search, no caching
}
```

**Actions**:
- [ ] Keep only high-value caching: templates and search suggestions
- [ ] Remove query result caching (poor hit rate for small dataset)
- [ ] Remove image caching complexity (use CDN approach instead)
- [ ] Compute filter options on-demand (cheap operation)
- [ ] Remove navigation link pre-computation

#### 4.2 Maintain Separate Filter and Search Logic
```go
// BEFORE: Complex filter structures with pointers
type ArtistFilterParams struct {
    CreationYearFrom *int `json:"creationYearFrom,omitempty"`
    CreationYearTo   *int `json:"creationYearTo,omitempty"`
    // ... complex pointer logic
}

// AFTER: Simple struct with zero values (keep as separate functionality)
// filters.go
type Filters struct {
    CreationYearMin int      `form:"creation_year_min"`
    CreationYearMax int      `form:"creation_year_max"`
    Countries       []string `form:"countries"`
    MemberCounts    []int    `form:"member_counts"`
}

func FilterArtists(artists []Artist, filters Filters) []Artist {
    // Simple, focused filtering logic
}

// search.go  
type SearchParams struct {
    Query   string  `form:"q"`
    Filters Filters `form:"filters"`
}

func FindArtists(params SearchParams, artists []Artist) []Artist {
    // Simple, focused search logic
}
```

**Actions**:
- [ ] Keep filtering and search as separate, focused modules
- [ ] Simplify filter parameter structures (remove pointer complexity)
- [ ] Maintain clear separation of concerns between filter and search
- [ ] Use simple, testable functions instead of complex method chains

## Simplified File Structure (Target)

```
groupie-tracker/
├── cmd/
│   └── main.go                 (~30 lines)
├── internal/
│   ├── data.go                 (~200 lines: models + API loading)
│   ├── filters.go              (~150 lines: filtering logic)
│   ├── search.go               (~100 lines: search functionality)
│   ├── handlers.go             (~350 lines: all HTTP handlers)
│   └── templates.go            (~100 lines: template utilities)
├── templates/                  (unchanged)
├── static/                     (unchanged)
└── tests/                      (simplified)
```

**Total reduction**: From ~2,400 lines across 15+ files to ~930 lines across 6 files

## Implementation Guidelines

### Do's ✅
- Use standard library patterns over custom abstractions
- Keep handlers simple with minimal business logic
- Use template functions for view logic
- Pass domain models directly to templates
- Follow Go naming conventions (`handler` not `Handler`)
- Keep strategic caching for expensive operations (templates, search suggestions)
- Maintain separation of concerns (data, filters, search, handlers)
- Use simple structs with zero values instead of pointer complexity

### Don'ts ❌
- Don't create service layers for simple CRUD operations
- Don't cache everything - only cache expensive operations with proven benefit
- Don't duplicate data in template-specific structs
- Don't use dependency injection for simple web apps
- Don't split packages unless functionally distinct (data vs filters vs search)
- Don't use complex pointer structures when zero values work fine

## Testing Strategy

1. **Maintain audit compliance**: Ensure Queen=7 members, Gorillaz="26-03-2001"
2. **Regression testing**: Keep existing functionality working
3. **Simplify test structure**: Remove complex test setup patterns

## Migration Path

### Week 1: Data Layer
1. Create new simplified `data.go` with strategic caching
2. Port existing functionality to simpler structures
3. Remove repository pattern abstractions
4. Keep `filters.go` and `search.go` as focused modules
5. Update all imports

### Week 2: Presentation Layer  
1. Eliminate template duplication
2. Consolidate server package (keep data operations separate)
3. Implement strategic caching (templates + search suggestions only)
4. Simplify configuration

### Week 3: Polish & Test
1. Comprehensive testing
2. Performance validation  
3. Documentation updates
4. Final cleanup

## Expected Benefits

- **Maintainability**: 60% fewer lines of code to maintain while keeping logical separation
- **Readability**: Clear, idiomatic Go without over-abstractions  
- **Performance**: Strategic caching for expensive operations, simpler code paths elsewhere
- **Testing**: Easier to test with focused, single-responsibility modules
- **Onboarding**: New developers can understand the codebase quickly
- **Separation of Concerns**: Clear boundaries between data, filtering, search, and presentation

## Risk Mitigation

- Implement changes incrementally with tests
- Maintain API compatibility during refactoring
- Keep original complex implementation in branch until validated
- Monitor performance metrics to ensure no regressions

---

*This refactoring plan prioritizes simplicity, maintainability, and Go idioms while retaining strategic performance optimizations and maintaining clear separation between filtering, search, and core data operations.*