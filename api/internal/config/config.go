// Package config loads runtime settings from environment variables.
package config

import (
	"os"
	"time"
)

type Config struct {
	Port          string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	Timezone      string

	// QuoteRefreshInterval controls how often the quote cache is refreshed.
	QuoteRefreshInterval time.Duration
	// CloseIngestAt is the local time (HH:MM) to ingest daily prices after market close.
	CloseIngestAt string
	// EveningIngestAt is the local time (HH:MM) to ingest institutional/valuation/revenue
	// data and run the daily screens.
	EveningIngestAt string
	// WebRequestInterval is the minimum gap between requests to the TWSE/TPEx web
	// endpoints. Going faster gets the IP banned.
	WebRequestInterval time.Duration

	// FinMindToken is optional; reserved for filling data gaps via FinMind.
	FinMindToken string

	// GeminiAPIKey enables the AI chat; empty disables it.
	GeminiAPIKey string
	GeminiModel  string
}

func Load() Config {
	return Config{
		Port:                 getenv("PORT", "8080"),
		DatabaseURL:          getenv("DATABASE_URL", "postgres://copium:copium@localhost:5432/copium?sslmode=disable"),
		RedisAddr:            getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        getenv("REDIS_PASSWORD", ""),
		Timezone:             getenv("TZ", "Asia/Taipei"),
		QuoteRefreshInterval: duration("QUOTE_REFRESH_INTERVAL", 10*time.Minute),
		CloseIngestAt:        getenv("CLOSE_INGEST_AT", "15:00"),
		EveningIngestAt:      getenv("EVENING_INGEST_AT", "20:00"),
		WebRequestInterval:   duration("WEB_REQUEST_INTERVAL", 3*time.Second),
		FinMindToken:         getenv("FINMIND_TOKEN", ""),
		GeminiAPIKey:         getenv("GEMINI_API_KEY", ""),
		GeminiModel:          getenv("GEMINI_MODEL", "gemini-3.8-flash,gemini-flash-latest"),
	}
}

// Location returns the configured time zone, falling back to UTC+8.
func (c Config) Location() *time.Location {
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return loc
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil || d <= 0 {
		return def
	}
	return d
}
