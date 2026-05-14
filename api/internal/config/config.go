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
	AgencyAPIURL     string
	PortalAdminSecret string

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
	MailerProvider string // "resend" or "brevo"
	EmailFrom      string // from address, used by all providers
	ResendAPIKey   string // required when MailerProvider=resend
	BrevoSMTPUser  string // required when MailerProvider=brevo
	BrevoSMTPKey   string // required when MailerProvider=brevo

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
		AgencyAPIURL:      os.Getenv("AGENCY_API_URL"),
		PortalAdminSecret: os.Getenv("PORTAL_ADMIN_SECRET"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		UpstashRedisURL:       os.Getenv("UPSTASH_REDIS_URL"),
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicDefaultModel: envOrDefault("ANTHROPIC_DEFAULT_MODEL", "claude-haiku-4-5-20251001"),
		MailerProvider: envOrDefault("MAILER_PROVIDER", "resend"),
		EmailFrom:      envOrDefault("EMAIL_FROM", os.Getenv("RESEND_FROM")),
		ResendAPIKey:   os.Getenv("RESEND_API_KEY"),
		BrevoSMTPUser:  os.Getenv("BREVO_SMTP_USER"),
		BrevoSMTPKey:   os.Getenv("BREVO_SMTP_KEY"),
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
		"SUPABASE_DB_URL":    cfg.SupabaseDBURL,
		"AGENCY_API_URL":     cfg.AgencyAPIURL,
		"PORTAL_ADMIN_SECRET": cfg.PortalAdminSecret,
		"JWT_SECRET":         cfg.JWTSecret,
		"UPSTASH_REDIS_URL":  cfg.UpstashRedisURL,
		"ANTHROPIC_API_KEY":  cfg.AnthropicAPIKey,
	}
	for name, val := range required {
		if val == "" {
			return nil, fmt.Errorf("required env var %s is not set", name)
		}
	}

	if cfg.EmailFrom == "" {
		return nil, fmt.Errorf("required env var EMAIL_FROM (or RESEND_FROM) is not set")
	}
	switch cfg.MailerProvider {
	case "resend":
		if cfg.ResendAPIKey == "" {
			return nil, fmt.Errorf("RESEND_API_KEY is required when MAILER_PROVIDER=resend")
		}
	case "brevo":
		if cfg.BrevoSMTPUser == "" {
			return nil, fmt.Errorf("BREVO_SMTP_USER is required when MAILER_PROVIDER=brevo")
		}
		if cfg.BrevoSMTPKey == "" {
			return nil, fmt.Errorf("BREVO_SMTP_KEY is required when MAILER_PROVIDER=brevo")
		}
	default:
		return nil, fmt.Errorf("MAILER_PROVIDER must be \"resend\" or \"brevo\", got %q", cfg.MailerProvider)
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
