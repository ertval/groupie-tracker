# Data Package Refactoring Plan

This document outlines a plan to refactor the `data` package to improve simplicity, performance, and maintainability, adhering to idiomatic Go practices and the KISS (Keep It Simple, Stupid) principle.

## 1. Goals

*   **Simplicity:** Reduce complexity by creating a clear and straightforward data-handling pipeline.
*   **Performance:** Eliminate all runtime computations (sorting, searching, aggregating) in getter methods by pre-computing and storing results during initial data load.
*   **Clarity:** Establish a clear separation between raw data structures fetched from the API and the internal domain models used by the application.
*   **Idiomatic Go:** Follow Go best practices for package design and data encapsulation.

## 2. Refactoring Strategy

The core idea is to transform the `Repository` from a data processor into a simple, read-only cache of pre-computed data. The `LoadData` function will become the sole engine for fetching, processing, and organizing all data required by the application's handlers and templates.

### Phase 1: Redefine Data Structures

We will create a clear distinction between the data as it comes from the API and the data as we use it internally.

#### a. API Data Models

These structs will exactly match the JSON structure of the external API. They will only be used during the fetching process.

```go
// In internal/data/api.go

// APIArtist represents a single artist record from the /api/artists endpoint.
type APIArtist struct {
    ID           int      `json:"id"`
    Name         string   `json:"name"`
    Members      []string `json:"members"`
    CreationYear int      `json:"creationDate"`
    FirstAlbum   string   `json:"firstAlbum"`
    Image        string   `json:"image"`
}

// APIRelation represents the concert relations for all artists from the /api/relation endpoint.
type APIRelation struct {
    Index map[string][]string `json:"index"` // ArtistID -> DatesLocations
}
```

#### b. Internal Domain Models

These are the clean, computed data structures the rest of our application will use. They contain no JSON tags and include fields for pre-computed values.

```go
// In internal/data/domain.go

// Artist is the rich internal representation of an artist.
type Artist struct {
    ID            int
    Name          string
    Slug          string
    Members       []string
    CreationYear  int
    FirstAlbum    string
    Image         string
    Concerts      []Concert // A slice of structured Concert objects
    ConcertCount  int
    Countries     []string // Pre-sorted list of unique countries
    NextArtistID  int      // ID of the next artist (for navigation)
    PrevArtistID  int      // ID of the previous artist
}

// Location is the rich internal representation of a concert location.
type Location struct {
    Name          string
    Slug          string
    Artists       []Artist // Artists who have played here
    ArtistCount   int
    TotalConcerts int
}

// Concert represents a single concert event.
type Concert struct {
    Date     string
    Location string // Normalized location name
}
```

### Phase 2: Redesign the Repository

The `Repository` will no longer hold intermediate data. It will hold the final, query-ready, pre-computed datasets that the handlers need. All fields will be unexported to maintain encapsulation.

```go
// In internal/data/repository.go

type Repository struct {
    // Configuration
    baseURL string
    client  *http.Client

    // Pre-computed and sorted data collections
    artists         []Artist
    artistsByID     map[int]Artist
    artistsBySlug   map[string]Artist
    locations       []Location
    locationsBySlug map[string]Location
    globalStats     map[string]int
}
```

### Phase 3: Re-implement `LoadData`

This function will perform the entire ETL (Extract, Transform, Load) process.

1.  **Extract:** Fetch raw data from `/api/artists` and `/api/relation` into the `APIArtist` and `APIRelation` structs.
2.  **Transform:**
    *   Create a map of `Artist` domain models from the fetched `APIArtist` slice.
    *   Iterate through the `APIRelation` data, creating `Concert` objects and appending them to the corresponding `Artist` in the map.
    *   While processing artists, compute their slugs, concert counts, and country lists.
    *   Create a map of `Location` domain models, populating them by iterating through all artists and their concerts. Compute `ArtistCount` and `TotalConcerts` for each location.
    *   Create a sorted slice of all `Artist` models.
    *   Iterate through the sorted artist slice to populate the `NextArtistID` and `PrevArtistID` fields for each artist.
    *   Create sorted slices of `Location` models.
    *   Compute global statistics (`total_artists`, etc.).
3.  **Load:**
    *   Populate the `Repository` fields (`artists`, `artistsByID`, `artistsBySlug`, etc.) with the fully processed and sorted data.

### Phase 4: Simplify Getter Methods

Getters will become trivial, high-performance map lookups or slice returns. They will perform **zero** computation.

```go
// In internal/data/repository.go

// GetArtists returns a pre-sorted slice of all artists.
func (r *Repository) GetArtists() []Artist {
    return r.artists
}

// GetArtistByID returns an artist by their ID.
func (r *Repository) GetArtistByID(id int) (Artist, bool) {
    artist, found := r.artistsByID[id]
    return artist, found
}

// GetArtistBySlug returns an artist by their slug.
func (r* Repository) GetArtistBySlug(slug string) (Artist, bool) {
    artist, found := r.artistsBySlug[slug]
    return artist, found
}

// GetLocations returns a pre-sorted slice of all locations.
func (r *Repository) GetLocations() []Location {
    return r.locations
}

// GetStats returns pre-computed global statistics.
func (r *Repository) GetStats() map[string]int {
    return r.globalStats
}
```

## 3. Summary of Benefits

*   **Drastically Improved Performance:** Template rendering will be much faster as all data lookups and iterations will be on pre-computed, sorted data.
*   **Simplified Logic:** Handlers become simpler because they no longer need to perform calculations. The logic is centralized in `LoadData`.
*   **Enhanced Readability:** The separation of concerns makes the entire package easier to understand and maintain.
*   **Robustness:** The system becomes more robust as the data is processed and validated once at startup, reducing the chance of runtime errors.
