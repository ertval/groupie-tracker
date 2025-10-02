// Package main is the entry point for the Groupie Tracker server application.
// Initializes the API client, creates the web server with dependency injection, and starts listening for HTTP requests.
package main

import (
	"log"
	"net/http"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/conf"
	"groupie-tracker/internal/web"
)

func main() {
	log.Println("Starting Groupie Tracker server...")

	// Initialize API client with configured base URL and request timeout
	// Timeout prevents hanging on slow/dead API responses during startup
	apiClient := api.NewClient(conf.APIBaseURL, conf.APIRequestTimeout)

	// Create and initialize the web server with injected dependencies
	// Server constructor loads all data from API, builds indexes, and compiles templates
	// WithCache flag enables/disables local image caching (scales with CPU cores if enabled)
	app, err := web.NewApp(apiClient, conf.WithCache)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err) // Fatal error prevents startup if data loading fails
	}

	// Start HTTP server (blocking operation - runs until interrupt signal or fatal error)
	// http.ErrServerClosed is expected during graceful shutdown, so we don't log it as fatal
	err = app.StartApp()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
