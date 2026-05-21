package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DSN            string        `env:"DSN" env-required:"true"`
	ScyllaHosts    []string      `env:"SCYLLA_HOSTS" env-default:"localhost" env-separator:","`
	ScyllaKeyspace string        `env:"SCYLLA_KEYSPACE" env-default:"ws"`
	RedisAddr      string        `env:"REDIS_ADDR" env-default:"localhost:6379"`
	RedisPassword  string        `env:"REDIS_PASSWORD"`
	JWTSecret      string        `env:"JWT_SECRET" env-required:"true"`
	JWTTTL         time.Duration `env:"JWT_TTL" env-default:"24h"`
	Addr           string        `env:"ADDR" env-default:":8080"`
	AllowedOrigins []string      `env:"ALLOWED_ORIGINS" env-separator:","`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		if envErr := cleanenv.ReadEnv(cfg); envErr != nil {
			return nil, fmt.Errorf("config: %w", envErr)
		}
	}
	return cfg, nil
}
