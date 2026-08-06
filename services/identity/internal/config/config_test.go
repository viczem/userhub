package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		wantAddr string
	}{
		{name: "default", wantAddr: defaultHTTPAddr},
		{name: "IPv4 override", value: "127.0.0.1:9000", wantAddr: "127.0.0.1:9000"},
		{name: "hostname override", value: "localhost:9000", wantAddr: "localhost:9000"},
		{name: "IPv6 override", value: "[::1]:9000", wantAddr: "[::1]:9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.HTTPAddr != tt.wantAddr {
				t.Errorf("Load().HTTPAddr = %q, want %q", cfg.HTTPAddr, tt.wantAddr)
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
