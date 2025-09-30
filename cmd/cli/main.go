// Package main is the entry point for the Groupie Tracker server application.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/search"
	"groupie-tracker/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.FromEnv()

	clientOpts := []api.Option{api.WithTimeout(cfg.HTTPTimeout)}
	if cfg.APIBaseURL != "" {
		clientOpts = append(clientOpts, api.WithBaseURL(cfg.APIBaseURL))
	}
	apiClient := api.NewClient(clientOpts...)

	loadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	store, err := data.Load(loadCtx, apiClient)
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}

	searchService := search.NewService(store)

	appServer, err := server.New(store, searchService, cfg)
	if err != nil {
		log.Fatalf("failed to build server: %v", err)
	}

	if err := appServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
