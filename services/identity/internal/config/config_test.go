package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		appEnv   string
		value    string
		wantEnv  string
		wantAddr string
	}{
		{name: "default", appEnv: AppEnvProduction, wantEnv: AppEnvProduction, wantAddr: defaultHTTPAddr},
		{name: "production environment", appEnv: "production", wantEnv: "production", wantAddr: defaultHTTPAddr},
		{name: "development environment", appEnv: "development", wantEnv: "development", wantAddr: defaultHTTPAddr},
		{name: "IPv4 override", value: "127.0.0.1:9000", wantEnv: AppEnvProduction, wantAddr: "127.0.0.1:9000"},
		{name: "hostname override", value: "localhost:9000", wantEnv: AppEnvProduction, wantAddr: "localhost:9000"},
		{name: "IPv6 override", value: "[::1]:9000", wantEnv: AppEnvProduction, wantAddr: "[::1]:9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("HTTP_ADDR", tt.value)

			cfg, err := NewConfig()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.AppEnv != tt.wantEnv {
				t.Errorf("Load().AppEnv = %q, want %q", cfg.AppEnv, tt.wantEnv)
			}
			if cfg.HTTPAddr != tt.wantAddr {
				t.Errorf("Load().HTTPAddr = %q, want %q", cfg.HTTPAddr, tt.wantAddr)
			}
		})
	}
}

func TestNewConfigHTTPRuntimeSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		clearRuntimeEnv(t)

		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("NewConfig() error = %v", err)
		}

		if cfg.HTTPMaxHeaderBytes != 16<<10 || cfg.HTTPMaxBodyBytes != 64<<10 ||
			cfg.HTTPWriteTimeout != 10*time.Second || cfg.HTTPReadTimeout != 5*time.Second ||
			cfg.HTTPReadHeaderTimeout != 2*time.Second || cfg.HTTPIdleTimeout != 30*time.Second ||
			cfg.HTTPGracefulShutdownTimeout != 30*time.Second {
			t.Errorf("NewConfig() HTTP defaults = %+v, want documented defaults", cfg)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		clearRuntimeEnv(t)
		t.Setenv("HTTP_MAX_HEADER_BYTES", "8192")
		t.Setenv("HTTP_MAX_BODY_BYTES", "16384")
		t.Setenv("HTTP_WRITE_TIMEOUT", "11s")
		t.Setenv("HTTP_READ_TIMEOUT", "6s")
		t.Setenv("HTTP_READ_HEADER_TIMEOUT", "3s")
		t.Setenv("HTTP_IDLE_TIMEOUT", "31s")
		t.Setenv("HTTP_GRACEFUL_SHUTDOWN_TIMEOUT", "32s")

		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("NewConfig() error = %v", err)
		}

		if cfg.HTTPMaxHeaderBytes != 8192 || cfg.HTTPMaxBodyBytes != 16384 ||
			cfg.HTTPWriteTimeout != 11*time.Second || cfg.HTTPReadTimeout != 6*time.Second ||
			cfg.HTTPReadHeaderTimeout != 3*time.Second || cfg.HTTPIdleTimeout != 31*time.Second ||
			cfg.HTTPGracefulShutdownTimeout != 32*time.Second {
			t.Errorf("NewConfig() HTTP overrides = %+v, want configured values", cfg)
		}
	})
}

func TestNewConfigRejectsInvalidHTTPRuntimeSettings(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		value    string
	}{
		{name: "small header limit", variable: "HTTP_MAX_HEADER_BYTES", value: "4095"},
		{name: "small body limit", variable: "HTTP_MAX_BODY_BYTES", value: "8191"},
		{name: "invalid write timeout", variable: "HTTP_WRITE_TIMEOUT", value: "invalid-duration"},
		{name: "small read timeout", variable: "HTTP_READ_TIMEOUT", value: "500ms"},
		{name: "small header read timeout", variable: "HTTP_READ_HEADER_TIMEOUT", value: "500ms"},
		{name: "small idle timeout", variable: "HTTP_IDLE_TIMEOUT", value: "500ms"},
		{name: "small shutdown timeout", variable: "HTTP_GRACEFUL_SHUTDOWN_TIMEOUT", value: "500ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRuntimeEnv(t)
			t.Setenv(tt.variable, tt.value)

			_, err := NewConfig()
			if err == nil {
				t.Fatal("NewConfig() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.variable) {
				t.Errorf("NewConfig() error = %q, want variable name", err)
			}
			if strings.Contains(err.Error(), tt.value) {
				t.Errorf("NewConfig() error exposes supplied value %q", tt.value)
			}
		})
	}
}

func TestLoadRejectsInvalidAppEnv(t *testing.T) {
	for _, value := range []string{"staging", "Production", "secret-environment-value"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("APP_ENV", value)

			_, err := NewConfig()
			if err == nil {
				t.Fatal("Load() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), "APP_ENV") {
				t.Errorf("Load() error = %q, want variable name", err)
			}
			if strings.Contains(err.Error(), value) {
				t.Errorf("Load() error exposes supplied value %q", value)
			}
		})
	}
}

func clearRuntimeEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV", "HTTP_ADDR", "HTTP_MAX_HEADER_BYTES", "HTTP_MAX_BODY_BYTES",
		"HTTP_WRITE_TIMEOUT", "HTTP_READ_TIMEOUT", "HTTP_READ_HEADER_TIMEOUT",
		"HTTP_IDLE_TIMEOUT", "HTTP_GRACEFUL_SHUTDOWN_TIMEOUT",
	} {
		unsetEnv(t, key)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, set := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q) error = %v", key, err)
	}
	t.Cleanup(func() {
		if set {
			_ = os.Setenv(key, value)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestLoadRejectsInvalidHTTPAddr(t *testing.T) {
	for _, value := range []string{
		"localhost",
		":0",
		":65536",
		"localhost:http",
		"bad host:8080",
		"-invalid.example:8080",
	} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", value)

			_, err := NewConfig()
			if err == nil {
				t.Fatal("Load() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), "HTTP_ADDR") {
				t.Errorf("Load() error = %q, want variable name", err)
			}
			if strings.Contains(err.Error(), value) {
				t.Errorf("Load() error exposes supplied value %q", value)
			}
		})
	}
}
