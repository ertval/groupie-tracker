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

// cacheImages downloads and caches artist images locally when caching is enabled.
// Returns whether caching was enabled along with cached/downloaded image counts.
func (s *Store) cacheImages(artists []Artist) (bool, int, int) {
	if !s.withCache {
		return false, 0, 0
	}

	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return false, 0, 0
	}

	// Adaptive worker count: scale with CPU cores, but cap at artist count
	numWorkers := runtime.NumCPU()
	if numWorkers > len(artists) {
		numWorkers = len(artists)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Job represents a download task
	type job struct {
		index     int
		artist    *Artist
		fileName  string
		filePath  string
		localPath string
		exists    bool
	}

	// Create job queue
	jobs := make(chan job, len(artists))

	// Prepare all jobs
	for i := range artists {
		artist := &artists[i]
		fileName := fmt.Sprintf("%s.jpg", artist.Slug)
		filePath := filepath.Join(cacheDir, fileName)
		localPath := "/" + filepath.ToSlash(filePath)
		exists := false

		// Check if file already exists
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
	close(jobs)

	// Atomic counters for thread-safe counting
	var cached, downloaded int32
	var mu sync.Mutex // Mutex for updating artist images

	// Start adaptive worker pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if j.exists {
					// Use cached file
					mu.Lock()
					j.artist.Image = j.localPath
					mu.Unlock()
					atomic.AddInt32(&cached, 1)
				} else {
					// Download image with timeout
					if downloadImage(j.artist.Image, j.filePath) {
						mu.Lock()
						j.artist.Image = j.localPath
						mu.Unlock()
						atomic.AddInt32(&downloaded, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	return true, int(atomic.LoadInt32(&cached)), int(atomic.LoadInt32(&downloaded))
}

// downloadImage downloads and saves a single image from a URL to local filesystem.
// Uses a 10-second timeout to prevent hanging on slow/dead URLs.
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

// getCachedSearchResults retrieves cached search results for a query.
func (s *Store) getCachedSearchResults(query string) ([]Artist, bool) {
	s.searchCacheMu.Lock()
	defer s.searchCacheMu.Unlock()

	results, ok := s.searchCache[query]
	if !ok {
		return nil, false
	}

	s.moveKeyToEndLocked(query)
	return results, true
}

// setCachedSearchResults stores search results in the cache with LRU eviction.
func (s *Store) setCachedSearchResults(query string, results []Artist) {
	s.searchCacheMu.Lock()
	defer s.searchCacheMu.Unlock()

	if s.searchCache == nil {
		s.searchCache = make(map[string][]Artist, s.searchCacheSize)
	}
	if s.searchCacheSize <= 0 {
		s.searchCacheSize = 50
	}

	if _, exists := s.searchCache[query]; exists {
		s.searchCache[query] = results
		s.moveKeyToEndLocked(query)
		return
	}

	if len(s.searchOrder) >= s.searchCacheSize {
		oldest := s.searchOrder[0]
		delete(s.searchCache, oldest)
		s.searchOrder = s.searchOrder[1:]
	}

	s.searchCache[query] = results
	s.searchOrder = append(s.searchOrder, query)
}

// moveKeyToEndLocked moves a key to the end of the LRU order (must be called with lock held).
func (s *Store) moveKeyToEndLocked(query string) {
	for i, key := range s.searchOrder {
		if key == query {
			if i == len(s.searchOrder)-1 {
				return
			}
			copy(s.searchOrder[i:], s.searchOrder[i+1:])
			s.searchOrder[len(s.searchOrder)-1] = query
			return
		}
	}

	s.searchOrder = append(s.searchOrder, query)
}
