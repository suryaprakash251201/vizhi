package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host            string
	Port            string
	JWTSecret       string
	JWTIssuer       string
	TokenDuration   time.Duration
	UploadDir       string
	AllowedUID      int
	AllowedGID      int
	AllowedApps     []string
	RateLimitPerSec int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	TLSEnabled      bool
	TLSCertFile     string
	TLSKeyFile      string
	LogLevel        string
	WSEmitInterval  time.Duration
	MaxUploadSize   int64
	ChunkSize       int64
}

func Load() *Config {
	return &Config{
		Host:            getEnv("VIZHI_HOST", "0.0.0.0"),
		Port:            getEnv("VIZHI_PORT", "8443"),
		JWTSecret:       getEnv("VIZHI_JWT_SECRET", "change-me-in-production"),
		JWTIssuer:       getEnv("VIZHI_JWT_ISSUER", "vizhi"),
		TokenDuration:   getDurationEnv("VIZHI_TOKEN_DURATION", 12*time.Hour),
		UploadDir:       getEnv("VIZHI_UPLOAD_DIR", "/data/uploads"),
		AllowedUID:      getIntEnv("VIZHI_ALLOWED_UID", 1000),
		AllowedGID:      getIntEnv("VIZHI_ALLOWED_GID", 1000),
		AllowedApps:     getEnvSlice("VIZHI_ALLOWED_APPS", []string{"firefox", "chromium", "nautilus", "gnome-terminal", "code", "gedit", "vlc", "thunderbird"}),
		RateLimitPerSec: getIntEnv("VIZHI_RATE_LIMIT", 30),
		ReadTimeout:     getDurationEnv("VIZHI_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:    getDurationEnv("VIZHI_WRITE_TIMEOUT", 30*time.Second),
		TLSEnabled:      getBoolEnv("VIZHI_TLS_ENABLED", false),
		TLSCertFile:     getEnv("VIZHI_TLS_CERT", "/data/tls/server.crt"),
		TLSKeyFile:      getEnv("VIZHI_TLS_KEY", "/data/tls/server.key"),
		LogLevel:        getEnv("VIZHI_LOG_LEVEL", "info"),
		WSEmitInterval:  getDurationEnv("VIZHI_WS_EMIT_INTERVAL", 2*time.Second),
		MaxUploadSize:   getInt64Env("VIZHI_MAX_UPLOAD_SIZE", 500*1024*1024),
		ChunkSize:       getInt64Env("VIZHI_CHUNK_SIZE", 4*1024*1024),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getInt64Env(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return fallback
}
