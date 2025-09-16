package main

import (
"testing"
"time"

"groupie-tracker/internal/app"
)

func TestStore(t *testing.T) {
store := app.NewStore("http://test", time.Second)
if store == nil {
t.Error("Store should not be nil")
}
}
