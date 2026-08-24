package rest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viczem/userhub/services/iam/internal/config"
)

func TestNewRESTRejectsRequestBodyExceedingLimit(t *testing.T) {
	const maxBodyBytes = 8 << 10

	router := NewREST(&config.Config{
		AppEnv:           config.AppEnvDevelopment,
		HTTPMaxBodyBytes: maxBodyBytes,
	}, testLogger(), ready)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", maxBodyBytes+1)))
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestNewRESTRejectsChunkedRequestBodyExceedingLimit(t *testing.T) {
	const maxBodyBytes = 8 << 10

	router := NewREST(&config.Config{
		AppEnv:           config.AppEnvDevelopment,
		HTTPMaxBodyBytes: maxBodyBytes,
	}, testLogger(), ready)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", maxBodyBytes+1)))
	req.ContentLength = -1
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", res.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestNewRESTAllowsRequestBodyWithinLimit(t *testing.T) {
	const maxBodyBytes = 8 << 10

	router := NewREST(&config.Config{
		AppEnv:           config.AppEnvDevelopment,
		HTTPMaxBodyBytes: maxBodyBytes,
	}, testLogger(), ready)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", maxBodyBytes)))
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewRESTLiveness(t *testing.T) {
	router := NewREST(&config.Config{HTTPMaxBodyBytes: 8 << 10}, testLogger(), ready)
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestNewRESTReadiness(t *testing.T) {
	tests := []struct {
		name      string
		readiness func(context.Context) error
		wantCode  int
	}{
		{name: "ready", readiness: ready, wantCode: http.StatusOK},
		{name: "database unavailable", readiness: func(context.Context) error { return errors.New("unavailable") }, wantCode: http.StatusServiceUnavailable},
		{name: "readiness unavailable", readiness: nil, wantCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewREST(&config.Config{HTTPMaxBodyBytes: 8 << 10}, testLogger(), tt.readiness)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

			if response.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", response.Code, tt.wantCode)
			}
		})
	}
}

func TestNewRESTReadinessRecoversWithoutRouterRestart(t *testing.T) {
	calls := 0
	router := NewREST(&config.Config{HTTPMaxBodyBytes: 8 << 10}, testLogger(), func(context.Context) error {
		calls++
		if calls == 1 {
			return errors.New("private-database-detail")
		}

		return nil
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if first.Code != http.StatusServiceUnavailable {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusServiceUnavailable)
	}

	if strings.Contains(first.Body.String(), "private-database-detail") {
		t.Errorf("readiness response exposes database detail: %q", first.Body.String())
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if second.Code != http.StatusOK {
		t.Errorf("second status = %d, want %d", second.Code, http.StatusOK)
	}
}

func TestNewRESTLivenessDoesNotCheckReadiness(t *testing.T) {
	called := false
	router := NewREST(&config.Config{HTTPMaxBodyBytes: 8 << 10}, testLogger(), func(context.Context) error {
		called = true

		return errors.New("unavailable")
	})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if called {
		t.Fatal("liveness checked readiness")
	}
}

func TestMaxBodyBytesMiddlewareMakesBodyAvailableToHandler(t *testing.T) {
	const body = "request body"

	handler := maxBodyBytesMiddleware(len(body))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actual, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(actual) != body {
			t.Errorf("body = %q, want %q", actual, body)
		}
	}))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestNewRESTLogsWithoutRequestBodyOrQuery(t *testing.T) {
	var logs bytes.Buffer

	router := NewREST(&config.Config{
		AppEnv:           config.AppEnvProduction,
		HTTPMaxBodyBytes: 8 << 10,
	}, slog.New(slog.NewJSONHandler(&logs, nil)), ready)
	request := httptest.NewRequest(http.MethodGet, "/health/live?token=secret-query", nil)
	request.Header.Set("User-Agent", "test-client")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	output := logs.String()
	if !strings.Contains(output, `"msg":"/health/live"`) {
		t.Errorf("request log = %q, want path", output)
	}

	if strings.Contains(output, "secret-query") {
		t.Errorf("request log exposes query value: %q", output)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func ready(context.Context) error {
	return nil
}
