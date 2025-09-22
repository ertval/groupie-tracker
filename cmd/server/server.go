package main

import (
	"net/http"

	"groupie-tracker/internal/server"
)

// newServer creates and initializes a new HTTP server.
func newServer() (*http.Server, error) {
	return server.NewServer()
}
