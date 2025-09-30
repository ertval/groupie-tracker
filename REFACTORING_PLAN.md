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