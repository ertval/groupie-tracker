package config

import "time"

// WithCache controls whether the repository should cache artist images locally.
// If true, the repository will attempt to download images into the local
// `static/img/artists` cache during data loading. If false, images will be
// left as provided by the API (no caching attempted).
//
// Default: true (preserves previous behavior).
var WithCache = false

// Server and API configuration defaults. Tests can override these values.
var (
	// API base URL used by the repository when fetching data
	APIBaseURL = "https://groupietrackers.herokuapp.com"

	// Request timeout for API calls
	APIRequestTimeout = 30 * time.Second

	// HTTP server defaults (port and timeouts)
	DefaultPort  = ":8080"
	ReadTimeout  = 15 * time.Second
	WriteTimeout = 15 * time.Second
	IdleTimeout  = 60 * time.Second
)
