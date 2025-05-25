package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DiscordToken     string
	GoogleSheetsID   string
	GoogleAPIKey     string
	CacheDuration    time.Duration
	CommandPrefix    string
	LogLevel         string
}

func Load() (*Config, error) {
	cacheDuration := 5 * time.Minute
	if d := os.Getenv("CACHE_DURATION_MINUTES"); d != "" {
		if minutes, err := strconv.Atoi(d); err == nil {
			cacheDuration = time.Duration(minutes) * time.Minute
		}
	}

	return &Config{
		DiscordToken:     os.Getenv("DISCORD_TOKEN"),
		GoogleSheetsID:   os.Getenv("GOOGLE_SHEETS_ID"),
		GoogleAPIKey:     os.Getenv("GOOGLE_API_KEY"),
		CacheDuration:    cacheDuration,
		CommandPrefix:    getEnvOrDefault("COMMAND_PREFIX", "!"),
		LogLevel:         getEnvOrDefault("LOG_LEVEL", "info"),
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}