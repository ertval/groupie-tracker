# Hourly Data Rehydration - Implementation Plan

## Current State

The application currently:
- Fetches all data from the API at startup
- Caches artist images locally in `static/img/artists/`
- Stores all data in an immutable in-memory store
- Does **not** refresh data after initial load

## Requirements

Implement automatic data rehydration that:
1. Refreshes API data (artists, relations) every hour
2. Re-caches any new/updated images
3. Rebuilds the in-memory store with fresh data
4. Does not block incoming HTTP requests during refresh
5. Handles failures gracefully (keep serving old data on error)

## Implementation Plan

### 1. Add Ticker to Server (`internal/web/server.go`)

```go
type App struct {
	store      *data.Store
	templates  map[string]*template.Template
	httpServer *http.Server
	Handler    http.Handler
	
	// Add for data rehydration
	storeMu    sync.RWMutex          // Protects store during refresh
	ticker     *time.Ticker           // Triggers hourly refresh
	stopChan   chan struct{}          // Signals shutdown
}

func NewApp(apiClient *api.Client) (*App, error) {
	// ... existing initialization ...
	
	// Start background refresh
	app.startDataRefresh(apiClient)
	
	return app, nil
}

func (s *App) startDataRefresh(apiClient *api.Client) {
	s.ticker = time.NewTicker(1 * time.Hour)
	s.stopChan = make(chan struct{})
	
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.refreshData(apiClient)
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *App) refreshData(apiClient *api.Client) {
	log.Println("Starting hourly data refresh...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create new store and load fresh data
	newStore := data.NewStore(apiClient)
	if err := newStore.Load(ctx); err != nil {
		log.Printf("Data refresh failed: %v (keeping old data)", err)
		return
	}
	
	// Atomically swap stores
	s.storeMu.Lock()
	s.store = newStore
	s.storeMu.Unlock()
	
	stats := newStore.Stats()
	log.Printf("Data refresh complete - %d artists (cached: %d, downloaded: %d)",
		stats.TotalArtists, stats.CachedImages, stats.DownloadedImages)
}

func (s *App) getStore() *data.Store {
	s.storeMu.RLock()
	defer s.storeMu.RUnlock()
	return s.store
}

func (s *App) Shutdown(ctx context.Context) error {
	close(s.stopChan)
	s.ticker.Stop()
	return s.httpServer.Shutdown(ctx)
}
```

### 2. Update All Handlers to Use `getStore()`

Change all handlers from:
```go
artists := s.store.Artists()
```

To:
```go
artists := s.getStore().Artists()
```

This ensures handlers always read from the current store safely.

### 3. Make Store Thread-Safe for Reads During Refresh

The `Store` is already immutable after `Load()`, so concurrent reads are safe. However, we need to ensure the swap itself is atomic (handled by `storeMu`).

### 4. Add Configuration for Refresh Interval (`internal/conf/conf.go`)

```go
var (
	// ... existing config ...
	
	// Data refresh interval (default: 1 hour)
	DataRefreshInterval = 1 * time.Hour
)
```

### 5. Add Metrics/Monitoring

Track refresh statistics:
```go
type RefreshStats struct {
	LastRefresh     time.Time
	RefreshCount    int
	FailureCount    int
	LastError       error
	LastDuration    time.Duration
}
```

Expose via health endpoint:
```json
{
  "status": "healthy",
  "uptime": "2h30m",
  "last_refresh": "2025-10-03T10:30:00Z",
  "refresh_count": 3,
  "refresh_failures": 0
}
```

## Testing Strategy

### Unit Tests
1. Test `startDataRefresh()` with mock ticker
2. Test `refreshData()` with failing API client
3. Test `getStore()` concurrent access during swap
4. Test `Shutdown()` stops ticker and goroutine

### Integration Tests
1. Start server, wait for refresh, verify new data loaded
2. Simulate API failure during refresh, verify old data still served
3. Test concurrent requests during refresh (no races)

### Manual Testing
1. Set refresh interval to 1 minute for testing
2. Modify API mock to return different data each time
3. Verify logs show successful refreshes
4. Verify HTTP requests continue working during refresh

## Rollout Plan

### Phase 1: Add Infrastructure (Low Risk)
- Add `storeMu`, `ticker`, `stopChan` fields
- Add `getStore()` method
- Add `Shutdown()` method
- **No behavior change yet**

### Phase 2: Update Handlers (Low Risk)
- Change all handlers to use `getStore()`
- Run full test suite
- **Still no automatic refresh**

### Phase 3: Enable Refresh (Medium Risk)
- Uncomment `startDataRefresh()` call
- Add configuration for interval
- Add logging and metrics
- **Automatic refresh enabled**

### Phase 4: Monitoring & Tuning
- Monitor refresh success rate
- Tune timeout values
- Add alerting for consecutive failures

## Considerations

### Memory Usage
- Two stores in memory briefly during swap (~2x memory)
- Old store becomes eligible for GC after swap
- Consider memory limits on hosted environments

### API Rate Limiting
- Hourly refresh = 24 API calls/day
- Monitor API rate limits
- Add exponential backoff on failures

### Image Cache
- Existing images won't be re-downloaded (already on disk)
- New artists' images will be cached
- Consider adding cache cleanup for deleted artists

### Error Handling
- Failed refresh keeps serving old data (graceful degradation)
- Log all failures for monitoring
- Consider alerting after N consecutive failures

## Alternative: Manual Refresh Endpoint

Instead of automatic refresh, provide an admin endpoint:

```go
func (s *App) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Optional: Add authentication/authorization
	
	go s.refreshData(s.apiClient) // Async to not block response
	
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Data refresh started")
}
```

Benefits:
- Control over when refresh happens
- Simpler implementation
- No background goroutines
- Can trigger manually or via cron

## Recommendation

Start with **Manual Refresh Endpoint** for simplicity:
1. Easier to test and debug
2. No background goroutines to manage
3. Can trigger via cron job or script
4. Upgrade to automatic ticker later if needed

Then, if automatic refresh is required, implement the full ticker-based solution.

---

**Status**: 📝 Planning  
**Complexity**: Medium  
**Estimated Effort**: 4-6 hours  
**Priority**: Low (current behavior is functional)
