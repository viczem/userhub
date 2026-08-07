package rest

import (
	"bytes"
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
	}, testLogger())
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
	}, testLogger())
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
	}, testLogger())
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", maxBodyBytes)))
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewRESTLiveness(t *testing.T) {
	router := NewREST(&config.Config{HTTPMaxBodyBytes: 8 << 10}, testLogger())
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestMaxBodyBytesMiddlewareMakesBodyAvailableToHandler(t *testing.T) {
	const body = "request body"
	handler := maxBodyBytesMiddleware(len(body))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}, slog.New(slog.NewJSONHandler(&logs, nil)))
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
