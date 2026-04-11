package config

import (
	"fmt"
	"os"
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
	ORCID    ORCIDConfig
}

type ServerConfig struct {
	Port           string
	FrontendURL    string
	AllowedOrigins string
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
			Port:           getEnv("SERVER_PORT", "8080"),
			FrontendURL:    getEnv("FRONTEND_URL", "http://localhost:3000"),
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
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
		ORCID: ORCIDConfig{
			ClientID:     getEnv("ORCID_CLIENT_ID", ""),
			ClientSecret: getEnv("ORCID_CLIENT_SECRET", ""),
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
