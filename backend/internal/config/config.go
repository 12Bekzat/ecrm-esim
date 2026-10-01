package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                    string
	HTTPAddr               string
	DatabaseURL            string
	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	CORSOrigins            []string
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
	BootstrapAdminName     string
	AutoMigrate            bool
	ZesimoBaseURL          string
	ZesimoAPIKey           string
	ZesimoKeyName          string
	ZesimoTimeout          time.Duration
}

func Load() (Config, error) {
	accessTTL, err := time.ParseDuration(getenv("ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getenv("REFRESH_TOKEN_TTL", "720h"))
	if err != nil {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL: %w", err)
	}
	autoMigrate, err := strconv.ParseBool(getenv("AUTO_MIGRATE", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("AUTO_MIGRATE: %w", err)
	}
	zesimoTimeout, err := time.ParseDuration(getenv("ZESIMO_TIMEOUT", "30s"))
	if err != nil {
		return Config{}, fmt.Errorf("ZESIMO_TIMEOUT: %w", err)
	}

	cfg := Config{
		Env:                    getenv("APP_ENV", "development"),
		HTTPAddr:               getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:            getenv("DATABASE_URL", "postgres://ecrm:ecrm@localhost:5432/ecrm?sslmode=disable"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		AccessTokenTTL:         accessTTL,
		RefreshTokenTTL:        refreshTTL,
		CORSOrigins:            splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173")),
		BootstrapAdminEmail:    strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapAdminName:     getenv("BOOTSTRAP_ADMIN_NAME", "Administrator"),
		AutoMigrate:            autoMigrate,
		ZesimoBaseURL:          getenv("ZESIMO_BASE_URL", "https://zesimo.com"),
		ZesimoAPIKey:           strings.TrimSpace(os.Getenv("ZESIMO_API_KEY")),
		ZesimoKeyName:          getenv("ZESIMO_KEY_NAME", "my_key"),
		ZesimoTimeout:          zesimoTimeout,
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}
