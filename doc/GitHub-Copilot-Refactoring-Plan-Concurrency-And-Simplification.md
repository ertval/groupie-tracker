# Refactoring & Optimization Plan: Concurrency and Simplification

This document outlines a refactoring plan for the Groupie Tracker application. The primary goals are to improve performance by introducing concurrency during startup and to simplify the data loading and processing logic, adhering to idiomatic Go practices and the KISS (Keep It Simple, Stupid) principle.

## 1. Executive Summary

The current application loads all data from the external API sequentially at startup. This is a bottleneck that increases server start time. The proposed refactoring will parallelize the fetching of artist and relation data, significantly reducing the initial load time.

The core of this plan involves modifying the `domain.Repository`'s data loading mechanism to use goroutines and channels, making the startup process faster and more robust.

## 2. Analysis of Current State

The application follows a clean three-layer architecture: `api`, `domain`, and `web`. The `domain.Repository` is responsible for all data operations and holds the application's state in memory.

- **Bottleneck:** The `NewRepository` function (or its equivalent `LoadData` method) currently fetches data sequentially. It gets a list of all artists and then iterates through them one by one to fetch their corresponding relation data (concert dates, locations).
- **Opportunity:** Since the API calls for each artist's relations are independent, they are a perfect candidate for concurrent execution.

## 3. Proposed Refactoring Plan

### Phase 1: Concurrent Data Loading in `internal/domain/repository.go`

The main focus is to refactor the data loading logic to fetch artist details and their relations concurrently.

**File to Modify:** `internal/domain/repository.go`

**Function to Refactor:** `LoadData()` (and by extension, `NewRepository()`)

**Step-by-step Implementation:**

1.  **Introduce Goroutines for API Calls:**
    -   Modify the `LoadData` function to iterate over the list of base artist information.
    -   For each artist, launch a goroutine to perform the following tasks:
        -   Fetch the full artist details from the `/artists/<id>` endpoint.
        -   Fetch the relation data from the `/relation/<id>` endpoint.
    -   This will change the data fetching pattern from a sequential loop to a parallel fan-out pattern.

2.  **Manage Concurrency with `sync.WaitGroup`:**
    -   Use a `sync.WaitGroup` to ensure that the `LoadData` function waits for all artist-fetching goroutines to complete before it proceeds to build the final data structures and indexes.

3.  **Aggregate Results and Handle Errors Safely:**
    -   Use channels to collect the results (the fully enriched `Artist` structs) and any potential errors from the goroutines.
    -   Create a struct (e.g., `artistJobResult`) to wrap the resulting `Artist` and an `error` field. This allows passing both over a single channel.
    -   After dispatching all goroutines, start a separate goroutine to read from the results channel. This collector will populate the `artists` slice and log any errors encountered during the fetches.

4.  **Refactor Data Processing:**
    -   Once all data is fetched and collected, the final processing steps (e.g., creating location data, building indexes for `artistsBySlug`, `locationsBySlug`) can proceed as they do now, but operating on the complete, concurrently-fetched dataset.

**Example Snippet (Conceptual):**

```go
// In internal/domain/repository.go

type artistJobResult struct {
    artist Artist
    err    error
}

func (r *Repository) LoadData() error {
    // ... fetch initial list of artists from API ...
    
    numArtists := len(initialArtists)
    resultsChan := make(chan artistJobResult, numArtists)
    var wg sync.WaitGroup
    
    for _, basicArtist := range initialArtists {
        wg.Add(1)
        go func(artistID int) {
            defer wg.Done()
            
            // Fetch full artist and relation data concurrently
            fullArtist, err := r.apiClient.GetArtist(artistID)
            if err != nil {
                resultsChan <- artistJobResult{err: fmt.Errorf("failed to get artist %d: %w", artistID, err)}
                return
            }
            
            relations, err := r.apiClient.GetRelation(artistID)
            if err != nil {
                resultsChan <- artistJobResult{err: fmt.Errorf("failed to get relations for artist %d: %w", artistID, err)}
                return
            }
            
            // Process and combine data into a domain.Artist
            processedArtist := processArtistData(fullArtist, relations) 
            resultsChan <- artistJobResult{artist: processedArtist}
        }(basicArtist.ID)
    }
    
    // Wait for all goroutines to finish, then close the channel
    wg.Wait()
    close(resultsChan)
    
    // Collect results and build the repository data
    for result := range resultsChan {
        if result.err != nil {
            // Log the error and decide whether to continue or fail
            log.Printf("Error loading artist data: %v", result.err)
            continue // Or return an error to halt startup
        }
        r.artists = append(r.artists, result.artist)
    }
    
    // ... proceed with building indexes and pre-calculating stats ...
    
    return nil
}
```

### Phase 2: Simplification and Code Cleanup

- **Review `internal/domain/models.go`:** Ensure that the domain models are concise and only contain fields necessary for the application's logic and templates. Remove any redundant or calculated fields that can be computed on-the-fly if they are not frequently accessed. (The current structure with pre-computed fields seems efficient for a read-only system, so this is more of a review than a required change).
- **Error Handling:** Standardize error handling. Ensure errors from the API layer are wrapped with context in the `domain` layer, providing a clear trace for debugging.

## 5. Expected Benefits

1.  **Improved Performance:** A significant reduction in server startup time, as dozens of API calls will be made in parallel instead of sequentially.
2.  **Enhanced Robustness:** By handling errors from individual API calls within goroutines, the system can be made more resilient. It could potentially start up with partial data if some non-critical API calls fail, rather than failing to start entirely.
3.  **Idiomatic Code:** The use of goroutines, channels, and wait groups is the standard, idiomatic Go way to handle concurrent I/O-bound operations.

## 6. Validation

After refactoring, the following steps will be taken to ensure correctness:
1.  Run all existing unit and end-to-end tests using `go test ./...`.
2.  Manually verify that the application starts correctly and that all data (artists, locations, concert dates) is present and accurate on the website.
3.  Add a new unit test for the `LoadData` function that uses a mock API client to verify the concurrent fetching logic.
