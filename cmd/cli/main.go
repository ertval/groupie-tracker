// Package main is the entry point for the Groupie Tracker server application.
package main

import (
	"log"

	"groupie-tracker/internal/data"
)

func main() {
	// Initialize server with data loading and caching
	if err := data.InitializeServer(); err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Create and start HTTP server
	server := data.CreateServer()
	
	log.Printf("Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
