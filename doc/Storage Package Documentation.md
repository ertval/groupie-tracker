# Storage Package Documentation

## Overview

# Storage Package Refactoring

This document describes the refactoring of the storage package, which has been split into two main components: base storage operations and advanced data manipulation services.

## Architecture Overview

The storage package now consists of three main files:

### 1. `base_store.go` - Core Storage & Cache Operations
- **BaseStore**: Thread-safe in-memory storage with cache functionality
- **Cache Management**: Periodic updates from external API
- **CRUD Operations**: Basic create, read, update, delete operations
- **Data Loading**: Bulk data loading and derived data computation

### 2. `service.go` - Data Manipulation & Business Logic
- **Service**: Advanced data operations on top of BaseStore
- **Search Operations**: Text-based search across artists and members
- **Filtering**: Year-based, member count, and location filtering
- **Sorting**: Multiple sorting strategies (name, year, member count)
- **Analytics**: Popular locations and detailed statistics

### 3. `store.go` - Unified Interface
- **Store**: Combines BaseStore and Service using composition
- **Backward Compatibility**: Maintains existing API for handlers
- **Enhanced Functionality**: Adds consistent sorting to existing methods

## Key Benefits

### Separation of Concerns
- **Base operations** are isolated from **business logic**
- **Cache management** is separate from **data manipulation**
- **Thread safety** is handled at the base level
- **Service operations** can focus on algorithms and logic

### Testability
- Each component can be tested independently
- Mock interfaces for testing service layer without storage
- Cleaner test structure with focused responsibilities

### Maintainability
- Easier to add new search/filter operations
- Cache logic changes don't affect business operations
- Clear boundaries between different types of operations

### Performance
- Service operations work on copies to prevent data races
- Immutable results prevent accidental modifications
- Consistent sorting improves user experience

## Interface Definitions

### DataReader Interface
```go
type DataReader interface {
    GetAllArtists() []models.Artist
    GetArtist(id int) (models.Artist, bool)
    GetAllLocations() []models.Location
    // ... other read operations
}
```

### APIClient Interface (unchanged)
```go
type APIClient interface {
    FetchAllData(ctx context.Context) (*models.APIResponse, error)
}
```

## Usage Examples

### Basic Usage (Backward Compatible)
```go
// Existing code continues to work
store := storage.NewStore()
artists := store.GetAllArtists()
results := store.SearchArtists("Queen")
```

### Advanced Service Operations
```go
// New functionality through service layer
store := storage.NewStore()

// Advanced filtering
artists := store.Service.FilterArtistsByMemberCount(4, true)
sorted := store.Service.SortArtistsByYear(artists)

// Analytics
popular := store.Service.GetMostPopularLocations(5)
stats := store.Service.GetDetailedStats()
```

### Cache Operations (Unchanged)
```go
// Cache functionality remains the same
store := storage.NewStoreWithCache(apiClient)
store.StartCache(ctx)
defer store.StopCache()
```

## Migration Notes

### For Handlers
- **No changes required** - existing handler code continues to work
- **Enhanced sorting** - `GetAllArtists()` now returns alphabetically sorted results
- **New features available** - can access advanced operations via `store.Service`

### For Tests
- **Existing tests** continue to work with minimal changes
- **New test files** added for service layer testing
- **Mock interfaces** available for isolated testing

## New Features Added

### Advanced Filtering
- `FilterArtistsByMemberCount(count int, exact bool)` - Filter by band size
- `SearchArtistsByLocation(query string)` - Find artists by performance locations

### Enhanced Sorting
- `SortArtistsByName(artists)` - Alphabetical sorting
- `SortArtistsByYear(artists)` - Chronological sorting  
- `SortArtistsByMemberCount(artists)` - Sort by band size

### Analytics & Statistics
- `GetMostPopularLocations(limit int)` - Most frequent concert venues
- `GetDetailedStats()` - Comprehensive data analysis

### Data Integrity
- All service methods return **copies** of data
- **Immutable results** prevent accidental modifications
- **Thread-safe operations** at all levels

## Future Extensibility

The new architecture makes it easy to add:
- **New search algorithms** (fuzzy matching, phonetic search)
- **Additional filtering criteria** (genre, album count, etc.)
- **Complex analytics** (tour patterns, collaboration networks)
- **Caching strategies** for expensive operations
- **Data export/import** functionality

## Testing Strategy

### Base Store Tests (`store_test.go`)
- Cache functionality and thread safety
- Basic CRUD operations
- Data loading and integrity
- Backward compatibility verification

### Service Tests (`service_test.go`)
- Search and filter algorithms
- Sorting consistency
- Analytics accuracy
- Result immutability

This refactoring maintains full backward compatibility while providing a cleaner, more maintainable architecture for future development. It has been refactored to include production-ready features like periodic data updates, concurrent access management, and efficient derived data computation.

## Key Features

### 🔄 Automatic Cache Updates
- **Periodic Updates**: Automatically fetches fresh data from the API every 30 seconds
- **Background Processing**: Updates run in separate goroutines without blocking operations
- **Error Resilience**: Continues operating even when API calls fail
- **Graceful Shutdown**: Properly handles context cancellation and cleanup

### 🔒 Thread-Safe Operations
- **Concurrent Access**: Safe for multiple goroutines to read/write simultaneously
- **Read-Write Locks**: Optimized for high-read, low-write scenarios
- **Atomic Operations**: Cache state management using atomic booleans
- **Data Consistency**: Guaranteed consistent state during updates

### ⚡ Performance Optimizations
- **Derived Data Caching**: Pre-computed unique locations and dates
- **Efficient Memory Usage**: Optimized data structures and copying
- **Minimal Lock Contention**: Separate locks for cache management and data access
- **Fast Lookups**: Hash map-based storage for O(1) access times

## Architecture

### Core Components

#### Store
```go
type Store struct {
    // Data storage
    artists   map[int]models.Artist
    locations map[int]models.Location
    dates     map[int]models.Date
    relations map[int]models.Relation
    
    // Cache management
    apiClient       APIClient
    cacheRunning    atomic.Bool
    stopCache       chan struct{}
    updateInterval  time.Duration
    
    // Pre-computed data
    uniqueLocations []string
    uniqueDates     []string
}
```

#### APIClient Interface
```go
type APIClient interface {
    FetchAllData(ctx context.Context) (*APIData, error)
}
```

### Cache Lifecycle

1. **Initialization**: `NewStoreWithCache(apiClient)`
2. **Start**: `StartCache(ctx)` begins periodic updates
3. **Operation**: Background goroutine updates data every 30 seconds
4. **Shutdown**: `StopCache()` or context cancellation stops updates

## Usage Examples

### Basic Usage
```go
// Create store with cache
store := storage.NewStoreWithCache(apiClient)

// Start automatic updates
ctx := context.Background()
store.StartCache(ctx)

// Use store normally
artists := store.GetAllArtists()
locations := store.GetUniqueLocations()

// Stop cache when done
store.StopCache()
```

### Production Server Setup
```go
func setupServer() {
    apiClient := api.NewClient(baseURL, timeout)
    store := storage.NewStoreWithCache(apiClient)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    store.StartCache(ctx)
    
    // Server runs with automatic data updates
    // Cache stops automatically on context cancellation
}
```

### Manual Updates
```go
// Force immediate update
data := fetchFromAPI()
store.LoadData(models.APIResponse{
    Artists:   data.Artists,
    Locations: data.Locations,
    Dates:     data.Dates,
    Relations: data.Relations,
})
```

## Configuration

### Constants
```go
const (
    CacheUpdateInterval = 30 * time.Second  // Update frequency
)
```

### Customization
```go
store := storage.NewStoreWithCache(apiClient)
// Modify update interval for testing
store.updateInterval = 5 * time.Second
```

## Thread Safety Guarantees

### Read Operations (RLock)
- `GetArtist()`, `GetAllArtists()`
- `GetLocation()`, `GetAllLocations()`
- `GetDate()`, `GetAllDates()`
- `GetRelation()`, `GetAllRelations()`
- `GetUniqueLocations()`, `GetUniqueDates()`
- `SearchArtists()`, `FilterArtistsByYear()`
- `GetStats()`

### Write Operations (Lock)
- `AddArtist()`, `AddLocation()`, `AddDate()`, `AddRelation()`
- `LoadData()`
- Internal `computeDerivedData()`

### Cache Operations (Separate Mutex)
- `GetLastUpdate()` (RLock)
- Cache state updates (Lock)

## Performance Characteristics

### Memory Usage
- **Base Storage**: O(n) where n = total data items
- **Derived Data**: O(u) where u = unique locations + unique dates
- **Concurrent Safety**: Minimal overhead from sync primitives

### Time Complexity
- **Lookups**: O(1) for all get operations
- **Search**: O(n) for text search operations
- **Updates**: O(n) for full data reload + O(u) for derived data

### Concurrent Performance
- **Read Scaling**: Excellent - multiple readers don't block each other
- **Write Blocking**: Minimal - writes are infrequent and fast
- **Cache Updates**: Non-blocking - run in background goroutines

## Error Handling

### API Failures
- Cache continues running on API errors
- Existing data remains available
- Errors logged but don't crash the application

### Concurrent Access
- Deadlock prevention through consistent lock ordering
- Panic recovery in critical sections
- Graceful degradation on resource exhaustion

### Resource Management
- Proper cleanup of goroutines and channels
- Context-based cancellation support
- Memory-efficient data structures

## Testing

### Coverage
- **86.1%** test coverage achieved
- **26** test functions covering all major scenarios
- **Concurrent access** testing with multiple goroutines
- **Cache functionality** testing with mock API clients

### Key Test Scenarios
- Cache start/stop operations
- Periodic update functionality
- Context cancellation handling
- Concurrent read/write operations
- Error resilience and recovery
- Data consistency during updates

### Running Tests
```bash
# Run all tests with coverage
go test -cover ./internal/storage/

# Run specific test categories
go test -run TestStore_Cache ./internal/storage/
go test -run TestStore_Concurrent ./internal/storage/
```

## Migration Guide

### From Old Storage (v1)
The new storage package maintains full API compatibility while adding cache functionality:

#### Before (Manual Loading)
```go
store := storage.NewStore()
data := fetchFromAPI()
store.LoadData(data)
```

#### After (Automatic Updates)
```go
store := storage.NewStoreWithCache(apiClient)
store.StartCache(ctx)
// Data loads automatically and updates every 30 seconds
```

### Breaking Changes
- None - fully backward compatible
- Old `NewStore()` still works without cache features
- All existing methods have identical signatures

### New Features Available
- `NewStoreWithCache()` - Enhanced constructor
- `StartCache()` / `StopCache()` - Cache management
- `GetLastUpdate()` - Cache status information
- `IsRunning()` - Cache state checking

## Best Practices

### Production Deployment
1. **Always use cache**: `NewStoreWithCache()` for production
2. **Context management**: Use proper context cancellation
3. **Graceful shutdown**: Call `StopCache()` before server shutdown
4. **Monitor cache**: Check `GetLastUpdate()` for health monitoring

### Development/Testing
1. **Use shorter intervals**: Set `updateInterval` for faster testing
2. **Mock API clients**: Implement `APIClient` interface for tests
3. **Verify thread safety**: Test concurrent access patterns
4. **Check coverage**: Maintain high test coverage for reliability

### Performance Tuning
1. **Read-heavy workloads**: Current design is optimal
2. **Write-heavy workloads**: Consider batching updates
3. **Memory constraints**: Monitor derived data size
4. **Network reliability**: Adjust update intervals based on API stability

## Troubleshooting

### Common Issues

#### Cache Not Starting
```go
// Check API client is set
if store.apiClient == nil {
    log.Fatal("API client not configured")
}

// Verify context is not cancelled
ctx := context.Background() // Fresh context
store.StartCache(ctx)
```

#### High Memory Usage
```go
// Monitor derived data size
stats := store.GetStats()
log.Printf("Unique locations: %d, dates: %d", 
    stats["locations"], stats["dates"])
```

#### Slow Updates
```go
// Check last update time
lastUpdate := store.GetLastUpdate()
if time.Since(lastUpdate) > 2*storage.CacheUpdateInterval {
    log.Warn("Cache updates are lagging")
}
```

### Debug Information
```go
// Cache status
fmt.Printf("Cache running: %v\n", store.IsRunning())
fmt.Printf("Last update: %v\n", store.GetLastUpdate())
fmt.Printf("Stats: %+v\n", store.GetStats())
```

## Future Enhancements

### Planned Features
- **Configurable update intervals** per data type
- **Selective updates** (only changed data)
- **Persistence layer** integration
- **Metrics collection** for monitoring
- **Circuit breaker** pattern for API failures

### API Evolution
The storage package is designed for future extensibility:
- Interface-based design allows easy mocking and testing
- Separate concerns (storage, caching, API communication)
- Clean separation between data and cache management
- Pluggable architecture for different backends

---

*This documentation is current as of the latest refactoring. For implementation details, see the source code and comprehensive test suite.*
