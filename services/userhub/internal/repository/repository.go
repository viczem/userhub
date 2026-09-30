// Package repository provides database access and repository functionality.
package repository

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/viczem/userhub/services/userhub/internal/config"
)

// Repository provides database access within a transaction.
type Repository struct {
	tx pgx.Tx
}

// Commit commits the transaction.
func (repo Repository) Commit(ctx context.Context) error {
	return repo.tx.Commit(ctx)
}

// Rollback rolls back the transaction.
func (repo Repository) Rollback(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	err := repo.tx.Rollback(ctx)
	if errors.Is(err, pgx.ErrTxClosed) {
		return nil
	}

	return err
}

// Close rolls back the transaction and logs any error.
func (repo Repository) Close(ctx context.Context) {
	if err := repo.Rollback(ctx); err != nil {
		slog.Default().ErrorContext(
			ctx,
			"rollback",
			"error", err,
		)
	}
}

// Database owns the runtime PostgreSQL pool.
type Database struct {
	accepting atomic.Bool
	pool      *pgxpool.Pool
	ping      func(context.Context) error
	close     sync.Once
}

// Open creates the runtime PostgreSQL pool without requiring the endpoint to be available.
func Open(ctx context.Context, cfg *config.Config) (*Database, error) {
	poolConfig, err := parsePoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.New("create database pool")
	}

	db := &Database{pool: pool}
	db.ping = pool.Ping
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

	poolConfig.MaxConns = cfg.DB.MaxOpenConns
	poolConfig.MinConns = cfg.DB.MinConns
	poolConfig.MaxConnLifetime = cfg.DB.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.DB.ConnMaxIdleTime

	if pooled {
		poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	return poolConfig, nil
}

// Close closes the PostgreSQL pool.
func (db *Database) Close() {
	db.close.Do(db.pool.Close)
}

// NewRepository begins a transaction and returns a repository backed by it.
func (db *Database) NewRepository(ctx context.Context) (*Repository, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Repository{tx: tx}, nil
}
