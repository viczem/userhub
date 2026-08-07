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
	defaultHTTPAddr   = ":8080"
	AppEnvDevelopment = "development"
	AppEnvProduction  = "production"
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
	})
)

// Config contains Identity runtime settings.
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
}

// NewConfig parses and validates Identity configuration from the environment.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	if errs := schema.Parse(zenv.NewDataProvider(), cfg); errs != nil {
		return nil, newError(z.Issues.Prettify(errs))
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
