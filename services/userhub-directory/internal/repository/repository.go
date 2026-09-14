// Package repository provides database access and repository functionality.
package repository

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/viczem/userhub/services/userhub-directory/internal/config"
)

// Repository owns the runtime PostgreSQL pool and exposes sqlx for repository queries.
type Repository struct {
	db        *sqlx.DB
	accepting atomic.Bool
	pool      *pgxpool.Pool
	ping      func(context.Context) error
	close     sync.Once
	closeErr  error
}

// Open creates the runtime PostgreSQL pool without requiring the endpoint to be available.
func Open(ctx context.Context, cfg *config.Config) (*Repository, error) {
	poolConfig, err := parsePoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.New("create database pool")
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	db := &Repository{
		db:   sqlx.NewDb(sqlDB, "pgx"),
		pool: pool,
	}
	db.ping = db.db.PingContext
	db.accepting.Store(true)

	return db, nil
}

func parsePoolConfig(cfg *config.Config) (*pgxpool.Config, error) {
	databaseURL := cfg.DB.DirectURL

	pooled := cfg.DB.PoolURL != ""
	if pooled {
		databaseURL = cfg.DB.PoolURL
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("parse selected database configuration")
	}

	poolConfig.MaxConns = int32(cfg.DB.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.DB.MinConns)
	poolConfig.MaxConnLifetime = cfg.DB.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.DB.ConnMaxIdleTime

	if pooled {
		poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	return poolConfig, nil
}

// Close stops sqlx work before closing the underlying pgxpool.
func (db *Repository) Close() error {
	db.close.Do(func() {
		db.closeErr = db.db.Close()
		db.pool.Close()
	})

	return db.closeErr
}
