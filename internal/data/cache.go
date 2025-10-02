package data

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// cacheImages downloads and caches artist images locally using an adaptive worker pool.
// Worker count scales with CPU cores (runtime.NumCPU) for optimal concurrency on any system.
// Returns: (cacheEnabled, cachedImageCount, downloadedImageCount)
// - cacheEnabled: false if cache directory creation fails, otherwise true
// - cachedImageCount: number of images already present on disk (skipped download)
// - downloadedImageCount: number of images successfully downloaded in this run
func (s *Store) cacheImages(artists []*Artist) (bool, int, int) {
	if !s.withCache { // Early return if caching disabled in config
		return false, 0, 0
	}

	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil { // Create cache directory with proper permissions
		return false, 0, 0 // Directory creation failed, disable caching
	}

	// Adaptive worker pool: scales with CPU cores for optimal concurrency
	// On a 12-core system, we get 12 parallel downloads instead of fixed 4 workers
	numWorkers := runtime.NumCPU()
	if numWorkers > len(artists) { // Don't create more workers than artists to download
		numWorkers = len(artists)
	}
	if numWorkers < 1 { // Safety check: ensure at least 1 worker exists
		numWorkers = 1
	}

	// job represents a single image download/cache task
	type job struct {
		index     int     // Artist index for updating the slice
		artist    *Artist // Pointer to artist for updating image path
		fileName  string  // Target filename (e.g., "queen.jpg")
		filePath  string  // Full filesystem path (e.g., "static/img/artists/queen.jpg")
		localPath string  // URL path for HTML (e.g., "/static/img/artists/queen.jpg")
		exists    bool    // Whether file already exists on disk (skip download)
	}

	// Create buffered channel for all download jobs
	jobs := make(chan job, len(artists))

	// Prepare all jobs upfront (producer phase)
	for i, artist := range artists {
		fileName := fmt.Sprintf("%s.jpg", artist.Slug()) // Convert artist slug to filename
		filePath := filepath.Join(cacheDir, fileName)    // Full path on disk
		localPath := "/" + filepath.ToSlash(filePath)    // Convert to forward slashes for URLs
		exists := false

		// Check if file already exists to avoid unnecessary downloads
		if _, err := os.Stat(filePath); err == nil {
			exists = true
		}

		jobs <- job{
			index:     i,
			artist:    artist,
			fileName:  fileName,
			filePath:  filePath,
			localPath: localPath,
			exists:    exists,
		}
	}
	close(jobs) // Close channel to signal no more jobs coming

	// Atomic counters for thread-safe tracking across goroutines
	var cached, downloaded int32 // int32 required for atomic operations
	var mu sync.Mutex            // Mutex protects concurrent artist slice updates

	// Start adaptive worker pool (consumer phase)
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() { // Each worker goroutine processes jobs from the channel
			defer wg.Done()
			for j := range jobs { // Range over channel until it's closed
				if j.exists { // File already on disk, just update the path
					mu.Lock() // Protect concurrent writes to artists slice
					j.artist.Image = j.localPath
					mu.Unlock()
					atomic.AddInt32(&cached, 1) // Thread-safe increment
				} else { // File doesn't exist, download it
					if downloadImage(j.artist.Image, j.filePath) { // downloadImage handles HTTP GET with timeout
						mu.Lock() // Protect concurrent writes to artists slice
						j.artist.Image = j.localPath
						mu.Unlock()
						atomic.AddInt32(&downloaded, 1) // Thread-safe increment
					}
					// If download fails, original external URL remains in j.artist.Image
				}
			}
		}()
	}

	wg.Wait() // Block until all workers complete

	return true, int(atomic.LoadInt32(&cached)), int(atomic.LoadInt32(&downloaded)) // Return final counts (convert from int32 to int)
}

// downloadImage downloads and saves a single image from a URL to local filesystem.
// Uses a 10-second timeout to prevent hanging on slow/dead URLs, protecting startup time.
func downloadImage(url, path string) bool {
	if strings.TrimSpace(url) == "" {
		return false
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return false
	}
	defer resp.Body.Close()

	file, err := os.Create(path)
	if err != nil {
		return false
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err == nil
}
