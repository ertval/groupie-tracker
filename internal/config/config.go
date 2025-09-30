package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config captures the runtime configuration knobs for the HTTP server.
type Config struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	APIBaseURL   string
	HTTPTimeout  time.Duration
	MaxBodyBytes int64
}

const (
	defaultPort         = "8082"
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 60 * time.Second
	defaultHTTPTimeout  = 10 * time.Second
	defaultMaxBodyMB    = 32 // MB
)

// FromEnv builds a Config instance by merging environment variables with defaults.
func FromEnv() Config {
	cfg := Config{
		Port:         ":" + defaultPort,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
		APIBaseURL:   "",
		HTTPTimeout:  defaultHTTPTimeout,
		MaxBodyBytes: int64(defaultMaxBodyMB) << 20,
	}

	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			cfg.Port = port
		} else {
			cfg.Port = ":" + port
		}
	}

	if baseURL := strings.TrimSpace(os.Getenv("GROUPIE_API_BASE_URL")); baseURL != "" {
		cfg.APIBaseURL = baseURL
	}

	if timeoutStr := strings.TrimSpace(os.Getenv("GROUPIE_HTTP_TIMEOUT")); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil && duration > 0 {
			cfg.HTTPTimeout = duration
		}
	}

	if readTimeout := strings.TrimSpace(os.Getenv("GROUPIE_READ_TIMEOUT")); readTimeout != "" {
		if duration, err := time.ParseDuration(readTimeout); err == nil && duration > 0 {
			cfg.ReadTimeout = duration
		}
	}

	if writeTimeout := strings.TrimSpace(os.Getenv("GROUPIE_WRITE_TIMEOUT")); writeTimeout != "" {
		if duration, err := time.ParseDuration(writeTimeout); err == nil && duration > 0 {
			cfg.WriteTimeout = duration
		}
	}

	if idleTimeout := strings.TrimSpace(os.Getenv("GROUPIE_IDLE_TIMEOUT")); idleTimeout != "" {
		if duration, err := time.ParseDuration(idleTimeout); err == nil && duration > 0 {
			cfg.IdleTimeout = duration
		}
	}

	if maxBody := strings.TrimSpace(os.Getenv("GROUPIE_MAX_BODY_MB")); maxBody != "" {
		if value, err := strconv.Atoi(maxBody); err == nil && value > 0 {
			cfg.MaxBodyBytes = int64(value) << 20
		}
	}

	return cfg
}
