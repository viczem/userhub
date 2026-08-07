package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		appEnv   string
		value    string
		wantEnv  string
		wantAddr string
	}{
		{name: "default", appEnv: defaultAppEnv, wantEnv: defaultAppEnv, wantAddr: defaultHTTPAddr},
		{name: "production environment", appEnv: "production", wantEnv: "production", wantAddr: defaultHTTPAddr},
		{name: "development environment", appEnv: "development", wantEnv: "development", wantAddr: defaultHTTPAddr},
		{name: "IPv4 override", value: "127.0.0.1:9000", wantEnv: defaultAppEnv, wantAddr: "127.0.0.1:9000"},
		{name: "hostname override", value: "localhost:9000", wantEnv: defaultAppEnv, wantAddr: "localhost:9000"},
		{name: "IPv6 override", value: "[::1]:9000", wantEnv: defaultAppEnv, wantAddr: "[::1]:9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("HTTP_ADDR", tt.value)

			cfg, err := Load()
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

func TestLoadRejectsInvalidAppEnv(t *testing.T) {
	for _, value := range []string{"staging", "Production", "secret-environment-value"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("APP_ENV", value)

			_, err := Load()
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

			_, err := Load()
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
