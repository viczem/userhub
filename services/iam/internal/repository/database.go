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
	"github.com/viczem/userhub/services/iam/internal/config"
)

// DB owns the runtime PostgreSQL pool and exposes sqlx for repository queries.
type DB struct {
	SQLX *sqlx.DB

	accepting atomic.Bool
	pool      *pgxpool.Pool
	ping      func(context.Context) error
	close     sync.Once
	closeErr  error
}

// Open creates the runtime PostgreSQL pool without requiring the endpoint to be available.
func Open(ctx context.Context, cfg *config.Config) (*DB, error) {
	poolConfig, err := parsePoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.New("create database pool")
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	db := &DB{
		SQLX: sqlx.NewDb(sqlDB, "pgx"),
		pool: pool,
	}
	db.ping = db.SQLX.PingContext
	db.accepting.Store(true)

	return db, nil
}

func parsePoolConfig(cfg *config.Config) (*pgxpool.Config, error) {
	databaseURL := cfg.DBDirectURL

	pooled := cfg.DBPoolURL != ""
	if pooled {
		databaseURL = cfg.DBPoolURL
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("parse selected database configuration")
	}

	poolConfig.MaxConns = int32(cfg.DBMaxOpenConns)
	poolConfig.MinConns = int32(cfg.DBMinConns)
	poolConfig.MaxConnLifetime = cfg.DBConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.DBConnMaxIdleTime

	if pooled {
		poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	return poolConfig, nil
}

// Close stops sqlx work before closing the underlying pgxpool.
func (db *DB) Close() error {
	db.close.Do(func() {
		db.closeErr = db.SQLX.Close()
		db.pool.Close()
	})

	return db.closeErr
}
