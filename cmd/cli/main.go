// Package main is the entry point for the Groupie Tracker server application.
package main

import (
	"fmt"
	"log"
	"net/http"

	"groupie-tracker/internal/server"
)

func main() {
	log.Println("Starting Groupie Tracker server...")

	server, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server (blocking)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", fmt.Errorf("server failed to start: %w", err))
	}
}
