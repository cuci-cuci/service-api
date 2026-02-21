package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port               int    `env:"PORT" envDefault:"8080"`
	DatabaseURL        string `env:"DATABASE_URL,required"`
	JWTSecret          string `env:"JWT_SECRET,required"`
	JWTExpiryHours     int    `env:"JWT_EXPIRY_HOURS" envDefault:"24"`
	CORSAllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}
