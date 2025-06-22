package config

import (
	"testing"
)

func TestLoad_FromEnv(t *testing.T) {

	t.Run("default", func(t *testing.T) {

		cfg := Load()

		if cfg.RadiusSecret != "testing123" {
			t.Errorf("expected default secret, got %s", cfg.RadiusSecret)
		}
		if cfg.RedisAddr != "localhost:6379" {
			t.Errorf("expected default redis addr, got %s", cfg.RedisAddr)
		}
		if cfg.RedisPass != "" {
			t.Errorf("expected empty redis password, got %s", cfg.RedisPass)
		}
		if cfg.LogFilePath != "/var/log/radius_updates.log" {
			t.Errorf("expected default log path, got %s", cfg.LogFilePath)
		}
	})
}
