// Package config loads and validates application configuration.
// Configuration is provided exclusively through environment variables, which
// are documented in the README.
package config

import (
	"net"
	"strconv"
	"strings"
	"time"

	z "github.com/Oudwins/zog"
	"github.com/Oudwins/zog/zenv"
)

const (
	defaultHTTPAddr       = ":8080"
	defaultDBMaxOpenConns = 20
	defaultDBMinConns     = 2
	// AppEnvDevelopment is the development application environment.
	AppEnvDevelopment = "development"
	// AppEnvProduction is the production application environment.
	AppEnvProduction = "production"
)

var (
	schema = z.Struct(z.Shape{
		"AppEnv":                      z.String().OneOf([]string{"production", "development"}).Default(AppEnvProduction),
		"HTTPAddr":                    z.String().Default(defaultHTTPAddr).TestFunc(validHTTPAddr),
		"HTTPMaxHeaderBytes":          z.Int().Default(16 << 10).GTE(4 << 10),
		"HTTPMaxBodyBytes":            z.Int().Default(64 << 10).GTE(8 << 10),
		"HTTPWriteTimeout":            durationSchema(10 * time.Second),
		"HTTPReadTimeout":             durationSchema(5 * time.Second),
		"HTTPReadHeaderTimeout":       durationSchema(2 * time.Second),
		"HTTPIdleTimeout":             durationSchema(30 * time.Second),
		"HTTPGracefulShutdownTimeout": durationSchema(30 * time.Second),
		"DBDirectURL":                 z.String().Required(),
		"DBPoolURL":                   z.String().Optional(),
		"DBMaxOpenConns":              z.Int().Default(defaultDBMaxOpenConns).GTE(1).LTE(1<<31 - 1),
		"DBMinConns":                  z.Int().Default(defaultDBMinConns).GTE(0).LTE(1<<31 - 1),
		"DBConnMaxLifetime":           nonNegativeDurationSchema(0),
		"DBConnMaxIdleTime":           nonNegativeDurationSchema(0),
	})
)

// Config contains IAM runtime settings.
type Config struct {
	AppEnv                      string        `env:"APP_ENV"`
	HTTPAddr                    string        `env:"HTTP_ADDR"`
	HTTPMaxHeaderBytes          int           `env:"HTTP_MAX_HEADER_BYTES"`
	HTTPMaxBodyBytes            int           `env:"HTTP_MAX_BODY_BYTES"`
	HTTPWriteTimeout            time.Duration `env:"HTTP_WRITE_TIMEOUT"`
	HTTPReadTimeout             time.Duration `env:"HTTP_READ_TIMEOUT"`
	HTTPReadHeaderTimeout       time.Duration `env:"HTTP_READ_HEADER_TIMEOUT"`
	HTTPIdleTimeout             time.Duration `env:"HTTP_IDLE_TIMEOUT"`
	HTTPGracefulShutdownTimeout time.Duration `env:"HTTP_GRACEFUL_SHUTDOWN_TIMEOUT"`
	DBDirectURL                 string        `env:"DB_URL"`
	DBPoolURL                   string        `env:"DB_URL_POOL"`
	DBConnMaxLifetime           time.Duration `env:"DB_CONN_MAX_LIFETIME"`
	DBConnMaxIdleTime           time.Duration `env:"DB_CONN_MAX_IDLE_TIME"`
	DBMaxOpenConns              int           `env:"DB_MAX_OPEN_CONNS"`
	DBMinConns                  int           `env:"DB_MIN_CONNS"`
}

// NewConfig parses and validates IAM configuration from the environment.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	if errs := schema.Parse(zenv.NewDataProvider(), cfg); errs != nil {
		return nil, newError(z.Issues.Prettify(errs))
	}

	if cfg.DBMinConns > cfg.DBMaxOpenConns {
		return nil, newError("DB_MIN_CONNS must be less than or equal to DB_MAX_OPEN_CONNS")
	}

	return cfg, nil
}

func validHTTPAddr(value *string, _ z.Ctx) bool {
	host, portText, err := net.SplitHostPort(*value)
	if err != nil || !validHost(host) {
		return false
	}

	port, err := strconv.Atoi(portText)

	return err == nil && port >= 1 && port <= 65535
}

func validHost(host string) bool {
	if host == "" || net.ParseIP(host) != nil {
		return true
	}

	host = strings.TrimSuffix(host, ".")
	if host == "" || len(host) > 253 {
		return false
	}

	for label := range strings.SplitSeq(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}

		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}

	return true
}
