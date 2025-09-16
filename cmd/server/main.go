// Package main is the entry point for the Groupie Tracker server application.
package main

import "log"

func main() {
	log.Println("Starting Groupie Tracker server...")

	server, err := newServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := start(server); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
