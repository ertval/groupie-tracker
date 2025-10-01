package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkFetchAPIData_Sequential simulates the old sequential approach
func BenchmarkFetchAPIData_Sequential(b *testing.B) {
	// Create a test server that simulates API delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate network latency
		time.Sleep(50 * time.Millisecond)

		if r.URL.Path == "/api/artists" {
			w.Write([]byte(`[{"id": 1, "name": "Test Artist", "members": ["Member1"], "creationDate": 2000, "firstAlbum": "01-01-2001", "image": "test.jpg"}]`))
		} else if r.URL.Path == "/api/relation" {
			w.Write([]byte(`{"index": [{"id": 1, "datesLocations": {"test-location": ["01-01-2020"]}}]}`))
		}
	}))
	defer server.Close()

	// Create repository with test server
	repo := &Repository{
		apiEndpoint: server.URL,
		apiClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, _, err := repo.fetchAPIData(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFetchAPIData_Concurrent measures our new concurrent implementation
func BenchmarkFetchAPIData_Concurrent(b *testing.B) {
	// Create a test server that simulates API delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate network latency
		time.Sleep(50 * time.Millisecond)

		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[{"id": 1, "name": "Test Artist", "members": ["Member1"], "creationDate": 2000, "firstAlbum": "01-01-2001", "image": "test.jpg"}]`))
		case "/api/relation":
			w.Write([]byte(`{"index": [{"id": 1, "datesLocations": {"test-location": ["01-01-2020"]}}]}`))
		}
	}))
	defer server.Close()

	// Create repository with test server
	repo := &Repository{
		apiEndpoint: server.URL,
		apiClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, _, err := repo.fetchAPIData(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestConcurrentAPIFetching verifies that both endpoints are called concurrently
func TestConcurrentAPIFetching(t *testing.T) {
	callTimes := make(map[string]time.Time)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callTimes[r.URL.Path] = time.Now()

		// Add small delay to make timing measurable
		time.Sleep(10 * time.Millisecond)

		if r.URL.Path == "/api/artists" {
			w.Write([]byte(`[{"id": 1, "name": "Test Artist", "members": ["Member1"], "creationDate": 2000, "firstAlbum": "01-01-2001", "image": "test.jpg"}]`))
		} else if r.URL.Path == "/api/relation" {
			w.Write([]byte(`{"index": [{"id": 1, "datesLocations": {"test-location": ["01-01-2020"]}}]}`))
		}
	}))
	defer server.Close()

	repo := &Repository{
		apiEndpoint: server.URL,
		apiClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	ctx := context.Background()
	_, _, err := repo.fetchAPIData(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Verify both endpoints were called
	if len(callTimes) != 2 {
		t.Fatalf("Expected 2 API calls, got %d", len(callTimes))
	}

	artistsTime := callTimes["/api/artists"]
	relationsTime := callTimes["/api/relation"]

	// The calls should happen within a very short time window (concurrent)
	timeDiff := artistsTime.Sub(relationsTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	// They should be called within 5ms of each other (concurrent)
	if timeDiff > 5*time.Millisecond {
		t.Errorf("API calls were not concurrent. Time difference: %v", timeDiff)
	}

	t.Logf("API calls were concurrent. Time difference: %v", timeDiff)
}
