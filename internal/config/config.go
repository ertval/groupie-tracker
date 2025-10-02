package config

import "time"

// Server and API configuration defaults. Tests can override these values.
var (
	// Enable or disable image caching (true = enabled, false = disabled)
	WithCache = false

	// API base URL used by the data layer when fetching data
	APIBaseURL = "https://groupietrackers.herokuapp.com"

	// Request timeout for API calls
	APIRequestTimeout = 30 * time.Second

	// HTTP server defaults (port and timeouts)
	DefaultPort  = ":8080"
	ReadTimeout  = 15 * time.Second
	WriteTimeout = 15 * time.Second
	IdleTimeout  = 60 * time.Second
)
