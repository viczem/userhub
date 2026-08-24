package repository

import (
	"context"
	"errors"
	"time"
)

const readinessTimeout = 2 * time.Second

var errNotAcceptingWork = errors.New("service is not accepting work")

// Ready reports whether the database can serve work before its deadline.
func (db *DB) Ready(ctx context.Context) error {
	if !db.accepting.Load() {
		return errNotAcceptingWork
	}

	ctx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()

	if err := db.ping(ctx); err != nil {
		return err
	}

	if !db.accepting.Load() {
		return errNotAcceptingWork
	}

	return nil
}

// StopReadiness prevents this database from reporting ready during shutdown.
func (db *DB) StopReadiness() {
	db.accepting.Store(false)
}
