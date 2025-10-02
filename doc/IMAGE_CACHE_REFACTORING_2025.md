# Image Cache Refactoring - October 2025

## Overview

This refactoring simplifies the image caching functionality by removing the optional caching behavior and making it always enabled. The project now always caches all artist images to the local drive at startup after initial fetch.

## Changes Made

### 1. Configuration Simplification (`internal/conf/conf.go`)
- **Removed**: `WithCache` boolean configuration variable
- **Reason**: Caching is now always enabled, no need for configuration option

### 2. Data Store Simplification (`internal/data/store.go`)
- **Removed**: `withCache` and `cacheEnabled` boolean fields from `Store` struct
- **Removed**: `withCache` parameter from `NewStore()` constructor
- **Removed**: `CacheEnabled()` accessor method
- **Simplified**: `loadData()` method now always executes image caching without conditional logic
- **Updated**: Comments to reflect that caching is always enabled

### 3. Cache Implementation (`internal/data/cache.go`)
- **Removed**: Early return check for `withCache` flag
- **Updated**: Documentation to clarify that images are always cached
- **Preserved**: Adaptive worker pool that scales with CPU cores for optimal performance

### 4. Web Server (`internal/web/server.go`)
- **Removed**: `withCache` parameter from `NewApp()` function
- **Simplified**: Logging to always show cache statistics (cached/downloaded images)
- **Updated**: Comments to reflect automatic caching behavior

### 5. Main Entry Point (`cmd/server/main.go`)
- **Removed**: Reference to `conf.WithCache` when creating the app
- **Updated**: Comments to clarify image caching is always enabled

### 6. Test Files
- **Updated**: `tests/integration_test.go` - removed WithCache configuration
- **Updated**: `tests/e2e_test.go` - removed WithCache configuration and cleanup
- **Updated**: `internal/web/web_test.go` - removed WithCache configuration and CacheEnabled test
- **Updated**: `internal/data/fixtures.go` - removed cacheEnabled field assignment

### 7. Documentation (`README.md`)
- **Updated**: Highlights section to clarify image caching is always enabled
- **Updated**: Data layer description to reflect automatic caching behavior

## Benefits

1. **Simplified Code**: Removed ~50 lines of conditional logic and configuration
2. **Reduced Complexity**: No need to track cache state or handle enabled/disabled scenarios
3. **Consistent Behavior**: Application always behaves the same way regarding image caching
4. **Better Performance**: Images are always available locally, reducing external API calls
5. **Easier Testing**: No need to test multiple caching scenarios

## Technical Details

### Image Caching Behavior
- **When**: Images are cached at application startup during the `Load()` phase
- **Where**: Stored in `static/img/artists/` directory
- **How**: Adaptive worker pool scales with `runtime.NumCPU()` for optimal concurrency
- **Naming**: Images saved as `{artist-slug}.jpg` (e.g., `queen.jpg`)
- **Reuse**: Existing cached images are detected and reused (no re-download)

### Statistics Tracking
The application still tracks and logs:
- **Cached Images**: Number of images already present on disk
- **Downloaded Images**: Number of images downloaded in current run
- Logged at startup: `Data loaded - X artists (cached: Y images, downloaded: Z images)`

### Error Handling
- If cache directory creation fails, the application continues with external URLs
- Individual image download failures don't block startup
- 10-second timeout per image prevents hanging on slow/dead URLs

## Compatibility

### Breaking Changes
- **API Change**: `NewStore()` no longer accepts `withCache` parameter
- **API Change**: `NewApp()` no longer accepts `withCache` parameter
- **Removed API**: `Store.CacheEnabled()` method no longer exists
- **Configuration**: `conf.WithCache` variable removed

### Migration Guide
If you have custom code that uses these APIs:

**Before:**
```go
store := data.NewStore(apiClient, true)
app, err := web.NewApp(apiClient, conf.WithCache)
if store.CacheEnabled() {
    // ...
}
```

**After:**
```go
store := data.NewStore(apiClient)
app, err := web.NewApp(apiClient)
// Caching is always enabled, no need to check
```

## Testing

All tests pass after refactoring:
- ✅ `internal/data` package tests (100+ test cases)
- ✅ `internal/web` package tests (10 test suites)
- ✅ `tests` integration tests
- ✅ Build succeeds without errors

## Performance Impact

**No negative performance impact**. The changes simplify code without affecting runtime performance:
- Image caching still uses adaptive worker pool (same performance characteristics)
- No additional overhead from removed conditional checks
- Cache directory is created once at startup (minimal cost)

## Future Considerations

### Hourly Rehydration
The current implementation caches images at startup. For hourly rehydration:
1. Add a background goroutine with `time.Ticker`
2. Call `store.Load()` or `store.cacheImages()` periodically
3. Consider graceful refresh without blocking requests

### Cache Management
Potential future enhancements:
- Cache expiration/TTL
- Cache size limits
- Automatic cleanup of stale images
- Cache warming endpoint for manual refresh

## Conclusion

This refactoring successfully simplifies the codebase by removing optional caching logic while maintaining all functionality. The project now has a cleaner, more maintainable architecture with consistent behavior across all deployments.

---

**Date**: October 3, 2025  
**Status**: ✅ Complete  
**Tests**: ✅ All Passing
