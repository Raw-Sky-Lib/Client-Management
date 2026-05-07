package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Portal DB
	SupabaseDBURL string
	DBSSLMode     string

	// Agency-hub
	AgencyAPIURL          string
	AgencyClientID        string
	AgencyManagementToken string

	// Auth
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	// Redis
	UpstashRedisURL string

	// Claude
	AnthropicAPIKey                 string
	AnthropicDefaultModel           string
	ClaudeDefaultMonthlyTokenBudget int

	// Email
	ResendAPIKey string
	ResendFrom   string

	// App
	Environment string
	PublicURL   string // backend's own public-facing URL (used in emails)
	FrontendURL string
	Port        string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file, reading from environment")
	}

	cfg := &Config{
		SupabaseDBURL:         os.Getenv("SUPABASE_DB_URL"),
		DBSSLMode:             envOrDefault("DB_SSLMODE", "require"),
		AgencyAPIURL:          os.Getenv("AGENCY_API_URL"),
		AgencyClientID:        os.Getenv("AGENCY_CLIENT_ID"),
		AgencyManagementToken: os.Getenv("AGENCY_MANAGEMENT_TOKEN"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		UpstashRedisURL:       os.Getenv("UPSTASH_REDIS_URL"),
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicDefaultModel: envOrDefault("ANTHROPIC_DEFAULT_MODEL", "claude-haiku-4-5-20251001"),
		ResendAPIKey:          os.Getenv("RESEND_API_KEY"),
		ResendFrom:            os.Getenv("RESEND_FROM"),
		Environment: envOrDefault("ENVIRONMENT", "development"),
		PublicURL:   envOrDefault("PUBLIC_URL", "http://localhost:8081"),
		FrontendURL: envOrDefault("FRONTEND_URL", "http://localhost:5174"),
		Port:        envOrDefault("PORT", "8081"),
	}

	if v := os.Getenv("CLAUDE_DEFAULT_MONTHLY_TOKEN_BUDGET"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("CLAUDE_DEFAULT_MONTHLY_TOKEN_BUDGET must be an integer: %w", err)
		}
		cfg.ClaudeDefaultMonthlyTokenBudget = n
	} else {
		cfg.ClaudeDefaultMonthlyTokenBudget = 150000
	}

	var err error
	if cfg.JWTAccessExpiry, err = parseDuration("JWT_ACCESS_EXPIRY", "15m"); err != nil {
		return nil, err
	}
	if cfg.JWTRefreshExpiry, err = parseDuration("JWT_REFRESH_EXPIRY", "168h"); err != nil {
		return nil, err
	}

	required := map[string]string{
		"SUPABASE_DB_URL":         cfg.SupabaseDBURL,
		"AGENCY_API_URL":          cfg.AgencyAPIURL,
		"AGENCY_CLIENT_ID":        cfg.AgencyClientID,
		"AGENCY_MANAGEMENT_TOKEN": cfg.AgencyManagementToken,
		"JWT_SECRET":              cfg.JWTSecret,
		"UPSTASH_REDIS_URL":       cfg.UpstashRedisURL,
		"ANTHROPIC_API_KEY":       cfg.AnthropicAPIKey,
		"RESEND_API_KEY":          cfg.ResendAPIKey,
		"RESEND_FROM":             cfg.ResendFrom,
	}
	for name, val := range required {
		if val == "" {
			return nil, fmt.Errorf("required env var %s is not set", name)
		}
	}

	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseDuration(key, def string) (time.Duration, error) {
	v := envOrDefault(key, def)
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration (e.g. 15m): %w", key, err)
	}
	return d, nil
}
