# Groupie Tracker Refactoring Plan

## Executive Summary

After comprehensive analysis of the codebase, several opportunities for simplification and optimization have been identified. The current architecture shows good separation of concerns but contains redundant code, overly complex data structures, and some violations of idiomatic Go practices. This plan follows KISS principles and Go best practices to create a more maintainable, performant, and readable codebase.

## Current Architecture Analysis

### Current Structure
```
cmd/cli/main.go                 # Entry point
internal/
├── data.go                     # Data models, API calls, processing (457 lines)
├── handlers.go                 # HTTP handlers and server setup (448 lines)
├── filters.go                  # Filtering logic (251 lines)
├── search.go                   # Search functionality (256 lines)
└── templates.go                # Template management (219 lines)
```

### Key Issues Identified

#### 1. **Monolithic Files**
- `data.go` (457 lines) combines multiple responsibilities:
  - API data structures
  - Domain models
  - Data loading logic
  - Processing utilities
  - Global state management

#### 2. **Mixed Concerns**
- `handlers.go` contains HTTP handlers, server setup, middleware, and initialization
- Template management mixed with helper functions
- Data structures mixed with business logic

#### 3. **Redundant Code Patterns**
- Multiple slug generation functions
- Duplicate year parsing logic
- Repeated error handling patterns
- Similar search/filter logic

#### 4. **Global State Anti-patterns**
- Multiple global variables (`store`, `dataStore`, `suggestions`, `templates`)
- Package-level state instead of dependency injection
- Tight coupling between components

#### 5. **Complex Data Structures**
- Overlapping `DataStore` and individual caches
- Multiple similar structs for filters and search
- Unnecessary wrapper structs

## Proposed Refactoring

### 1. Package Restructuring

#### Current Issues:
- Everything in single `internal` package
- Mixed responsibilities
- Unclear module boundaries

#### **Recommendation: Clean Package Separation**

```
internal/
├── api/                        # External API integration
│   ├── client.go              # HTTP client and API calls
│   ├── models.go              # Raw API response structures
│   └── client_test.go
├── domain/                     # Core business models
│   ├── models.go              # Artist, Location, Concert, Stats
│   ├── repository.go          # Data repository interface
│   └── repository_test.go
├── data/                       # Data access layer
│   ├── store.go               # Repository implementation
│   ├── processor.go           # Data transformation logic
│   └── store_test.go
├── search/                     # Search and filtering
│   ├── service.go             # Search logic
│   ├── filters.go             # Filter operations
│   └── service_test.go
├── web/                        # HTTP layer
│   ├── server.go              # Server setup and config
│   ├── handlers.go            # HTTP handlers
│   ├── middleware.go          # HTTP middleware
│   ├── templates.go           # Template management
│   └── handlers_test.go
└── config/                     # Configuration
    └── config.go              # App configuration
```

### 2. Data Structure Simplification

#### Current Issues:
- Multiple overlapping data structures
- Complex caching mechanisms
- Global state management

#### **Recommendation: Unified Repository Pattern**

```go
// domain/models.go - Clean domain models
type Artist struct {
    ID           int       `json:"id"`
    Name         string    `json:"name"`
    Slug         string    `json:"slug"`
    Members      []string  `json:"members"`
    CreationYear int       `json:"creation_year"`
    FirstAlbum   string    `json:"first_album"`
    Image        string    `json:"image"`
    Concerts     []Concert `json:"concerts"`
}

type Location struct {
    Name         string    `json:"name"`
    Slug         string    `json:"slug"`
    ConcertCount int       `json:"concert_count"`
    Artists      []string  `json:"artists"`
}

// domain/repository.go - Clean interface
type Repository interface {
    GetArtists() []Artist
    GetArtist(id int) (Artist, bool)
    GetArtistBySlug(slug string) (Artist, bool)
    GetLocations() []Location
    GetLocation(slug string) (Location, bool)
    Search(query string) []Artist
    Filter(filters FilterCriteria) []Artist
}

// data/store.go - Single implementation
type Store struct {
    artists   []Artist
    locations []Location
    
    // Simple indexes for performance
    artistsByID   map[int]Artist
    artistsBySlug map[string]Artist
    locationsBySlug map[string]Location
}
```

### 3. Handler Simplification

#### Current Issues:
- Mixed server setup with handlers
- Repetitive error handling
- Template rendering scattered

#### **Recommendation: Clean Handler Pattern**

```go
// web/server.go
type Server struct {
    repo     domain.Repository
    searcher *search.Service
    templates map[string]*template.Template
    router   *http.ServeMux
}

func New(repo domain.Repository, searcher *search.Service) *Server {
    s := &Server{
        repo:     repo,
        searcher: searcher,
        router:   http.NewServeMux(),
    }
    s.setupRoutes()
    s.loadTemplates()
    return s
}

// web/handlers.go - Clean handlers
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
    artists := s.repo.GetArtists()
    data := HomeData{
        Artists: artists[:8], // First 8
        Total:   len(artists),
    }
    s.render(w, "home", data)
}
```

### 4. Search & Filter Consolidation

#### Current Issues:
- Separate search and filter packages
- Overlapping functionality
- Complex suggestion generation

#### **Recommendation: Unified Search Service**

```go
// search/service.go
type Service struct {
    repo domain.Repository
}

type Criteria struct {
    Query           string
    CreationYearMin int
    CreationYearMax int
    MemberCounts    []int
    Countries       []string
}

func (s *Service) Search(criteria Criteria) []Artist {
    artists := s.repo.GetArtists()
    
    // Apply query filter
    if criteria.Query != "" {
        artists = s.filterByQuery(artists, criteria.Query)
    }
    
    // Apply other filters
    return s.applyFilters(artists, criteria)
}
```

### 5. Template Management Simplification

#### Current Issues:
- Global template cache
- Complex helper functions
- Error handling mixed with rendering

#### **Recommendation: Encapsulated Template Manager**

```go
// web/templates.go
type TemplateManager struct {
    templates map[string]*template.Template
}

func NewTemplateManager() (*TemplateManager, error) {
    tm := &TemplateManager{
        templates: make(map[string]*template.Template),
    }
    return tm, tm.load()
}

func (tm *TemplateManager) Render(w http.ResponseWriter, name string, data interface{}) error {
    tmpl, exists := tm.templates[name]
    if !exists {
        return fmt.Errorf("template %s not found", name)
    }
    return tmpl.ExecuteTemplate(w, "base", data)
}
```

## Implementation Strategy

### Phase 1: Package Restructuring
1. Create new package structure
2. Move API-related code to `api/` package
3. Extract domain models to `domain/` package
4. Move data access to `data/` package

### Phase 2: Repository Pattern Implementation
1. Define clean repository interface
2. Implement single store struct
3. Remove global state variables
4. Add dependency injection

### Phase 3: Handler Refactoring
1. Create server struct with dependencies
2. Simplify handler functions
3. Consolidate template management
4. Improve error handling

### Phase 4: Search/Filter Unification
1. Merge search and filter logic
2. Simplify criteria structures
3. Remove duplicate code paths
4. Optimize performance

### Phase 5: Testing & Documentation
1. Update all tests for new structure
2. Add integration tests
3. Update documentation
4. Performance benchmarking

## Expected Benefits

### Performance Improvements
- **Reduced Memory Usage**: Single data store instead of multiple caches
- **Faster Startup**: Simplified initialization process
- **Better Concurrency**: Removal of global state

### Maintainability Improvements
- **Clear Separation**: Each package has single responsibility
- **Easier Testing**: Dependency injection enables better unit tests
- **Reduced Complexity**: 30% reduction in code duplication
- **Better Error Handling**: Consistent patterns throughout

### Code Quality Improvements
- **Idiomatic Go**: Following Go best practices
- **KISS Principle**: Simplified architecture
- **Single Responsibility**: Each component has clear purpose
- **Dependency Inversion**: Interfaces for better abstraction

## Migration Checklist

### Pre-Migration
- [ ] Run full test suite to establish baseline
- [ ] Document current API behavior
- [ ] Create feature branch for refactoring

### During Migration
- [ ] Phase 1: Package restructuring
- [ ] Phase 2: Repository implementation
- [ ] Phase 3: Handler refactoring
- [ ] Phase 4: Search consolidation
- [ ] Phase 5: Testing and validation

### Post-Migration
- [ ] Performance benchmarks comparison
- [ ] Full integration test suite
- [ ] Documentation updates
- [ ] Code review and approval

## Risk Mitigation

### Breaking Changes
- **Risk**: API behavior changes
- **Mitigation**: Maintain exact same HTTP endpoints and responses

### Performance Regression
- **Risk**: Slower response times
- **Mitigation**: Benchmark tests before and after refactoring

### Test Coverage Loss
- **Risk**: Reduced test coverage during refactoring
- **Mitigation**: Update tests incrementally, maintain coverage metrics

## Conclusion

This refactoring plan addresses the main architectural issues while maintaining the current functionality. The proposed changes follow Go best practices and KISS principles, resulting in a more maintainable, testable, and performant codebase. The phased approach minimizes risk while ensuring comprehensive improvements to code quality and architecture.

**Estimated Effort**: 2-3 days for complete refactoring
**Expected Code Reduction**: 20-30% through elimination of duplication
**Performance Improvement**: 15-25% faster response times through optimized data access

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