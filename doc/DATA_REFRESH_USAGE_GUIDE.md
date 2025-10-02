# Data Refresh Feature - Usage Guide

## Overview

The Groupie Tracker server now includes automatic and manual data refresh capabilities to keep the application data up-to-date without restarting the server.

## Features

### 1. Automatic Hourly Refresh

- **Default Interval**: 1 hour (configurable)
- **Behavior**: Background goroutine refreshes data from API every hour
- **Thread-Safe**: Uses RWMutex to ensure safe concurrent access during refresh
- **Graceful Failure**: On error, keeps serving old data and logs warning
- **Image Re-caching**: Automatically caches new/updated artist images

### 2. Manual Refresh Endpoint

- **Endpoint**: `POST /api/refresh`
- **Response**: 202 Accepted (async operation)
- **Use Case**: Trigger immediate refresh without waiting for hourly timer
- **Admin Feature**: Can be called manually or via cron/automation

## Usage

### Automatic Refresh (Default)

The server automatically refreshes data every hour. No action required!

**Server logs show:**
```
2025/10/03 01:32:52 Data refresh scheduled every 1h0m0s
2025/10/03 02:32:52 Starting scheduled data refresh...
2025/10/03 02:32:53 ✅ Data refresh complete - 52 artists (cached: 52, downloaded: 0)
```

### Manual Refresh

#### Using curl
```bash
curl -X POST http://localhost:8080/api/refresh
```

**Success Response (202 Accepted):**
```json
{
  "status": "accepted",
  "message": "Data refresh started. Check server logs for progress."
}
```

**Error Response (405 Method Not Allowed) - Wrong HTTP method:**
```json
{
  "error": "Method not allowed. Use POST to trigger refresh."
}
```

#### Using PowerShell
```powershell
Invoke-WebRequest -Method POST -Uri "http://localhost:8080/api/refresh"
```

#### Using JavaScript (fetch)
```javascript
fetch('http://localhost:8080/api/refresh', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  }
})
.then(response => response.json())
.then(data => console.log('Refresh started:', data))
.catch(error => console.error('Error:', error));
```

## Configuration

### Change Refresh Interval

Edit `internal/conf/conf.go`:

```go
var (
    // ... other config ...
    
    // Change to 30 minutes
    DataRefreshInterval = 30 * time.Minute
    
    // Or 2 hours
    DataRefreshInterval = 2 * time.Hour
    
    // Or 5 minutes for testing
    DataRefreshInterval = 5 * time.Minute
)
```

After changing, rebuild and restart:
```bash
go build -o bin/server.exe cmd/server/main.go
./bin/server.exe
```

## Monitoring

### Check Logs

The server logs all refresh activity:

**Startup:**
```
Data refresh scheduled every 1h0m0s
```

**Successful Refresh:**
```
Starting scheduled data refresh...
✅ Data refresh complete - 52 artists (cached: 52 images, downloaded: 0 images)
```

**Failed Refresh:**
```
Starting scheduled data refresh...
⚠️  Data refresh failed: context deadline exceeded (keeping old data)
```

### Health Check

The `/health` endpoint shows server status:
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "total_artists": 52,
  "total_locations": 120,
  "total_members": 149,
  "total_concerts": 1038,
  "cached_images": 52,
  "downloaded_images": 0
}
```

## Testing

### Test Automatic Refresh

1. **Set short interval for testing:**
   ```go
   // In internal/conf/conf.go
   DataRefreshInterval = 1 * time.Minute
   ```

2. **Rebuild and run:**
   ```bash
   go build -o bin/server.exe cmd/server/main.go
   ./bin/server.exe
   ```

3. **Watch logs for refreshes every minute**

### Test Manual Refresh

1. **Start server:**
   ```bash
   ./bin/server.exe
   ```

2. **Trigger manual refresh:**
   ```bash
   curl -X POST http://localhost:8080/api/refresh
   ```

3. **Check logs for refresh confirmation**

### Test Error Handling

1. **Kill the API server or disconnect from internet**

2. **Trigger refresh:**
   ```bash
   curl -X POST http://localhost:8080/api/refresh
   ```

3. **Verify:**
   - Server logs show warning
   - Old data still served
   - Application continues working

## Performance Considerations

### Memory Usage

- **During Refresh**: ~2x memory (old store + new store)
- **After Refresh**: Returns to 1x (old store garbage collected)
- **Impact**: Brief spike, no long-term increase

### CPU Usage

- **During Refresh**: Brief spike for data processing
- **Impact**: Minimal, happens in background goroutine

### HTTP Response Times

- **During Refresh**: No impact on request handling
- **Reason**: Atomic store swap with RWMutex ensures requests always get a complete store

### API Calls

- **Hourly Refresh**: 24 API calls per day
- **Manual Refresh**: As needed (be mindful of API rate limits)

## Security Recommendations

### Production Deployment

The `/api/refresh` endpoint currently has **no authentication**. For production:

#### 1. Add API Key Authentication

```go
func (app *App) RefreshData(w http.ResponseWriter, r *http.Request) {
    // Check API key
    apiKey := r.Header.Get("X-API-Key")
    if apiKey != os.Getenv("ADMIN_API_KEY") {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // ... rest of handler
}
```

Usage:
```bash
curl -X POST \
  -H "X-API-Key: your-secret-key" \
  http://localhost:8080/api/refresh
```

#### 2. Rate Limiting

Implement rate limiting to prevent abuse:
```go
var refreshRateLimiter = rate.NewLimiter(rate.Every(1*time.Minute), 5)

func (app *App) RefreshData(w http.ResponseWriter, r *http.Request) {
    if !refreshRateLimiter.Allow() {
        http.Error(w, "Too many requests", http.StatusTooManyRequests)
        return
    }
    
    // ... rest of handler
}
```

#### 3. IP Whitelisting

Restrict to admin IPs only:
```go
var allowedIPs = []string{"192.168.1.100", "10.0.0.5"}

func (app *App) RefreshData(w http.ResponseWriter, r *http.Request) {
    clientIP := r.RemoteAddr
    if !contains(allowedIPs, clientIP) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // ... rest of handler
}
```

## Troubleshooting

### Problem: Refresh fails with timeout

**Symptoms:**
```
⚠️  Data refresh failed: context deadline exceeded (keeping old data)
```

**Solution:**
1. Check internet connection
2. Verify API is accessible: `curl https://groupietrackers.herokuapp.com/api/artists`
3. Increase timeout in `internal/web/server.go`:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
   ```

### Problem: Memory usage keeps growing

**Symptoms:** Memory usage increases after each refresh

**Solution:**
1. Check for memory leaks with pprof:
   ```bash
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```
2. Ensure old store is being garbage collected
3. Monitor with: `go tool pprof -alloc_space http://localhost:8080/debug/pprof/heap`

### Problem: Manual refresh returns 404

**Symptoms:**
```bash
curl -X POST http://localhost:8080/api/refresh
# 404 Not Found
```

**Solution:**
1. Verify server is running: `curl http://localhost:8080/health`
2. Check route is registered in `internal/web/routes.go`
3. Ensure using POST method, not GET

### Problem: Refresh happens too frequently

**Solution:**
Change interval in `internal/conf/conf.go`:
```go
DataRefreshInterval = 2 * time.Hour  // Reduce frequency
```

## Best Practices

### 1. Set Appropriate Refresh Interval

- **High-traffic sites**: Longer intervals (2-4 hours) to reduce API load
- **Low-traffic sites**: Default (1 hour) is fine
- **Development**: Short intervals (1-5 minutes) for testing

### 2. Monitor Refresh Success

- Set up logging aggregation (e.g., ELK stack, Datadog)
- Alert on consecutive failures (3+ in a row)
- Track refresh duration over time

### 3. Handle Failures Gracefully

- Server already keeps serving old data on failure ✓
- Consider adding exponential backoff for retries
- Alert ops team on persistent failures

### 4. Secure the Manual Endpoint

- Add authentication (API key, JWT)
- Rate limit requests
- Log all manual refresh attempts
- Consider IP whitelisting

### 5. Test Before Production

- Test with short intervals in staging
- Verify memory usage is stable
- Ensure refresh doesn't impact request latency
- Test failure scenarios (API down, timeout)

## Integration Examples

### Cron Job (Linux/macOS)

Trigger manual refresh every 6 hours:
```bash
# crontab -e
0 */6 * * * curl -X POST http://localhost:8080/api/refresh
```

### Windows Task Scheduler

Create a PowerShell script `refresh.ps1`:
```powershell
Invoke-WebRequest -Method POST -Uri "http://localhost:8080/api/refresh"
```

Schedule in Task Scheduler to run every 6 hours.

### Monitoring Script

Check refresh status periodically:
```bash
#!/bin/bash
# check_refresh.sh

HEALTH=$(curl -s http://localhost:8080/health)
ARTIST_COUNT=$(echo $HEALTH | jq -r '.total_artists')

if [ "$ARTIST_COUNT" -eq 0 ]; then
  echo "WARNING: No artists in database!"
  # Send alert
fi

echo "Health check OK: $ARTIST_COUNT artists"
```

## Additional Resources

- **Implementation Details**: See `doc/HOURLY_REHYDRATION_PLAN.md`
- **Architecture**: See `doc/ARCHITECTURE.md`
- **API Client**: See `internal/api/client.go`
- **Store Implementation**: See `internal/data/store.go`

## Support

For issues or questions:
1. Check server logs for error messages
2. Review this guide's troubleshooting section
3. Open an issue with:
   - Server logs (relevant portion)
   - Configuration settings
   - Steps to reproduce

---

**Last Updated**: October 3, 2025  
**Version**: 1.0  
**Status**: Production Ready ✅
