package search

import (
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/data"
)

func buildSuggestions(artists []data.Artist) []Suggestion {
	if len(artists) == 0 {
		return nil
	}

	suggestions := make([]Suggestion, 0)
	seen := make(map[string]struct{})

	record := func(key string, suggestion Suggestion) {
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		suggestions = append(suggestions, suggestion)
	}

	for _, artist := range artists {
		artistURL := "/artists/" + artist.Slug

		record(normalize(artist.Name), Suggestion{
			Text:        artist.Name,
			Type:        SuggestionArtist,
			Description: "Artist",
			URL:         artistURL,
			ArtistID:    artist.ID,
		})

		for _, member := range artist.Members {
			record(normalize(member), Suggestion{
				Text:        member,
				Type:        SuggestionMember,
				Description: "Band member of " + artist.Name,
				URL:         artistURL,
				ArtistID:    artist.ID,
			})
		}

		for _, country := range artist.Countries {
			record(normalize(country), Suggestion{
				Text:        country,
				Type:        SuggestionLocation,
				Description: "Concert location",
				URL:         "/search?q=" + country,
			})
		}

		for _, concert := range artist.Concerts {
			if concert.Location == "" {
				continue
			}
			display := formatLocationName(concert.Location)
			record(normalize(concert.Location), Suggestion{
				Text:        display,
				Type:        SuggestionLocation,
				Description: "Concert venue",
				URL:         "/search?q=" + concert.Location,
			})
		}

		if artist.CreationYear > 0 {
			year := strconv.Itoa(artist.CreationYear)
			record(year, Suggestion{
				Text:        year,
				Type:        SuggestionCreation,
				Description: "Formation year",
				URL:         "/search?q=" + year,
			})
		}

		if artist.FirstAlbum != "" {
			key := normalize(artist.FirstAlbum)
			record(key, Suggestion{
				Text:        artist.FirstAlbum,
				Type:        SuggestionFirstAlbum,
				Description: "First album date",
				URL:         artistURL,
				ArtistID:    artist.ID,
			})
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

func trimSuggestions(suggestions []Suggestion, limit int) []Suggestion {
	if limit <= 0 || len(suggestions) <= limit {
		return append([]Suggestion(nil), suggestions...)
	}
	return append([]Suggestion(nil), suggestions[:limit]...)
}

func formatLocationName(location string) string {
	parts := strings.Split(location, "-")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}
