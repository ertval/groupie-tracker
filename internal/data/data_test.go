package data

import "testing"

func TestBasic(t *testing.T) {
	repo := NewRepository()
	if repo == nil {
		t.Error("NewRepository() returned nil")
	}
}