# Migration Guide - Phase 5 Changes

This guide covers the API changes introduced in Phase 5 of the refactoring plan.

## Summary

Phase 5 focused on code polish and documentation with minimal breaking changes. The main changes are method name normalizations to follow idiomatic Go conventions.

## Breaking Changes

### Method Renames

The following public methods have been renamed to follow Go naming conventions (removing "Get" prefixes):

#### Store Methods

| Old Name | New Name | Description |
|----------|----------|-------------|
| `GetArtistFilterOptions()` | `ArtistFilterOptions()` | Returns filter metadata for artists |
| `GetLocationFilterOptions()` | `LocationFilterOptions()` | Returns filter metadata for locations |
| `GetAdjacentArtists()` | `AdjacentArtists()` | Returns previous/next artists |

### Migration Steps

#### 1. Update Filter Options Calls

**Before:**
```go
artistFilters := store.GetArtistFilterOptions()
locationFilters := store.GetLocationFilterOptions()
```

**After:**
```go
artistFilters := store.ArtistFilterOptions()
locationFilters := store.LocationFilterOptions()
```

#### 2. Update Adjacent Artists Calls

**Before:**
```go
prev, next := store.GetAdjacentArtists(currentID)
```

**After:**
```go
prev, next := store.AdjacentArtists(currentID)
```

### Automated Migration

You can use these commands to update your code automatically:

```bash
# For ArtistFilterOptions
find . -type f -name "*.go" -exec sed -i 's/GetArtistFilterOptions/ArtistFilterOptions/g' {} +

# For LocationFilterOptions
find . -type f -name "*.go" -exec sed -i 's/GetLocationFilterOptions/LocationFilterOptions/g' {} +

# For AdjacentArtists
find . -type f -name "*.go" -exec sed -i 's/GetAdjacentArtists/AdjacentArtists/g' {} +
```

On Windows with PowerShell:
```powershell
# For ArtistFilterOptions
Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
    (Get-Content $_.FullName) -replace 'GetArtistFilterOptions', 'ArtistFilterOptions' | Set-Content $_.FullName
}

# For LocationFilterOptions
Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
    (Get-Content $_.FullName) -replace 'GetLocationFilterOptions', 'LocationFilterOptions' | Set-Content $_.FullName
}

# For AdjacentArtists
Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
    (Get-Content $_.FullName) -replace 'GetAdjacentArtists', 'AdjacentArtists' | Set-Content $_.FullName
}
```

## Non-Breaking Changes

### Code Quality Improvements

1. **Comment Cleanup**: Removed obvious comments that stated "what" instead of "why"
2. **Formatting**: Applied `go fmt` across all files
3. **Package Documentation**: Updated package-level documentation

### Documentation Updates

1. **ARCHITECTURE.md**: New comprehensive architecture documentation
2. **README.md**: Updated with current state of the project
3. **Migration Guide**: This document

## Compatibility

- **Go Version**: Requires Go 1.24.3+ (no change)
- **API**: No changes to external API integration
- **Templates**: No changes to template interface
- **Configuration**: No changes to configuration

## Testing

All existing tests have been updated and pass:

```bash
go test ./... -v
```

Test coverage remains:
- Data layer: 60.5%
- Web layer: 48.3%

## Rollback

If you need to rollback these changes, simply revert the method renames:

```go
// Add these wrapper methods to Store
func (s *Store) GetArtistFilterOptions() ArtistFilterOptions {
    return s.ArtistFilterOptions()
}

func (s *Store) GetLocationFilterOptions() LocationFilterOptions {
    return s.LocationFilterOptions()
}

func (s *Store) GetAdjacentArtists(currentID int) (prev, next *Artist) {
    return s.AdjacentArtists(currentID)
}
```

## Support

For questions or issues with migration:
1. Check the updated [ARCHITECTURE.md](./ARCHITECTURE.md)
2. Review the [README.md](./README.md)
3. Check test examples in `internal/data/data_test.go` and `internal/web/web_test.go`

## Timeline

- **Phase 5 Start**: October 2, 2025
- **Phase 5 Complete**: October 2, 2025
- **Recommended Migration**: Immediate (changes are minimal)
