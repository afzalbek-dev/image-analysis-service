package config

import (
	"os"
	"strconv"
)

// Config represents runtime application configuration.
type Config struct {
	Port                     string
	MaxUploadSizeMB          int64
	SimilarityThreshold      float64
	HammingDistanceThreshold int
	ReadTimeoutSeconds       int
	WriteTimeoutSeconds      int
	Environment              string
}

// Load loads configuration from environment variables with fallback defaults.
func Load() *Config {
	cfg := &Config{
		Port:                     getEnv("PORT", "8080"),
		MaxUploadSizeMB:          getEnvInt64("MAX_UPLOAD_SIZE_MB", 20),
		SimilarityThreshold:      getEnvFloat64("SIMILARITY_THRESHOLD", 0.85),
		HammingDistanceThreshold: getEnvInt("HAMMING_DISTANCE_THRESHOLD", 10),
		ReadTimeoutSeconds:       getEnvInt("READ_TIMEOUT_SECONDS", 30),
		WriteTimeoutSeconds:      getEnvInt("WRITE_TIMEOUT_SECONDS", 30),
		Environment:              getEnv("APP_ENV", "development"),
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvFloat64(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}
