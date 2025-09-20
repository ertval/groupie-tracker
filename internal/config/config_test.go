package config

import (
	"testing"
	"time"
)

func TestDefaultValues(t *testing.T) {
	// Test that default values are as expected
	if WithCache != false {
		t.Errorf("expected WithCache to be false by default, got %t", WithCache)
	}

	if APIBaseURL != "https://groupietrackers.herokuapp.com" {
		t.Errorf("expected APIBaseURL to be 'https://groupietrackers.herokuapp.com', got %s", APIBaseURL)
	}

	if APIRequestTimeout != 30*time.Second {
		t.Errorf("expected APIRequestTimeout to be 30s, got %v", APIRequestTimeout)
	}

	if DefaultPort != ":8080" {
		t.Errorf("expected DefaultPort to be ':8080', got %s", DefaultPort)
	}

	if ReadTimeout != 15*time.Second {
		t.Errorf("expected ReadTimeout to be 15s, got %v", ReadTimeout)
	}

	if WriteTimeout != 15*time.Second {
		t.Errorf("expected WriteTimeout to be 15s, got %v", WriteTimeout)
	}

	if IdleTimeout != 60*time.Second {
		t.Errorf("expected IdleTimeout to be 60s, got %v", IdleTimeout)
	}
}

func TestConfigModification(t *testing.T) {
	// Save original values
	originalWithCache := WithCache
	originalAPIBaseURL := APIBaseURL
	originalAPIRequestTimeout := APIRequestTimeout
	originalDefaultPort := DefaultPort
	originalReadTimeout := ReadTimeout
	originalWriteTimeout := WriteTimeout
	originalIdleTimeout := IdleTimeout

	// Test modification of all config values
	WithCache = true
	if WithCache != true {
		t.Error("failed to modify WithCache")
	}

	APIBaseURL = "https://test-api.com"
	if APIBaseURL != "https://test-api.com" {
		t.Error("failed to modify APIBaseURL")
	}

	APIRequestTimeout = 10 * time.Second
	if APIRequestTimeout != 10*time.Second {
		t.Error("failed to modify APIRequestTimeout")
	}

	DefaultPort = ":3000"
	if DefaultPort != ":3000" {
		t.Error("failed to modify DefaultPort")
	}

	ReadTimeout = 30 * time.Second
	if ReadTimeout != 30*time.Second {
		t.Error("failed to modify ReadTimeout")
	}

	WriteTimeout = 30 * time.Second
	if WriteTimeout != 30*time.Second {
		t.Error("failed to modify WriteTimeout")
	}

	IdleTimeout = 120 * time.Second
	if IdleTimeout != 120*time.Second {
		t.Error("failed to modify IdleTimeout")
	}

	// Restore original values
	WithCache = originalWithCache
	APIBaseURL = originalAPIBaseURL
	APIRequestTimeout = originalAPIRequestTimeout
	DefaultPort = originalDefaultPort
	ReadTimeout = originalReadTimeout
	WriteTimeout = originalWriteTimeout
	IdleTimeout = originalIdleTimeout
}

func TestConfigTypes(t *testing.T) {
	// Test that config variables have the correct types
	var testBool bool = WithCache
	var testString string = APIBaseURL
	var testDuration time.Duration = APIRequestTimeout
	var testPort string = DefaultPort
	var testReadTimeout time.Duration = ReadTimeout
	var testWriteTimeout time.Duration = WriteTimeout
	var testIdleTimeout time.Duration = IdleTimeout

	// Use variables to prevent unused variable errors
	_ = testBool
	_ = testString
	_ = testDuration
	_ = testPort
	_ = testReadTimeout
	_ = testWriteTimeout
	_ = testIdleTimeout
}

func TestConfigValues_EdgeCases(t *testing.T) {
	// Test edge cases and boundary values
	originalTimeout := APIRequestTimeout

	// Test zero timeout
	APIRequestTimeout = 0
	if APIRequestTimeout != 0 {
		t.Error("failed to set zero timeout")
	}

	// Test negative timeout (unusual but possible)
	APIRequestTimeout = -1 * time.Second
	if APIRequestTimeout != -1*time.Second {
		t.Error("failed to set negative timeout")
	}

	// Test very large timeout
	APIRequestTimeout = 24 * time.Hour
	if APIRequestTimeout != 24*time.Hour {
		t.Error("failed to set large timeout")
	}

	// Restore original value
	APIRequestTimeout = originalTimeout

	// Test empty URL
	originalURL := APIBaseURL
	APIBaseURL = ""
	if APIBaseURL != "" {
		t.Error("failed to set empty URL")
	}
	APIBaseURL = originalURL

	// Test empty port
	originalPort := DefaultPort
	DefaultPort = ""
	if DefaultPort != "" {
		t.Error("failed to set empty port")
	}
	DefaultPort = originalPort
}

func TestConfigConcurrentAccess(t *testing.T) {
	// Test that config variables can be safely accessed from multiple goroutines
	// This is a basic test since Go's race detector would catch real issues
	originalValue := APIBaseURL

	done := make(chan bool, 2)

	// Goroutine 1: reads config
	go func() {
		for i := 0; i < 100; i++ {
			_ = APIBaseURL
			_ = WithCache
			_ = APIRequestTimeout
		}
		done <- true
	}()

	// Goroutine 2: reads config
	go func() {
		for i := 0; i < 100; i++ {
			_ = DefaultPort
			_ = ReadTimeout
			_ = WriteTimeout
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify value hasn't changed
	if APIBaseURL != originalValue {
		t.Error("APIBaseURL changed during concurrent access")
	}
}
