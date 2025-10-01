// Package main is the entry point for the Groupie Tracker server application.
package main

import (
	"log"
	"net/http"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/web"
)

func main() {
	log.Println("Starting Groupie Tracker server...")

	// Create API client
	apiClient := api.NewClient(config.APIBaseURL, config.APIRequestTimeout)

	// Create server with injected dependencies
	server, err := web.NewServer(apiClient, config.WithCache)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server (blocking)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
