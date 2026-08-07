package config

import (
	"errors"
	"net"
	"strconv"
	"strings"

	z "github.com/Oudwins/zog"
	"github.com/Oudwins/zog/zenv"
)

const (
	appEnvIssueCode = "invalid_app_env"
	defaultAppEnv   = "production"
	defaultHTTPAddr = ":8080"
)

var (
	errAppEnv   = errors.New("invalid environment variable APP_ENV: must be production or development")
	errHTTPAddr = errors.New("invalid environment variable HTTP_ADDR: must be a valid TCP listen address with a numeric port between 1 and 65535")
	schema      = z.Struct(z.Shape{
		"AppEnv":   z.String().Default(defaultAppEnv).TestFunc(validAppEnv, z.IssueCode(appEnvIssueCode)),
		"HTTPAddr": z.String().Default(defaultHTTPAddr).TestFunc(validHTTPAddr),
	})
)

// Config contains Identity runtime settings.
type Config struct {
	AppEnv   string `env:"APP_ENV"`
	HTTPAddr string `env:"HTTP_ADDR"`
}

// Load parses and validates Identity configuration from the environment.
func Load() (Config, error) {
	var cfg Config
	if issues := schema.Parse(zenv.NewDataProvider(), &cfg); issues != nil {
		for _, issue := range issues {
			if issue.Code == appEnvIssueCode {
				return Config{}, errAppEnv
			}
		}
		return Config{}, errHTTPAddr
	}

	return cfg, nil
}

func validAppEnv(value *string, _ z.Ctx) bool {
	return *value == "production" || *value == "development"
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
