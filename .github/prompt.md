Lets follow **test driven development** principles to implement the project as required in the `requirements.md`. Continue where you left the last time. Check `todo.md` for the next steps. Dont stop untill you finish all the steps.
- Create gitignore and license files and documentation, Have a readme file that explains the project structure, setup instructions, and any other relevant information that you keep updated after each important change.
- Write unit tests for all new functionality.
- Do it in small, manageable steps, writing tests for each piece of functionality before implementing it, commit to git after each step.
- Organize the project in a modular way, separating different components and functionalities into their own files and directories.
- Keep a `todo.md` file with the steps of the implementation. **Update it regularly** so you know what is the current state. Also include any new tasks that arise during development and keep detailed documentation of the implementation process.
- In the end write integration comprehensive tests based on `audit.md` to ensure all aspects of the requirements are covered, including edge cases and test all described inputs of the `audit.md` file, only then the project can be considered complete.

---

 Write end to end tests and check that everything works as intended, use #file:audit.md  and #file:requirements.md  for functionality reference. 
 - Check that all templates load correctly
 - Check all visually and functionally using mcp:playwright
 - Save tests in tests folder
 - Update documentation with current state

 ---

 ## We need to implement the following changes:
 - In the location template in the end there is list of most popular locations, it is not working correctly, it should show the locations with the most concerts in total. Fix this.
 - The storage and service layers are a bit messy and complicated, we need to refactor them to be more clear and simple. Use only one package, have a single store struct that handles all the data, and a single service struct that handles all the business logic. Remove any unnecessary abstractions or layers. Make sure the code is easy to read and understand.
 - Write comprehensive tests for the refactored code, covering all the main functionalities and edge cases. Make sure the tests are easy to read and understand, and provide good coverage of the codebase. Update existing tests as needed to reflect the changes made during the refactoring process.
 - Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.
 ### Make sure the project is well organized and structured, with clear separation of concerns and responsibilities. Use appropriate naming conventions and file structures to make it easy to navigate and understand the codebase.

# Clean UP and Optimization
- Remove any unused code, comments, or files that are no longer needed.
- Remove older versions of the files that you changed in the last refactoring. Keep only the simplified versions.
- Rename everything to remove the "Simplified" prefix, so that the new files have the proper shorter names.
- Restructure everything again to be more simple and clear, modular, easy to understand, maintainable and testable:
    - storage package: single store struct, all data operations
    - service package: single service struct, all business logic, calculations (all custom computations here: e.g. location stats, totals, etc.)
    - handlers package: single handlers struct, all HTTP handling
    - Update all imports and references accordingly.
-  - Write comprehensive tests for the refactored code, covering all the main functionalities and edge cases. Make sure the tests are easy to read and understand, and provide good coverage of the codebase. Update existing tests as needed to reflect the changes made during the refactoring process.
 - Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.
 
 ---

 These computations and any other business logic should be in the service layer, not in the storage layer.

```go
// SearchArtists searches for artists by name or member names (case-insensitive).
// Returns artists sorted alphabetically by name.
func (s *Store) SearchArtists(query string) []models.Artist {
	allArtists := s.GetAllArtists()

	if query == "" {
		return allArtists
	}

	query = strings.ToLower(query)
	var results []models.Artist

	for _, artist := range allArtists {
		// Search in artist name
		if strings.Contains(strings.ToLower(artist.Name), query) {
			results = append(results, artist)
			continue
		}

		// Search in member names
		found := false
		for _, member := range artist.Members {
			if strings.Contains(strings.ToLower(member), query) {
				results = append(results, artist)
				found = true
				break
			}
		}
		if found {
			continue
		}
	}

	return s.sortArtistsByName(results)
}

// FilterArtistsByYear filters artists by creation year range.
// If minYear or maxYear is 0, that bound is ignored.
// Returns artists sorted alphabetically by name.
func (s *Store) FilterArtistsByYear(minYear, maxYear int) []models.Artist {
	allArtists := s.GetAllArtists()
	var results []models.Artist

	for _, artist := range allArtists {
		// If no year restrictions, include all
		if minYear == 0 && maxYear == 0 {
			results = append(results, artist)
			continue
		}

		// Apply year filters
		if minYear > 0 && artist.CreationYear < minYear {
			continue
		}
		if maxYear > 0 && artist.CreationYear > maxYear {
			continue
		}

		results = append(results, artist)
	}

	return s.sortArtistsByName(results)
}

// computeUniqueData pre-computes unique locations and dates for performance.
// This method should be called after loading data and must be called with write lock held.
func (s *Store) computeUniqueData() {
	locationSet := make(map[string]bool)
	dateSet := make(map[string]bool)

	// Extract unique locations from relations
	for _, relation := range s.relations {
		for location, dates := range relation.DatesLocations {
			locationSet[location] = true
			for _, date := range dates {
				dateSet[date] = true
			}
		}
	}

	// Convert sets to sorted slices
	s.uniqueLocations = make([]string, 0, len(locationSet))
	for location := range locationSet {
		s.uniqueLocations = append(s.uniqueLocations, location)
	}
	sort.Strings(s.uniqueLocations)

	s.uniqueDates = make([]string, 0, len(dateSet))
	for date := range dateSet {
		s.uniqueDates = append(s.uniqueDates, date)
	}
	sort.Strings(s.uniqueDates)
}

// sortArtistsByName sorts artists alphabetically by name (case-insensitive).
func (s *Store) sortArtistsByName(artists []models.Artist) []models.Artist {
	// Create a copy to avoid modifying the original slice
	sorted := make([]models.Artist, len(artists))
	copy(sorted, artists)

	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}
```
Also 
// GetStats returns basic statistics about the stored data.
func (s *Store) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"artists":   len(s.artists),
		"locations": len(s.locations),
		"dates":     len(s.dates),
		"relations": len(s.relations),
	}
}
Seems to do the calculations wrong, it should return total number of concerts (relations) per location, not just count of unique locations also dates are wrong. Fix this. and move to service layer.

- Any functions in store should not sort the date of filter them, just return the raw data, sorting and filtering should be done in service layer.

---

We need to do the following changes:
- If a template is missing a required field, or the templates are not loaded correctly, the server should log an error and return a 500 Internal Server Error response. NOT PRINT SIMPLE HTML PAGE.
- The handler for artist details mux.HandleFunc("/artists/", h.ArtistDetailHandler) should not allow url like /artists/123/extra, it should return 404 for such urls.
- Remove the all extra functionality like search,filter and refresh. Remove all related code, templates, handlers, tests, etc. We want to keep it simple.
- Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.

---

Simplify the data package to have a single store struct and a single service struct (or as few as possible if one is not sufficient). Remove any unnecessary abstractions or layers. Make sure the code is easy to read and understand.
- Move all the API structs to the api package from the data package. Keep them one to one with what the external API returns. No custom fields or modifications. Have also the validation methods here.
- Keep only the repository structs in the data package. These should be simple structs that represent the application's data model. Simplify them as much as possible, removing any unnecessary fields or methods, reduce duplication, and dont recalculate things that can be precomputed once and stored.
- Be consistent with naming conventions, and file structures to make it easy to navigate and understand the codebase.
- Update the coverage html to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.
- Update the readme file to reflect the current state of the project, including any changes made during the refactoring process. Make sure it is clear and concise, and provides all the necessary information for someone to understand and use the project. Also update all other documentation files as needed.

---

Remove the start server function. Create a bakingInfo function that logs all the important information about the server and the data when the server starts creates the clickable link to open the server in the browser. Call this function from main after everything is initialized and before starting the server. Then start the server directly in main. Always use Idiomatic Go patterns with clean architecture in your implementation.