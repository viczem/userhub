package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

const healthCheckTimeout = 3 * time.Second

func healthCheck(ctx context.Context, addr string) error {
	endpoint, err := healthCheckURL(addr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create readiness request: %w", err)
	}

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request readiness: %w", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness returned HTTP status %d", response.StatusCode)
	}

	return nil
}

func healthCheckURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("parse HTTP address: %w", err)
	}

	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}

	endpoint := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
		Path:   "/health/ready",
	}

	return endpoint.String(), nil
}
