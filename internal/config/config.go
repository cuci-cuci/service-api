package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port               int    `env:"PORT" envDefault:"8080"`
	DatabaseURL        string `env:"DATABASE_URL,required"`
	JWTSecret          string `env:"JWT_SECRET,required"`
	JWTExpiryMinutes       int    `env:"JWT_EXPIRY_MINUTES" envDefault:"15"`
	JWTRefreshExpiryDays   int    `env:"JWT_REFRESH_EXPIRY_DAYS" envDefault:"7"`
	CORSAllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	LogLevel           string `env:"LOG_LEVEL" envDefault:"info"`
	MaxBodySize        int64  `env:"MAX_BODY_SIZE" envDefault:"1048576"`
	ReadTimeoutSecs    int    `env:"READ_TIMEOUT_SECONDS" envDefault:"30"`
	WriteTimeoutSecs   int    `env:"WRITE_TIMEOUT_SECONDS" envDefault:"60"`
	DefaultPerPage     int    `env:"DEFAULT_PER_PAGE" envDefault:"20"`
	MaxPerPage         int    `env:"MAX_PER_PAGE" envDefault:"100"`
	DefaultMemberTier  string `env:"DEFAULT_MEMBER_TIER" envDefault:"bronze"`
	FonnteAPIURL       string `env:"FONNTE_API_URL" envDefault:"https://api.fonnte.com/send"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}
