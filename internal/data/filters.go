package data

import (
	"net/http"
	"sort"
	"strconv"
)

// IntRange represents an inclusive integer range [Min, Max].
type IntRange struct {
	Min int
	Max int
}

// Contains checks if the value is within the range [Min, Max] inclusive.
func (r IntRange) Contains(value int) bool {
	return value >= r.Min && value <= r.Max
}

// IsZero checks if the range is unset (both Min and Max are zero).
func (r IntRange) IsZero() bool {
	return r.Min == 0 && r.Max == 0
}

// StringSet is a set of unique strings for efficient membership testing.
type StringSet map[string]struct{}

// NewStringSet creates a StringSet from the given items.
func NewStringSet(items ...string) StringSet {
	s := make(StringSet, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

// Contains checks if the item exists in the set.
func (s StringSet) Contains(item string) bool {
	_, ok := s[item]
	return ok
}

// IsEmpty checks if the set has no elements.
func (s StringSet) IsEmpty() bool {
	return len(s) == 0
}

// IntSet is a set of unique integers for efficient membership testing.
type IntSet map[int]struct{}

// NewIntSet creates an IntSet from the given items.
func NewIntSet(items ...int) IntSet {
	s := make(IntSet, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

// Contains checks if the item exists in the set.
func (s IntSet) Contains(item int) bool {
	_, ok := s[item]
	return ok
}

// IsEmpty checks if the set has no elements.
func (s IntSet) IsEmpty() bool {
	return len(s) == 0
}

// ArtistFilterFunc is a predicate function that tests whether an artist matches a filter criterion.
type ArtistFilterFunc func(*Artist) bool

// LocationFilterFunc is a predicate function that tests whether a location matches a filter criterion.
type LocationFilterFunc func(*Location) bool

// AndFilters combines multiple artist filter functions with AND logic.
// The returned filter passes only if ALL provided filters pass.
func AndFilters(filters ...ArtistFilterFunc) ArtistFilterFunc {
	return func(a *Artist) bool {
		for _, f := range filters {
			if !f(a) {
				return false
			}
		}
		return true
	}
}

// AndLocationFilters combines multiple location filter functions with AND logic.
// The returned filter passes only if ALL provided filters pass.
func AndLocationFilters(filters ...LocationFilterFunc) LocationFilterFunc {
	return func(l *Location) bool {
		for _, f := range filters {
			if !f(l) {
				return false
			}
		}
		return true
	}
}

// CreationYearBetween creates a filter that checks if artist creation year is within [min, max].
func CreationYearBetween(min, max int) ArtistFilterFunc {
	return CreationYearInRange(IntRange{Min: min, Max: max})
}

// CreationYearInRange creates a filter using an IntRange. Returns always-true filter if range is zero.
func CreationYearInRange(r IntRange) ArtistFilterFunc {
	if r.IsZero() {
		return func(*Artist) bool { return true }
	}
	return func(a *Artist) bool {
		return r.Contains(a.CreationYear)
	}
}

// HasMemberCount creates a filter that checks if artist has any of the specified member counts.
func HasMemberCount(counts ...int) ArtistFilterFunc {
	return HasMemberCountInSet(NewIntSet(counts...))
}

// HasMemberCountInSet creates a filter using an IntSet. Returns always-true filter if set is empty.
func HasMemberCountInSet(counts IntSet) ArtistFilterFunc {
	if counts.IsEmpty() {
		return func(*Artist) bool { return true }
	}
	return func(a *Artist) bool {
		return counts.Contains(a.MemberCount())
	}
}

// InCountries creates a filter that checks if artist performed in any of the specified countries.
func InCountries(countries ...string) ArtistFilterFunc {
	return InCountrySet(NewStringSet(countries...))
}

// InCountrySet creates a filter using a StringSet. Returns always-true filter if set is empty.
func InCountrySet(countries StringSet) ArtistFilterFunc {
	if countries.IsEmpty() {
		return func(*Artist) bool { return true }
	}
	return func(a *Artist) bool {
		for _, country := range a.Countries() {
			if countries.Contains(country) {
				return true
			}
		}
		return false
	}
}

// FirstAlbumYearBetween creates a filter that checks if first album year is within [min, max].
func FirstAlbumYearBetween(min, max int) ArtistFilterFunc {
	return FirstAlbumYearInRange(IntRange{Min: min, Max: max})
}

// FirstAlbumYearInRange creates a filter using an IntRange. Returns always-true filter if range is zero.
func FirstAlbumYearInRange(r IntRange) ArtistFilterFunc {
	if r.IsZero() {
		return func(*Artist) bool { return true }
	}
	return func(a *Artist) bool {
		year := a.FirstAlbumYear()
		if year == 0 {
			return true // Don't filter out artists with no album year
		}
		return r.Contains(year)
	}
}

// ArtistFilters represents a complete set of filter criteria for artists.
// This is a value type that can be easily passed around and tested.
type ArtistFilters struct {
	CreationYear IntRange
	MemberCounts IntSet
	Countries    StringSet
	FirstAlbum   IntRange
}

// Match checks if an artist satisfies all filter criteria.
// Returns true only if the artist passes ALL non-empty filters (AND logic).
func (f ArtistFilters) Match(a *Artist) bool {
	if !f.CreationYear.IsZero() && !f.CreationYear.Contains(a.CreationYear) {
		return false
	}

	if !f.MemberCounts.IsEmpty() && !f.MemberCounts.Contains(a.MemberCount()) {
		return false
	}

	if !f.Countries.IsEmpty() {
		hasMatch := false
		for _, country := range a.Countries() {
			if f.Countries.Contains(country) {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			return false
		}
	}

	if !f.FirstAlbum.IsZero() {
		year := a.FirstAlbumYear()
		if year > 0 && !f.FirstAlbum.Contains(year) {
			return false
		}
	}

	return true
}

// IsEmpty checks if no filter criteria are set.
func (f ArtistFilters) IsEmpty() bool {
	return f.CreationYear.IsZero() &&
		f.MemberCounts.IsEmpty() &&
		f.Countries.IsEmpty() &&
		f.FirstAlbum.IsZero()
}

// ToFilterFunc converts the ArtistFilters struct into a composable filter function.
func (f ArtistFilters) ToFilterFunc() ArtistFilterFunc {
	return func(a *Artist) bool {
		return f.Match(a)
	}
}

// LocationFilters represents a complete set of filter criteria for locations.
type LocationFilters struct {
	ConcertCount IntRange
	ArtistCount  IntRange
	YearRange    IntRange
	Countries    StringSet
}

// Match checks if a location satisfies all filter criteria.
// Returns true only if the location passes ALL non-empty filters (AND logic).
func (f LocationFilters) Match(l *Location) bool {
	if !f.ConcertCount.IsZero() {
		total := l.TotalConcerts()
		if !f.ConcertCount.Contains(total) {
			return false
		}
	}

	if !f.ArtistCount.IsZero() {
		count := l.ArtistCount()
		if !f.ArtistCount.Contains(count) {
			return false
		}
	}

	if !f.YearRange.IsZero() {
		earliest, latest := l.YearRange()
		// Location must overlap with the filter range
		if latest < f.YearRange.Min || earliest > f.YearRange.Max {
			return false
		}
	}

	if !f.Countries.IsEmpty() {
		country := l.Country()
		if !f.Countries.Contains(country) {
			return false
		}
	}

	return true
}

// IsEmpty checks if no filter criteria are set.
func (f LocationFilters) IsEmpty() bool {
	return f.ConcertCount.IsZero() &&
		f.ArtistCount.IsZero() &&
		f.YearRange.IsZero() &&
		f.Countries.IsEmpty()
}

// ToFilterFunc converts the LocationFilters struct into a composable filter function.
func (f LocationFilters) ToFilterFunc() LocationFilterFunc {
	return func(l *Location) bool {
		return f.Match(l)
	}
}

// FilterArtistsV2 applies the new ArtistFilters type to filter the artist collection.
// This is the preferred method going forward - uses composable filter structs.
func (s *Store) FilterArtistsV2(filters ArtistFilters) []*Artist {
	artists := s.Artists()
	if len(artists) == 0 || filters.IsEmpty() {
		return artists
	}

	var filtered []*Artist
	for _, artist := range artists {
		if filters.Match(artist) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// FilterLocationsV2 applies the new LocationFilters type to filter the location collection.
// This is the preferred method going forward - uses composable filter structs.
func (s *Store) FilterLocationsV2(filters LocationFilters) []Location {
	locations := s.Locations()
	if len(locations) == 0 || filters.IsEmpty() {
		return locations
	}

	var filtered []Location
	for _, location := range locations {
		if filters.Match(&location) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// FilterArtists applies user-specified filter criteria to the artist collection and returns matching artists.
// Filters support: creation year range, first album year range, member count values, and countries.
// All criteria are ANDed together (artist must match ALL specified filters to be included).
// DEPRECATED: Use FilterArtistsV2 with the new ArtistFilters type instead.
func (s *Store) FilterArtists(criteria ArtistFilterParams) []*Artist {
	artists := s.Artists()
	if len(artists) == 0 {
		return nil
	}

	var filtered []*Artist
	for _, artist := range artists {
		if matchesArtistFilters(*artist, criteria) { // Check if artist matches all filter criteria
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// FilterLocations applies user-specified filter criteria to the location collection and returns matching locations.
// Filters support: concert count range, artist count range, year range (earliest/latest), and country.
// All criteria are ANDed together (location must match ALL specified filters to be included).
// DEPRECATED: Use FilterLocationsV2 with the new LocationFilters type instead.
func (s *Store) FilterLocations(params LocationFilterParams) []Location {
	locations := s.Locations()
	if len(locations) == 0 {
		return nil
	}

	var filtered []Location
	for _, location := range locations {
		if matchesLocationFilters(location, params) { // Check if location matches all filter criteria
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// matchesArtistFilters checks whether a single artist satisfies all specified filter criteria.
// Returns true only if the artist matches ALL non-nil/non-empty filter parameters (AND logic).
func matchesArtistFilters(artist Artist, params ArtistFilterParams) bool {
	// Filter by creation year range (e.g., "Show bands formed between 1970-1980")
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	// Filter by first album year range (only check if artist has a valid album year > 0)
	if params.FirstAlbumYearFrom != nil || params.FirstAlbumYearTo != nil {
		albumYear := artist.FirstAlbumYear()
		if albumYear > 0 { // Only apply filter if artist has a valid first album year
			if params.FirstAlbumYearFrom != nil && albumYear < *params.FirstAlbumYearFrom {
				return false
			}
			if params.FirstAlbumYearTo != nil && albumYear > *params.FirstAlbumYearTo {
				return false
			}
		}
	}

	// Filter by member count (e.g., "Show solo artists (1) or bands with 4 members")
	// This is an OR within the member counts list, but AND with other filters
	if len(params.MemberCounts) > 0 {
		memberCount := artist.MemberCount()
		found := false
		for _, allowedCount := range params.MemberCounts {
			if memberCount == allowedCount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by countries (e.g., "Show artists who performed in USA or UK")
	// Artist must have performed in at least ONE of the specified countries (OR within countries, AND with other filters)
	if len(params.Countries) > 0 {
		allowed := make(map[string]struct{}, len(params.Countries)) // Use map for O(1) lookup
		for _, country := range params.Countries {
			allowed[country] = struct{}{}
		}

		hasMatchingCountry := false
		for _, country := range artist.Countries() {
			if _, ok := allowed[country]; ok {
				hasMatchingCountry = true
				break
			}
		}
		if !hasMatchingCountry {
			return false
		}
	}

	return true // Artist passed all filter checks
}

// matchesLocationFilters checks whether a single location satisfies all specified filter criteria.
// Returns true only if the location matches ALL non-nil/non-empty filter parameters (AND logic).
func matchesLocationFilters(location Location, params LocationFilterParams) bool {
	// Filter by concert count range (e.g., "Show locations with 10-50 concerts")
	totalConcerts := location.TotalConcerts()
	if params.ConcertCountFrom != nil && totalConcerts < *params.ConcertCountFrom {
		return false
	}
	if params.ConcertCountTo != nil && totalConcerts > *params.ConcertCountTo {
		return false
	}

	artistCount := location.ArtistCount()
	if params.ArtistCountFrom != nil && artistCount < *params.ArtistCountFrom {
		return false
	}
	if params.ArtistCountTo != nil && artistCount > *params.ArtistCountTo {
		return false
	}

	earliestYear, latestYear := location.YearRange()
	if params.ConcertYearFrom != nil && latestYear < *params.ConcertYearFrom {
		return false
	}
	if params.ConcertYearTo != nil && earliestYear > *params.ConcertYearTo {
		return false
	}

	if len(params.Countries) > 0 {
		locationCountry := location.Country()
		for _, allowedCountry := range params.Countries {
			if locationCountry == allowedCountry {
				return true
			}
		}
		return false
	}

	return true
}

// calculateArtistFilterOptions derives available artist filter metadata from the dataset.
func (s *Store) calculateArtistFilterOptions(artists []*Artist) ArtistFilterOptions {
	if len(artists) == 0 {
		return ArtistFilterOptions{}
	}

	minCreationYear, maxCreationYear := artists[0].CreationYear, artists[0].CreationYear
	minFirstAlbumYear, maxFirstAlbumYear := 0, 0
	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	for _, artist := range artists {
		if artist.CreationYear < minCreationYear {
			minCreationYear = artist.CreationYear
		}
		if artist.CreationYear > maxCreationYear {
			maxCreationYear = artist.CreationYear
		}

		albumYear := artist.FirstAlbumYear()
		if albumYear > 0 {
			if minFirstAlbumYear == 0 || albumYear < minFirstAlbumYear {
				minFirstAlbumYear = albumYear
			}
			if albumYear > maxFirstAlbumYear {
				maxFirstAlbumYear = albumYear
			}
		}

		memberCount := artist.MemberCount()
		memberCountSet[memberCount] = true

		for _, country := range artist.Countries() {
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	memberCounts := make([]int, 0, len(memberCountSet))
	for count := range memberCountSet {
		memberCounts = append(memberCounts, count)
	}
	sort.Ints(memberCounts)

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	if minFirstAlbumYear == 0 {
		minFirstAlbumYear = minCreationYear
	}
	if maxFirstAlbumYear == 0 {
		maxFirstAlbumYear = maxCreationYear
	}

	return ArtistFilterOptions{
		CreationYear: IntRange{Min: minCreationYear, Max: maxCreationYear},
		FirstAlbum:   IntRange{Min: minFirstAlbumYear, Max: maxFirstAlbumYear},
		MemberCounts: memberCounts,
		Countries:    countries,
	}
}

// calculateLocationFilterOptions derives available location filter metadata.
func (s *Store) calculateLocationFilterOptions(locations []Location) LocationFilterOptions {
	if len(locations) == 0 {
		return LocationFilterOptions{}
	}

	minConcerts, maxConcerts := locations[0].TotalConcerts(), locations[0].TotalConcerts()
	minArtists, maxArtists := locations[0].ArtistCount(), locations[0].ArtistCount()
	minYear, maxYear := locations[0].YearRange()
	countrySet := make(map[string]bool)

	for _, location := range locations {
		totalConcerts := location.TotalConcerts()
		if totalConcerts < minConcerts {
			minConcerts = totalConcerts
		}
		if totalConcerts > maxConcerts {
			maxConcerts = totalConcerts
		}

		artistCount := location.ArtistCount()
		if artistCount < minArtists {
			minArtists = artistCount
		}
		if artistCount > maxArtists {
			maxArtists = artistCount
		}

		earliestYear, latestYear := location.YearRange()
		if earliestYear > 0 && earliestYear < minYear {
			minYear = earliestYear
		}
		if latestYear > maxYear {
			maxYear = latestYear
		}

		country := location.Country()
		if country != "" {
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return LocationFilterOptions{
		ConcertCount: IntRange{Min: minConcerts, Max: maxConcerts},
		ArtistCount:  IntRange{Min: minArtists, Max: maxArtists},
		YearRange:    IntRange{Min: minYear, Max: maxYear},
		Countries:    countries,
	}
}

// ============================================================================
// FILTER PARSING FROM HTTP REQUESTS
// ============================================================================

// ParseArtistFilters extracts artist filter criteria from HTTP request form data.
// Returns a value type with zero values for unset filters.
func ParseArtistFilters(r *http.Request) (ArtistFilters, error) {
	if err := r.ParseForm(); err != nil {
		return ArtistFilters{}, err
	}

	filters := ArtistFilters{
		CreationYear: parseIntRange(r, "creationYearFrom", "creationYearTo"),
		FirstAlbum:   parseIntRange(r, "firstAlbumYearFrom", "firstAlbumYearTo"),
		MemberCounts: parseIntSet(r, "memberCounts"),
		Countries:    parseStringSet(r, "countries"),
	}

	return filters, nil
}

// ParseLocationFilters extracts location filter criteria from HTTP request form data.
// Returns a value type with zero values for unset filters.
func ParseLocationFilters(r *http.Request) (LocationFilters, error) {
	if err := r.ParseForm(); err != nil {
		return LocationFilters{}, err
	}

	filters := LocationFilters{
		ConcertCount: parseIntRange(r, "concertCountFrom", "concertCountTo"),
		ArtistCount:  parseIntRange(r, "artistCountFrom", "artistCountTo"),
		YearRange:    parseIntRange(r, "concertYearFrom", "concertYearTo"),
		Countries:    parseStringSet(r, "countries"),
	}

	return filters, nil
}

// parseIntRange extracts min and max values from form fields and returns an IntRange.
// Returns zero IntRange if both fields are empty or invalid.
func parseIntRange(r *http.Request, minField, maxField string) IntRange {
	minStr := r.FormValue(minField)
	maxStr := r.FormValue(maxField)

	var rng IntRange

	if minStr != "" {
		if val, err := strconv.Atoi(minStr); err == nil {
			rng.Min = val
		}
	}

	if maxStr != "" {
		if val, err := strconv.Atoi(maxStr); err == nil {
			rng.Max = val
		}
	}

	return rng
}

// parseIntSet extracts multiple checkbox values into an IntSet.
// Returns empty set if no valid values found.
func parseIntSet(r *http.Request, fieldName string) IntSet {
	values := r.Form[fieldName]
	if len(values) == 0 {
		return IntSet{}
	}

	set := make(IntSet, len(values))
	for _, valueStr := range values {
		if val, err := strconv.Atoi(valueStr); err == nil {
			set[val] = struct{}{}
		}
	}

	return set
}

// parseStringSet extracts multiple form values into a StringSet.
// Returns empty set if no values found.
func parseStringSet(r *http.Request, fieldName string) StringSet {
	values := r.Form[fieldName]
	if len(values) == 0 {
		return StringSet{}
	}

	set := make(StringSet, len(values))
	for _, val := range values {
		if val != "" {
			set[val] = struct{}{}
		}
	}

	return set
}
