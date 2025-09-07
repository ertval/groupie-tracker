# Storage Package Documentation

## Overview

The storage package provides a high-performance, thread-safe in-memory data store with automatic cache functionality for the Groupie Tracker application. It has been refactored to include production-ready features like periodic data updates, concurrent access management, and efficient derived data computation.

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
