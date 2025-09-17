package main

import (
	"testing"
	"time"

	"groupie-tracker/internal/data"
)

func TestStore(t *testing.T) {
	store := data.NewRepository("http://test", time.Second)
	if store == nil {
		t.Error("Store should not be nil")
	}
}
