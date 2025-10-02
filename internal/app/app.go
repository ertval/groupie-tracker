package app

import (
	"context"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/service"
)

// Initialize wires the data store and service, loading all necessary data up front.
func Initialize(ctx context.Context, apiClient *api.Client, withCache bool) (*data.Store, *service.Service, error) {
	store := data.NewStore(apiClient, withCache)
	svc := service.New(store)

	if err := svc.Load(ctx); err != nil {
		return nil, nil, err
	}

	return store, svc, nil
}
