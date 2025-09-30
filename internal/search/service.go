package search

import "groupie-tracker/internal/data"

// NewService constructs a Service and precomputes suggestions for fast responses.
func NewService(provider ArtistProvider) *Service {
	s := &Service{provider: provider}
	if provider != nil {
		s.suggestions = buildSuggestions(provider.Artists())
	}
	return s
}

// Search executes a full text + filter search against the current dataset.
func (s *Service) Search(params Params) Result {
	artists := s.safeArtists()
	if len(artists) == 0 {
		return Result{Query: params.Query}
	}

	normalizedQuery := normalize(params.Query)
	matches := artists

	if normalizedQuery != "" {
		matches = matches[:0]
		for _, artist := range artists {
			if matchesQuery(artist, normalizedQuery) {
				matches = append(matches, artist)
			}
		}
	}

	if !params.Filters.IsEmpty() {
		matches = filterArtists(matches, params.Filters)
	}

	return Result{
		Artists:      matches,
		TotalResults: len(matches),
		Query:        params.Query,
	}
}

// FilterOptions builds the available filter ranges from the full dataset.
func (s *Service) FilterOptions() FilterOptions {
	return computeFilterOptions(s.safeArtists())
}

// Suggestions returns the full suggestion cache.
func (s *Service) Suggestions() []Suggestion {
	return append([]Suggestion(nil), s.suggestions...)
}

// Suggest filters the cached suggestions by query and trims the response to limit.
func (s *Service) Suggest(query string, limit int) []Suggestion {
	if len(s.suggestions) == 0 {
		return nil
	}

	normalized := normalize(query)
	if normalized == "" {
		return trimSuggestions(s.suggestions, limit)
	}

	filtered := make([]Suggestion, 0, len(s.suggestions))
	for _, suggestion := range s.suggestions {
		if stringsContains(normalize(suggestion.Text), normalized) {
			filtered = append(filtered, suggestion)
		}
	}

	return trimSuggestions(filtered, limit)
}

func (s *Service) safeArtists() []data.Artist {
	if s == nil || s.provider == nil {
		return nil
	}
	original := s.provider.Artists()
	clone := make([]data.Artist, len(original))
	copy(clone, original)
	return clone
}
