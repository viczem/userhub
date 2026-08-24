package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/viczem/userhub/services/iam/internal/config"
)

func TestParsePoolConfigSelectsEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		poolURL       string
		wantPort      uint16
		wantQueryMode pgx.QueryExecMode
	}{
		{
			name:          "direct endpoint",
			wantPort:      5432,
			wantQueryMode: pgx.QueryExecModeCacheStatement,
		},
		{
			name:          "pooled endpoint",
			poolURL:       "postgres://pool-user:pool-password@pool.example:6432/iam",
			wantPort:      6432,
			wantQueryMode: pgx.QueryExecModeExec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parsePoolConfig(&config.Config{
				DBDirectURL:       "postgres://direct-user:direct-password@direct.example:5432/iam",
				DBPoolURL:         tt.poolURL,
				DBMaxOpenConns:    20,
				DBMinConns:        2,
				DBConnMaxLifetime: time.Hour,
				DBConnMaxIdleTime: 5 * time.Minute,
			})
			if err != nil {
				t.Fatalf("parsePoolConfig() error = %v", err)
			}

			if cfg.ConnConfig.Port != tt.wantPort {
				t.Errorf("port = %d, want %d", cfg.ConnConfig.Port, tt.wantPort)
			}

			if cfg.ConnConfig.DefaultQueryExecMode != tt.wantQueryMode {
				t.Errorf("query mode = %s, want %s", cfg.ConnConfig.DefaultQueryExecMode, tt.wantQueryMode)
			}

			if cfg.MaxConns != 20 || cfg.MinConns != 2 || cfg.MaxConnLifetime != time.Hour ||
				cfg.MaxConnIdleTime != 5*time.Minute {
				t.Errorf("pool settings = %+v, want configured values", cfg)
			}
		})
	}
}

func TestParsePoolConfigDoesNotExposeSelectedURL(t *testing.T) {
	const secret = "private-password"

	_, err := parsePoolConfig(&config.Config{
		DBDirectURL:    "postgres://user:direct-password@localhost:5432/iam",
		DBPoolURL:      "postgres://user:" + secret + "@localhost:invalid/iam",
		DBMaxOpenConns: 20,
		DBMinConns:     2,
	})
	if err == nil {
		t.Fatal("parsePoolConfig() error = nil, want parse error")
	}

	if strings.Contains(err.Error(), secret) {
		t.Errorf("parsePoolConfig() error exposes selected URL: %q", err)
	}
}

func TestDBCloseIsIdempotent(t *testing.T) {
	db, err := Open(context.Background(), &config.Config{
		DBDirectURL:    "postgres://user:password@127.0.0.1:1/iam?connect_timeout=1",
		DBMaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if db.SQLX.DriverName() != "pgx" {
		t.Errorf("driver = %q, want pgx", db.SQLX.DriverName())
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}
