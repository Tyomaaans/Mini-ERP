package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	AppPort         string
	DatabaseAppUser string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	JWTSecretKey    string
	JWTExpiry       time.Duration
	RefreshExpiry   time.Duration
	AllowedOrigins  []string
	AdminEmail      string
	AdminPassword   string
	DemoPassword    string
}

func NewConfig() (AppConfig, error) {
	_ = godotenv.Load()
	cfg := AppConfig{
		AppPort:         os.Getenv("APP_PORT"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DatabaseAppUser: os.Getenv("APP_DATABASE_USER"),
		RedisAddr:       os.Getenv("REDIS_ADDR"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		JWTSecretKey:    os.Getenv("JWT_SECRET_KEY"),
		DemoPassword:    os.Getenv("DEMO_PASSWORD"),
		AdminEmail:      os.Getenv("ADMIN_EMAIL"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		JWTExpiry:       15 * time.Minute,
		RefreshExpiry:   168 * time.Hour,
	}
	if cfg.AppPort == "" {
		cfg.AppPort = "8080"
	}
	port, err := strconv.Atoi(cfg.AppPort)
	if err != nil || port < 1 || port > 65535 {
		return cfg, fmt.Errorf("APP_PORT must be a valid port")
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	for name, target := range map[string]*time.Duration{"JWT_EXPIRY": &cfg.JWTExpiry, "REFRESH_EXPIRY": &cfg.RefreshExpiry} {
		if raw := os.Getenv(name); raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil || value <= 0 {
				return cfg, fmt.Errorf("%s must be a positive duration", name)
			}
			*target = value
		}
	}
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return cfg, fmt.Errorf("ALLOWED_ORIGINS must contain exact HTTP origins")
		}
		cfg.AllowedOrigins = append(cfg.AllowedOrigins, origin)
	}
	return cfg, nil
}

func (c AppConfig) ValidateServer() error {
	if len(c.JWTSecretKey) < 32 {
		return fmt.Errorf("JWT_SECRET_KEY must contain at least 32 characters")
	}
	if c.RefreshExpiry < c.JWTExpiry {
		return fmt.Errorf("REFRESH_EXPIRY must be at least JWT_EXPIRY")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	return nil
}
