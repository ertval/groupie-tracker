# Deployment Guide

This guide covers deployment options and best practices for the Groupie Tracker application.

## Quick Deploy

### Local Development
```bash
# Clone and run locally
git clone <repository>
cd groupie-tracker
go run ./cmd/cli
```

### Production Build
```bash
# Build optimized binary
go build -ldflags="-s -w" -o groupie-tracker ./cmd/cli

# Run production server
./groupie-tracker
```

## Environment Configuration

### Environment Variables
```bash
PORT=8082              # Server port (default: 8082)
API_BASE_URL=https://groupietrackers.herokuapp.com/api  # External API
API_TIMEOUT=10s        # API request timeout
READ_TIMEOUT=10s       # HTTP read timeout
WRITE_TIMEOUT=10s      # HTTP write timeout
IDLE_TIMEOUT=60s       # HTTP idle timeout
```

### Configuration Files
The application uses compile-time configuration. No external config files required.

## Docker Deployment

### Dockerfile
```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o groupie-tracker ./cmd/cli

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/groupie-tracker .
COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates

EXPOSE 8082
CMD ["./groupie-tracker"]
```

### Build and Run
```bash
# Build Docker image
docker build -t groupie-tracker .

# Run container
docker run -p 8082:8082 groupie-tracker

# Run with environment variables
docker run -p 8080:8080 -e PORT=8080 groupie-tracker
```

### Docker Compose
```yaml
version: '3.8'
services:
  groupie-tracker:
    build: .
    ports:
      - "8082:8082"
    environment:
      - PORT=8082
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8082/"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Cloud Deployment

### Heroku
```bash
# Create Heroku app
heroku create your-app-name

# Set buildpack
heroku buildpacks:set heroku/go

# Deploy
git push heroku main

# Set environment variables
heroku config:set PORT=8080
```

### Railway
```bash
# Install Railway CLI
npm install -g @railway/cli

# Login and deploy
railway login
railway init
railway up
```

### Render
1. Connect your GitHub repository
2. Set build command: `go build -o app ./cmd/cli`
3. Set start command: `./app`
4. Deploy automatically

### Google Cloud Run
```bash
# Build and push container
gcloud builds submit --tag gcr.io/PROJECT_ID/groupie-tracker

# Deploy to Cloud Run
gcloud run deploy --image gcr.io/PROJECT_ID/groupie-tracker --platform managed
```

## Production Considerations

### Performance Tuning

#### Memory Optimization
```go
// Already implemented optimizations:
// - Single-pass data processing
// - Efficient string operations
// - Pre-allocated slices
// - Copy-on-read for thread safety
```

#### CPU Optimization
```go
// Current optimizations:
// - Concurrent API fetching
// - Template pre-compilation
// - Efficient search algorithms
// - Minimal allocations in hot paths
```

### Security Hardening

#### HTTP Security Headers
```go
// Already implemented:
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
```

#### Additional Security (Production Recommended)
```bash
# Use HTTPS in production
# Implement rate limiting
# Add CSRF protection for forms
# Use security-focused reverse proxy (nginx, Cloudflare)
```

### Monitoring Setup

#### Health Checks
```go
// Add health check endpoint
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) error {
    stats := h.store.GetStats()
    health := map[string]interface{}{
        "status": "healthy",
        "artists": stats.TotalArtists,
        "locations": stats.TotalLocations,
        "uptime": time.Since(startTime).String(),
    }
    return json.NewEncoder(w).Encode(health)
}
```

#### Logging Configuration
```go
// Production logging setup
log.SetOutput(os.Stdout)
log.SetFlags(log.LstdFlags | log.Lmicroseconds)

// For structured logging, consider:
// - logrus
// - zap
// - slog (Go 1.21+)
```

### Database Migration (Future)

If you need persistent storage:

```go
// Example database interface
type ArtistRepository interface {
    GetAll(ctx context.Context) ([]models.Artist, error)
    GetByID(ctx context.Context, id int) (models.Artist, error)
    GetBySlug(ctx context.Context, slug string) (models.Artist, error)
}

// Current implementation is memory-based for simplicity
// For production with large datasets, consider:
// - PostgreSQL for relational data
// - Redis for caching
// - Elasticsearch for search
```

## Scaling Strategies

### Horizontal Scaling
```yaml
# Kubernetes deployment example
apiVersion: apps/v1
kind: Deployment
metadata:
  name: groupie-tracker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: groupie-tracker
  template:
    metadata:
      labels:
        app: groupie-tracker
    spec:
      containers:
      - name: groupie-tracker
        image: groupie-tracker:latest
        ports:
        - containerPort: 8082
        env:
        - name: PORT
          value: "8082"
        resources:
          limits:
            memory: "128Mi"
            cpu: "500m"
          requests:
            memory: "64Mi"
            cpu: "250m"
```

### Load Balancing
```nginx
# Nginx configuration
upstream groupie_tracker {
    server app1:8082;
    server app2:8082;
    server app3:8082;
}

server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://groupie_tracker;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location /static/ {
        alias /path/to/static/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

### Caching Strategy
```go
// Current in-memory caching is sufficient for single instances
// For distributed caching, consider:

// Redis caching layer
type CacheService struct {
    redis *redis.Client
}

// CDN for static assets
// - Cloudflare
// - AWS CloudFront
// - Google Cloud CDN
```

## Backup and Recovery

### Data Backup
```bash
# Current implementation doesn't require backup (data from API)
# For future database integration:

# PostgreSQL backup
pg_dump groupie_tracker > backup.sql

# Redis backup
redis-cli SAVE
cp /var/lib/redis/dump.rdb backup/
```

### Disaster Recovery
```bash
# Application recovery
# 1. Restore code from version control
# 2. Rebuild application
# 3. Deploy to new infrastructure
# 4. Update DNS if necessary

# Data recovery (if using persistent storage)
# 1. Restore database from backup
# 2. Verify data integrity
# 3. Update application configuration
```

## Maintenance

### Updates
```bash
# Update dependencies
go get -u ./...
go mod tidy

# Security updates
go list -json -m all | nancy sleuth

# Performance profiling
go tool pprof http://localhost:8082/debug/pprof/profile
```

### Monitoring Metrics
```go
// Key metrics to monitor:
// - Response time percentiles
// - Error rate
// - Memory usage
// - CPU utilization
// - API response time
// - Active connections
```

### Log Rotation
```bash
# Logrotate configuration
/var/log/groupie-tracker/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 app app
    postrotate
        systemctl reload groupie-tracker
    endscript
}
```

## Troubleshooting

### Common Issues

#### High Memory Usage
```go
// Check for memory leaks
go tool pprof http://localhost:8082/debug/pprof/heap

// Solutions:
// - Verify copy-on-read is working
// - Check for goroutine leaks
// - Monitor garbage collection
```

#### Slow Response Times
```go
// Profile CPU usage
go tool pprof http://localhost:8082/debug/pprof/profile

// Common causes:
// - External API latency
// - Template rendering bottlenecks
// - Search algorithm inefficiency
```

#### External API Failures
```go
// Monitor API health
// Implement circuit breaker pattern
// Add fallback mechanisms
// Cache API responses when appropriate
```

### Performance Testing
```bash
# Load testing with hey
hey -n 1000 -c 10 http://localhost:8082/

# Apache Bench
ab -n 1000 -c 10 http://localhost:8082/

# Advanced testing with k6
k6 run performance-test.js
```

This deployment guide provides comprehensive instructions for deploying the Groupie Tracker application in various environments, from local development to production cloud deployments.