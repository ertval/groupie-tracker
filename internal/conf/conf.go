package conf

import "time"

// Server and API configuration defaults. Tests can override these values.
var (
	// API base URL used by the data layer when fetching data
	APIBaseURL = "https://groupietrackers.herokuapp.com"

	// Request timeout for API calls
	APIRequestTimeout = 30 * time.Second

	// HTTP server defaults (port and timeouts)
	DefaultPort  = ":8080"
	ReadTimeout  = 15 * time.Second
	WriteTimeout = 15 * time.Second
	IdleTimeout  = 60 * time.Second

	// Data refresh interval (default: 1 hour)
	// Set to a shorter duration for testing (e.g., 1 * time.Minute)
	DataRefreshInterval = 1 * time.Hour

	// Rate limiter defaults (per-client)
	// Suggestion/search endpoints should be more restrictive in production.
	RateLimitRequestsPerSecond = 5  // tokens added per second
	RateLimitBurst             = 10 // maximum bucket size
)
