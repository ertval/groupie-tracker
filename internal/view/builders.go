package view

import (
	"fmt"
	"time"

	"groupie-tracker/internal/data"
)

// NewHomePage creates a home page view with featured artists and stats.
func NewHomePage(store *data.Store, featuredArtists []*data.Artist) HomePage {
	stats := store.Stats()
	suggestions := store.GenerateAllSearchSuggestions()

	return HomePage{
		Page: Page{
			Title:       "Home",
			ExtraCSS:    "home.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Artists:        featuredArtists,
		TotalMembers:   stats.TotalMembers,
		TotalLocations: stats.TotalLocations,
	}
}

// NewArtistListPage creates an artist listing page view.
func NewArtistListPage(
	store *data.Store,
	artists []*data.Artist,
	filterOptions data.ArtistFilterOptions,
	appliedFilters data.ArtistFilterParams,
	isFiltered bool,
	totalArtists int,
) ArtistListPage {
	suggestions := store.GenerateAllSearchSuggestions()

	return ArtistListPage{
		Page: Page{
			Title:       "Artists",
			ExtraCSS:    "artists.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Artists:        artists,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsFiltered:     isFiltered,
		TotalArtists:   totalArtists,
	}
}

// NewArtistDetailPage creates an artist detail page view.
func NewArtistDetailPage(
	store *data.Store,
	artist *data.Artist,
	prevArtist *data.Artist,
	nextArtist *data.Artist,
) ArtistDetailPage {
	suggestions := store.GenerateAllSearchSuggestions()

	return ArtistDetailPage{
		Page: Page{
			Title:       artist.Name,
			ExtraCSS:    "artist_detail.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Artist:     artist,
		PrevArtist: prevArtist,
		NextArtist: nextArtist,
	}
}

// NewLocationListPage creates a location listing page view.
func NewLocationListPage(
	store *data.Store,
	locations []data.Location,
	filterOptions data.LocationFilterOptions,
	appliedFilters data.LocationFilterParams,
	isFiltered bool,
	filterDescription string,
	totalLocations int,
) LocationListPage {
	stats := store.Stats()
	suggestions := store.GenerateAllSearchSuggestions()

	return LocationListPage{
		Page: Page{
			Title:       "Locations",
			ExtraCSS:    "locations.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Locations:         locations,
		FilterOptions:     filterOptions,
		AppliedFilters:    appliedFilters,
		IsFiltered:        isFiltered,
		FilterDescription: filterDescription,
		TotalLocations:    totalLocations,
		TotalCountries:    stats.TotalCountries,
		TotalConcerts:     stats.TotalConcerts,
	}
}

// NewLocationDetailPage creates a location detail page view.
func NewLocationDetailPage(
	store *data.Store,
	location data.Location,
	artists []data.ArtistAtLocation,
) LocationDetailPage {
	suggestions := store.GenerateAllSearchSuggestions()

	return LocationDetailPage{
		Page: Page{
			Title:       fmt.Sprintf("%s - Location", location.Name),
			ExtraCSS:    "location_detail.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Location:     location,
		Artists:      artists,
		PrevLocation: nil, // Could be implemented later for location navigation
		NextLocation: nil, // Could be implemented later for location navigation
	}
}

// NewSearchPage creates a search page view.
func NewSearchPage(
	store *data.Store,
	query string,
	results data.SearchResult,
	filterOptions data.ArtistFilterOptions,
	appliedFilters data.ArtistFilterParams,
	isSearch bool,
) SearchPage {
	suggestions := store.GenerateAllSearchSuggestions()

	return SearchPage{
		Page: Page{
			Title:       "Search",
			ExtraCSS:    "search.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Query:          query,
		Results:        results,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsSearch:       isSearch,
	}
}

// NewDevPage creates a developer tools page view.
func NewDevPage(store *data.Store) DevPage {
	suggestions := store.GenerateAllSearchSuggestions()

	links := []DevLink{
		{Href: "/dev/panic", Text: "Trigger Panic (/dev/panic)"},
		{Href: "/dev/404", Text: "Simulate 404 (/dev/404)"},
		{Href: "/dev/500", Text: "Simulate 500 (/dev/500)"},
		{Href: "/dev/tmpl-error", Text: "Simulate Template Error (/dev/tmpl-error)"},
		{Href: "/health", Text: "Health Check (/health)"},
	}

	return DevPage{
		Page: Page{
			Title:       "Developer Tools",
			ExtraCSS:    "dev.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Links: links,
	}
}

// NewErrorPage creates an error page view.
func NewErrorPage(status int, requestedURL string, message string) Page {
	return Page{
		Title:       fmt.Sprintf("%d %s", status, getStatusText(status)),
		ExtraCSS:    "errors.css",
		ExtraJS:     "",
		Suggestions: nil, // Error pages don't need search suggestions
		Error: &ErrorInfo{
			Code:         status,
			Message:      message,
			RequestedURL: requestedURL,
			Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		},
	}
}

// getStatusText returns HTTP status text for common codes.
func getStatusText(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	default:
		return "Error"
	}
}

// NewHealthResponse creates a health check response.
func NewHealthResponse(store *data.Store) HealthResponse {
	return HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Stats:     store.Stats(),
	}
}
