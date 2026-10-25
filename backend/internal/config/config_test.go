package config

import (
	"testing"
	"time"
)

func TestConfigurationRejectsInvalidRuntimeValues(t *testing.T) {
	for _, test := range []struct{ name, value string }{{"APP_PORT", "70000"}, {"APP_PORT", "invalid"}, {"JWT_EXPIRY", "0s"}, {"REFRESH_EXPIRY", "invalid"}, {"ALLOWED_ORIGINS", "*"}, {"ALLOWED_ORIGINS", "https://example.test/path"}} {
		t.Run(test.name+test.value, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://localhost/example")
			t.Setenv("APP_PORT", "8080")
			t.Setenv("JWT_EXPIRY", "15m")
			t.Setenv("REFRESH_EXPIRY", "168h")
			t.Setenv("ALLOWED_ORIGINS", "")
			t.Setenv(test.name, test.value)
			if _, err := NewConfig(); err == nil {
				t.Fatal("invalid configuration was accepted")
			}
		})
	}
	config := AppConfig{
		JWTSecretKey:  "short",
		RedisAddr:     "localhost:6379",
		JWTExpiry:     time.Minute,
		RefreshExpiry: time.Hour,
	}
	if config.ValidateServer() == nil {
		t.Fatal("short signing key was accepted")
	}
	config.JWTSecretKey = "a-private-test-key-at-least-32-characters"
	config.RefreshExpiry = time.Second
	if config.ValidateServer() == nil {
		t.Fatal("refresh lifetime shorter than access lifetime was accepted")
	}
}
