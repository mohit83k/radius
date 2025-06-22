package config

import (
	"os"
)

// Config holds all the environment-based configurations.
type Config struct {
	ServerPort   string
	RadiusSecret string
	RedisAddr    string
	RedisPass    string
	RedisDB      int
	LogFilePath  string
}

// Load reads the configuration from environment variables.
func Load() Config {
	return Config{
		ServerPort:   getEnv("RADIUS_PORT", "1813"),
		RadiusSecret: getEnv("RADIUS_SECRET", "testing123"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:      0, // optional: parseInt(getEnv("REDIS_DB", "0"))
		LogFilePath:  getEnv("LOG_FILE_PATH", "/var/log/radius_updates.log"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
