package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	DeepSeek DeepSeekConfig
	Scopus   ScopusConfig
	CrossRef CrossRefConfig
	OpenAlex OpenAlexConfig
	BulkSync BulkSyncConfig
	ORCID    ORCIDConfig
	Google   GoogleConfig
	SMTP     SMTPConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Port            string
	FrontendURL     string
	AllowedOrigins  string
	UploadDir       string
	RateLimitPerMin int
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

type DeepSeekConfig struct {
	APIKey  string
	BaseURL string
}

type ScopusConfig struct {
	APIKey string
}

type CrossRefConfig struct {
	Mailto string
}

// BulkSyncConfig paces the admin's sync of every researcher with an ORCID. Each one
// spends up to six OpenAlex calls from the budget journal screening also uses.
type BulkSyncConfig struct {
	Delay      time.Duration
	DailyLimit int
}

// OpenAlexConfig holds the optional key; without it requests share a small
// anonymous daily budget. InstitutionROR identifies the university's own output
// (the public "published research" count).
type OpenAlexConfig struct {
	APIKey         string
	InstitutionROR string
}

// GoogleConfig enables "Sign in with Google" restricted to one institutional domain.
type GoogleConfig struct {
	ClientID      string
	AllowedDomain string
}

// SMTPConfig enables outgoing notification email; empty Host disables it.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

type AuthConfig struct {
	// AllowPasswordRegister enables open self-registration with email + password.
	// Keep it off in production: faculty sign in with their university Google account.
	AllowPasswordRegister bool
}

type ORCIDConfig struct {
	ClientID     string
	ClientSecret string
}

func Load() (*Config, error) {
	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:3000"),
			AllowedOrigins:  getEnv("ALLOWED_ORIGINS", "*"),
			UploadDir:       getEnv("UPLOAD_DIR", "/data/uploads"),
			RateLimitPerMin: getEnvInt("RATE_LIMIT_PER_MIN", 300),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
			Expiry: jwtExpiry,
		},
		DeepSeek: DeepSeekConfig{
			APIKey:  getEnv("DEEPSEEK_API_KEY", ""),
			BaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		},
		Scopus: ScopusConfig{
			APIKey: getEnv("SCOPUS_API_KEY", ""),
		},
		CrossRef: CrossRefConfig{
			Mailto: getEnv("CROSSREF_MAILTO", ""),
		},
		OpenAlex: OpenAlexConfig{
			APIKey:         getEnv("OPENALEX_API_KEY", ""),
			InstitutionROR: getEnv("UNIVERSITY_ROR", "019vd4365"),
		},
		BulkSync: BulkSyncConfig{
			Delay:      time.Duration(getEnvInt("BULK_SYNC_DELAY_SECONDS", 5)) * time.Second,
			DailyLimit: getEnvInt("BULK_SYNC_DAILY_LIMIT", 300),
		},
		ORCID: ORCIDConfig{
			ClientID:     getEnv("ORCID_CLIENT_ID", ""),
			ClientSecret: getEnv("ORCID_CLIENT_SECRET", ""),
		},
		Google: GoogleConfig{
			ClientID:      getEnv("GOOGLE_CLIENT_ID", ""),
			AllowedDomain: strings.ToLower(getEnv("GOOGLE_ALLOWED_DOMAIN", "uoturath.edu.iq")),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnv("SMTP_PORT", "587"),
			Username: getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASS", ""),
			From:     getEnv("SMTP_FROM", ""),
			FromName: getEnv("SMTP_FROM_NAME", "شعبة الشؤون العلمية — جامعة التراث"),
		},
		Auth: AuthConfig{
			AllowPasswordRegister: getEnv("ALLOW_PASSWORD_REGISTER", "false") == "true",
		},
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
