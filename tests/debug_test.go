package tests

import (
	"testing"
)

func TestDebugServerRunning(t *testing.T) {
	t.Log("Testing isServerRunning function...")

	url := "http://localhost:8080"
	if isServerRunning(url) {
		t.Logf("✅ Server is running at %s", url)
	} else {
		t.Logf("❌ Server is NOT running at %s", url)
	}
}
