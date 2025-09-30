package data

import (
	"sort"

	"groupie-tracker/internal/api"
)

func processArtists(apiArtists []api.Artist, apiRelations []api.RelationIndex) []Artist {
	relationByID := make(map[int]api.RelationIndex, len(apiRelations))
	for _, relation := range apiRelations {
		relationByID[relation.ID] = relation
	}

	artists := make([]Artist, 0, len(apiArtists))
	for _, dto := range apiArtists {
		artist := Artist{
			ID:           dto.ID,
			Name:         dto.Name,
			Slug:         slugify(dto.Name),
			Members:      append([]string(nil), dto.Members...),
			CreationYear: dto.CreationYear,
			FirstAlbum:   dto.FirstAlbum,
			Image:        dto.Image,
		}

		if relation, ok := relationByID[dto.ID]; ok {
			concerts, countries := buildConcerts(relation.DatesLocations)
			artist.Concerts = concerts
			artist.Countries = countries
			artist.ConcertCount = len(concerts)
		}

		artists = append(artists, artist)
	}

	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

func buildConcerts(datesLocations map[string][]string) ([]Concert, []string) {
	concerts := make([]Concert, 0)
	countrySet := make(map[string]struct{})

	for location, dates := range datesLocations {
		if location == "" {
			continue
		}

		country := CountryFromLocation(location)
		if country != "" {
			countrySet[country] = struct{}{}
		}

		for _, date := range dates {
			concerts = append(concerts, Concert{
				Location: location,
				Country:  country,
				Date:     date,
				Year:     YearFromDate(date),
			})
		}
	}

	sort.Slice(concerts, func(i, j int) bool {
		if concerts[i].Year == concerts[j].Year {
			return concerts[i].Date < concerts[j].Date
		}
		return concerts[i].Year < concerts[j].Year
	})

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return concerts, countries
}

func createLocations(artists []Artist) []Location {
	locationsByName := make(map[string]*Location)

	for _, artist := range artists {
		for _, concert := range artist.Concerts {
			name := concert.Location
			if name == "" {
				continue
			}

			loc, exists := locationsByName[name]
			if !exists {
				loc = &Location{
					Name:          name,
					Slug:          slugify(name),
					Artists:       []string{artist.Name},
					ArtistCount:   1,
					TotalConcerts: 1,
					Concerts:      []Concert{concert},
					EarliestYear:  concert.Year,
					LatestYear:    concert.Year,
				}
				locationsByName[name] = loc
				continue
			}

			loc.TotalConcerts++
			loc.Concerts = append(loc.Concerts, concert)

			year := concert.Year
			if year > 0 {
				if loc.EarliestYear == 0 || year < loc.EarliestYear {
					loc.EarliestYear = year
				}
				if year > loc.LatestYear {
					loc.LatestYear = year
				}
			}

			if !containsString(loc.Artists, artist.Name) {
				loc.Artists = append(loc.Artists, artist.Name)
				loc.ArtistCount = len(loc.Artists)
			}
		}
	}

	locations := make([]Location, 0, len(locationsByName))
	for _, location := range locationsByName {
		locations = append(locations, *location)
	}

	sort.Slice(locations, func(i, j int) bool {
		if locations[i].TotalConcerts == locations[j].TotalConcerts {
			return locations[i].Name < locations[j].Name
		}
		return locations[i].TotalConcerts > locations[j].TotalConcerts
	})

	return locations
}

func calculateStats(artists []Artist, locations []Location) AppStats {
	stats := AppStats{
		TotalArtists:   len(artists),
		TotalLocations: len(locations),
	}

	for _, artist := range artists {
		stats.TotalConcerts += artist.ConcertCount

		for _, concert := range artist.Concerts {
			year := concert.Year
			if year == 0 {
				continue
			}
			if stats.EarliestYear == 0 || year < stats.EarliestYear {
				stats.EarliestYear = year
			}
			if year > stats.LatestYear {
				stats.LatestYear = year
			}
		}
	}

	return stats
}
