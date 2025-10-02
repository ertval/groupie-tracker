### GEMINI REFACTORING PLAN

This plan aims to simplify the Groupie Tracker codebase to make it more compact, clearer, more maintanable, and easier to understand. The focus is on simplifying data structures and the search/filter functionality, while adhering to idiomatic Go and the KISS principle.

#### 1. Simplify Data Structures (`internal/data/models.go`)

*   **Objective:** Make the core data models (`Artist`, `Location`) leaner by separating computed/derived data from the core fields. This will make the models represent the raw data more closely and reduce their in-memory footprint.

*   **Actions:**
    1.  **Refactor `Artist` and `Location` structs:**
        *   Remove the computed fields (`ConcertCount`, `Countries`, `MemberCount`, `FirstAlbumYear` from `Artist`, and `ArtistCount`, `TotalConcerts`, `EarliestYear`, `LatestYear` from `Location`) from the structs.
        *   Calculate these values on-the-fly when needed. Given the small dataset, the performance impact will be negligible, and this will significantly simplify the data models.
    2.  **Simplify Filter Parameter Structs:**
        *   The current use of pointers (`*int`) in `ArtistFilterParams` and `LocationFilterParams` is acceptable for distinguishing between zero-values and not-provided values. We will keep this, but simplify the filtering logic that uses these structs.

#### 2. Refactor Search and Filter Logic (`internal/data/filters.go`, `internal/data/searches.go`)

*   **Objective:** Make the filtering and searching logic more functional, composable, and easier to read.

*   **Actions:**
    1.  **Introduce a Functional Filtering Approach:**
        *   Instead of large `matchesArtistFilters` and `matchesLocationFilters` functions, create a `Filter` type (e.g., `type ArtistFilter func(*Artist) bool`).
        *   Create a chain of filters. Each filter function will be responsible for a single criterion (e.g., `byCreationYear`, `byMemberCount`).
        *   The main `FilterArtists` function will then iterate through the artists and apply the chain of filters. This makes the code more modular and easier to test.
    2.  **Simplify Search Suggestion Logic:**
        *   In `filterSearchSuggestions`, instead of using three separate slices for exact, prefix, and contains matches, use a single slice and sort it based on a calculated relevance score. This will make the code shorter and easier to understand.
    3.  **Simplify Suggestion Generation:**
        *   The `generateSearchSuggestions` function can be streamlined to reduce complexity and improve readability.

#### 3. Improve Web Handlers (`internal/web/handlers.go`)

*   **Objective:** Make the HTTP handlers thinner and more focused on handling HTTP-related tasks, moving business logic to the `data` layer.

*   **Actions:**
    1.  **Use Named Structs for Template Data:**
        *   Replace the anonymous structs used for passing data to templates with named structs (e.g., `HomePageData`, `ArtistsPageData`). This improves readability and maintainability.
    2.  **Centralize Repetitive Logic:**
        *   Create helper functions for common tasks like parsing form data and preparing template data.
    3.  **Move Logic out of Handlers:**
        *   Move the filter parameter parsing logic from the handlers into dedicated functions.
        *   Move sorting logic from handlers to the `data` layer if it represents a core business rule.

#### 4. General Code Simplification and Best Practices

*   **Objective:** Apply general Go best practices to improve the overall code quality.

*   **Actions:**
    1.  **Review variable names for clarity and conciseness.**
    2.  **Ensure all functions have clear documentation.**
    3.  **Add more unit tests for the new filter and search logic.**
