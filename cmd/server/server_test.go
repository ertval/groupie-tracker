package main

import (
	"testing"
	"time"

	"groupie-tracker/internal/repository"
)

func TestStore(t *testing.T) {
	store := repository.NewRepository("http://test", time.Second)
	if store == nil {
		t.Error("Store should not be nil")
	}
}
