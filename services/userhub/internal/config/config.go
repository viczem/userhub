// Package config loads and validates application configuration provided by environment variables.
package config

import (
	"reflect"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

const (
	// AppEnvDevelopment is the development application environment.
	AppEnvDevelopment = "development"
	// AppEnvProduction is the production application environment.
	AppEnvProduction = "production"
)

// HTTPConfig contains HTTP server configuration.
type HTTPConfig struct {
	Addr                    string        `env:"ADDR" envDefault:":8080"`
	MaxHeaderBytes          int           `env:"MAX_HEADER_BYTES" envDefault:"16384" validate:"gte=4096"`
	MaxBodyBytes            int           `env:"MAX_BODY_BYTES" envDefault:"65536" validate:"gte=8192"`
	WriteTimeout            time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	ReadTimeout             time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	ReadHeaderTimeout       time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"2s"`
	IdleTimeout             time.Duration `env:"IDLE_TIMEOUT" envDefault:"30s"`
	GracefulShutdownTimeout time.Duration `env:"GRACEFUL_SHUTDOWN_TIMEOUT" envDefault:"30s"`
}

// DBConfig contains database configuration.
type DBConfig struct {
	DirectURL       string        `env:"URL,required" validate:"required"`
	PoolURL         string        `env:"URL_POOL"`
	ConnMaxLifetime time.Duration `env:"CONN_MAX_LIFETIME" validate:"gte=0"`
	ConnMaxIdleTime time.Duration `env:"CONN_MAX_IDLE_TIME" validate:"gte=0"`
	MaxOpenConns    int           `env:"MAX_OPEN_CONNS" envDefault:"20" validate:"gte=1"`
	MinConns        int           `env:"MIN_CONNS" envDefault:"2" validate:"gte=0,ltefield=MaxOpenConns"`
}

// Config contains UserHub Service runtime settings.
type Config struct {
	AppEnv            string     `env:"APP_ENV" envDefault:"production" validate:"oneof=development production"`
	DB                DBConfig   `envPrefix:"DB_"`
	HTTP              HTTPConfig `envPrefix:"HTTP_"`
	KeyringHMAC       Keyring    `env:"KEYRING_HMAC,required"`
	KeyringEncryption Keyring    `env:"KEYRING_ENCRYPTION,required"`
}

// NewConfig parses and validates UserHub Service configuration from the environment.
func NewConfig() (*Config, error) {
	var (
		cfg      Config
		validate = validator.New()
		options  = env.Options{
			FuncMap: map[reflect.Type]env.ParserFunc{
				reflect.TypeFor[Keyring](): func(v string) (any, error) {
					return parseKeyring(v)
				},
			},
		}
	)

	if err := env.ParseWithOptions(&cfg, options); err != nil {
		return nil, err
	}

	if err := validate.Struct(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
