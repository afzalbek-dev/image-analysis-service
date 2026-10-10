package config

import (
	"os"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.MaxUploadSizeMB != 20 {
		t.Errorf("expected default max upload size 20, got %d", cfg.MaxUploadSizeMB)
	}
	if cfg.SimilarityThreshold != 0.85 {
		t.Errorf("expected default similarity threshold 0.85, got %f", cfg.SimilarityThreshold)
	}
	if cfg.HammingDistanceThreshold != 10 {
		t.Errorf("expected default hamming distance 10, got %d", cfg.HammingDistanceThreshold)
	}
}

func TestConfigEnvironmentOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("MAX_UPLOAD_SIZE_MB", "50")
	os.Setenv("SIMILARITY_THRESHOLD", "0.92")
	os.Setenv("HAMMING_DISTANCE_THRESHOLD", "5")
	os.Setenv("APP_ENV", "production")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.MaxUploadSizeMB != 50 {
		t.Errorf("expected max upload size 50, got %d", cfg.MaxUploadSizeMB)
	}
	if cfg.SimilarityThreshold != 0.92 {
		t.Errorf("expected similarity threshold 0.92, got %f", cfg.SimilarityThreshold)
	}
	if cfg.HammingDistanceThreshold != 5 {
		t.Errorf("expected hamming distance 5, got %d", cfg.HammingDistanceThreshold)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected env production, got %s", cfg.Environment)
	}
}
