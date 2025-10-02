# Location Sorting Fix

## Date: October 2, 2025

## Issue
The locations page was not sorted by the number of concerts per location. Locations were being sorted alphabetically by name instead.

## Root Cause
In `internal/data/catalog.go`, the `AllLocations()` method was using alphabetical sorting:

```go
sort.Slice(locations, func(i, j int) bool {
    return locations[i].Name < locations[j].Name
})
```

## Solution
Updated the `AllLocations()` method to sort by total concert count (descending), with name as a tiebreaker:

```go
sort.Slice(locations, func(i, j int) bool {
    countI := locations[i].TotalConcerts()
    countJ := locations[j].TotalConcerts()
    if countI != countJ {
        return countI > countJ // Descending by concert count
    }
    return locations[i].Name < locations[j].Name // Ascending by name for ties
})
```

## Testing
- Added new test `TestCatalogLocationsSortedByConcertCount` to verify sorting behavior
- All existing tests continue to pass (33/33)
- Server starts successfully and locations are now properly sorted

## Files Modified
- `internal/data/catalog.go` - Updated `AllLocations()` sorting logic
- `internal/data/catalog_test.go` - Added test for location sorting

## Verification
The locations page now displays locations ordered by:
1. **Primary:** Total concert count (highest to lowest)
2. **Secondary:** Location name (alphabetically) for locations with the same concert count

This provides a better user experience by showing the most popular venues first.
