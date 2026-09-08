package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// ErrMissingCredentials is returned when a request needs authentication but no
// Console username/password were configured.
var ErrMissingCredentials = errors.New("console credentials not configured: set CONSOLE_USERNAME and CONSOLE_PASSWORD")

// ConsoleClient is a minimal REST client for the Console backend. It performs
// JWT login lazily and transparently re-authenticates once on a 401 response.
type ConsoleClient struct {
	baseURL  string
	username string
	password string
	http     *http.Client

	mu    sync.Mutex
	token string
}

// NewConsoleClient constructs a ConsoleClient from the provided configuration.
func NewConsoleClient(cfg Config) *ConsoleClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}, //nolint:gosec // opt-in for self-signed Console certs
	}

	return &ConsoleClient{
		baseURL:  cfg.ConsoleBaseURL,
		username: cfg.ConsoleUsername,
		password: cfg.ConsolePassword,
		http: &http.Client{
			Timeout:   cfg.RequestTimeout,
			Transport: transport,
		},
	}
}

// login exchanges the configured credentials for a JWT and caches it.
func (c *ConsoleClient) login(ctx context.Context) error {
	if c.username == "" || c.password == "" {
		return ErrMissingCredentials
	}

	payload, err := json.Marshal(map[string]string{
		"username": c.username,
		"password": c.password,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/authorize", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("authorize request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authorize failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decode authorize response: %w", err)
	}

	if out.Token == "" {
		return errors.New("authorize response did not contain a token")
	}

	c.mu.Lock()
	c.token = out.Token
	c.mu.Unlock()

	return nil
}

func (c *ConsoleClient) currentToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.token
}

// doJSON performs an authenticated request against the Console API, decoding a
// JSON response into out. It authenticates on first use and retries once after
// re-authenticating if the server rejects the cached token with a 401.
func (c *ConsoleClient) doJSON(ctx context.Context, method, path string, body any, out *json.RawMessage) error {
	if c.currentToken() == "" {
		if err := c.login(ctx); err != nil {
			return err
		}
	}

	raw, status, err := c.rawRequest(ctx, method, path, body)
	if err != nil {
		return err
	}

	if status == http.StatusUnauthorized {
		if err := c.login(ctx); err != nil {
			return err
		}

		raw, status, err = c.rawRequest(ctx, method, path, body)
		if err != nil {
			return err
		}
	}

	if status < 200 || status >= 300 {
		return fmt.Errorf("console API %s %s returned %d: %s", method, path, status, strings.TrimSpace(string(raw)))
	}

	if out != nil {
		if len(bytes.TrimSpace(raw)) == 0 {
			*out = json.RawMessage("null")
		} else {
			*out = json.RawMessage(raw)
		}
	}

	return nil
}

// rawRequest sends a single authenticated request and returns the raw body and
// status code without interpreting them.
func (c *ConsoleClient) rawRequest(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var reader io.Reader

	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}

		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Authorization", "Bearer "+c.currentToken())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return data, resp.StatusCode, nil
}

// get is a convenience wrapper for GET requests.
func (c *ConsoleClient) get(ctx context.Context, path string) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)

	return out, err
}

// post is a convenience wrapper for POST requests.
func (c *ConsoleClient) post(ctx context.Context, path string, body any) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.doJSON(ctx, http.MethodPost, path, body, &out)

	return out, err
}

// escapePath escapes a single path segment (e.g. a device GUID) so it is safe
// to embed in a URL.
func escapePath(segment string) string {
	return url.PathEscape(segment)
}
