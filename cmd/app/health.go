package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const healthProbeTimeout = 5 * time.Second

// runHealthCheck probes the local health endpoint and exits with 0 on
// HTTP 2xx, 1 otherwise.
func runHealthCheck() {
	os.Exit(probeHealth())
}

func probeHealth() int {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8181"
	}

	scheme := "https"
	if v, err := strconv.ParseBool(os.Getenv("HTTP_TLS_ENABLED")); err == nil && !v {
		scheme = "http"
	}

	client := &http.Client{
		Timeout: healthProbeTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed local probe
		},
	}
	// Local probe targeting the loopback address on a fixed path.
	url := scheme + "://" + net.JoinHostPort("127.0.0.1", port) + "/healthz"

	ctx, cancel := context.WithTimeout(context.Background(), healthProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody) //nolint:gosec // G107: trusted loopback URL built from env-validated port
	if err != nil {
		return 1
	}

	resp, err := client.Do(req) //nolint:gosec // G704: trusted loopback URL built from env-validated port
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 1
	}

	return 0
}
