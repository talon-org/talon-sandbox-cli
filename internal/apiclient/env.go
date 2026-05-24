package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EnvClient makes direct env-var API calls for operations not yet in the SDK.
type EnvClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewEnvClient creates an EnvClient from server URL and API key.
func NewEnvClient(baseURL, apiKey string) *EnvClient {
	return &EnvClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// List retrieves all env vars for a sandbox.
func (e *EnvClient) List(ctx context.Context, sandboxID string) (map[string]string, error) {
	u := fmt.Sprintf("%s/v1/sandboxes/%s/env", e.baseURL, sandboxID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	e.setAuth(req)
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("env list: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("env list: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Try to unmarshal as map[string]string.
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		// Fallback: try {"env": {"KEY": "VALUE"}}.
		var wrapper struct {
			Env map[string]string `json:"env"`
		}
		if err2 := json.Unmarshal(body, &wrapper); err2 == nil && wrapper.Env != nil {
			return wrapper.Env, nil
		}
		return nil, fmt.Errorf("env list: decode: %w", err)
	}
	return result, nil
}

// Remove deletes an env var.
func (e *EnvClient) Remove(ctx context.Context, sandboxID, key string) error {
	u := fmt.Sprintf("%s/v1/sandboxes/%s/env/%s", e.baseURL, sandboxID, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	e.setAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("env remove: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("env remove: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// Set sets a single env var (direct HTTP, mirrors SDK's env.Set).
func (e *EnvClient) Set(ctx context.Context, sandboxID, key, value string) error {
	payload, _ := json.Marshal(map[string]string{"key": key, "value": value})
	u := fmt.Sprintf("%s/v1/sandboxes/%s/env", e.baseURL, sandboxID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	e.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("env set %q: %w", key, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("env set %q: HTTP %d: %s", key, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (e *EnvClient) setAuth(req *http.Request) {
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}
}
