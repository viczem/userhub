package repository

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDBReadinessRecovers(t *testing.T) {
	var calls atomic.Int32

	db := readyTestDB(func(context.Context) error {
		if calls.Add(1) == 1 {
			return errors.New("unavailable")
		}

		return nil
	})

	if err := db.Ready(context.Background()); err == nil {
		t.Fatal("first Ready() error = nil, want unavailable")
	}

	if err := db.Ready(context.Background()); err != nil {
		t.Fatalf("second Ready() error = %v, want recovery", err)
	}
}

func TestDBReadinessBoundsPing(t *testing.T) {
	db := readyTestDB(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("Ready() ping context has no deadline")
		}

		if remaining := time.Until(deadline); remaining <= 0 || remaining > readinessTimeout {
			t.Errorf("Ready() ping deadline remaining = %s, want at most %s", remaining, readinessTimeout)
		}

		return nil
	})

	if err := db.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
}

func TestDBStopReadinessPreventsFurtherPings(t *testing.T) {
	called := false
	db := readyTestDB(func(context.Context) error {
		called = true

		return nil
	})
	db.StopReadiness()

	if err := db.Ready(context.Background()); !errors.Is(err, errNotAcceptingWork) {
		t.Fatalf("Ready() error = %v, want not accepting work", err)
	}

	if called {
		t.Fatal("Ready() ping called after StopReadiness()")
	}
}

func TestDBDoesNotBecomeReadyDuringPingAfterStop(t *testing.T) {
	pingStarted := make(chan struct{})
	releasePing := make(chan struct{})
	db := readyTestDB(func(context.Context) error {
		close(pingStarted)
		<-releasePing

		return nil
	})

	result := make(chan error, 1)
	go func() {
		result <- db.Ready(context.Background())
	}()

	<-pingStarted
	db.StopReadiness()
	close(releasePing)

	if err := <-result; !errors.Is(err, errNotAcceptingWork) {
		t.Fatalf("Ready() error = %v, want not accepting work", err)
	}
}

func readyTestDB(ping func(context.Context) error) *DB {
	db := &DB{ping: ping}
	db.accepting.Store(true)

	return db
}
