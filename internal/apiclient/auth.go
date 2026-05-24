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

// AuthClient makes direct auth API calls.
type AuthClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAuthClient creates an AuthClient.
func NewAuthClient(baseURL, apiKey string) *AuthClient {
	return &AuthClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoginResponse is returned by POST /v1/auth/login.
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	TenantID  string `json:"tenant_id"`
}

// MeResponse is returned by GET /v1/auth/me.
type MeResponse struct {
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
}

// Login calls POST /v1/auth/login.
func (a *AuthClient) Login(ctx context.Context, username, password, tenant string) (*LoginResponse, error) {
	body := map[string]any{
		"username": username,
		"password": password,
	}
	if tenant != "" {
		body["tenant"] = tenant
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/auth/login", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("login: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData)))
	}

	var out LoginResponse
	if err := json.Unmarshal(respData, &out); err != nil {
		return nil, fmt.Errorf("login: decode: %w", err)
	}
	return &out, nil
}

// Logout calls POST /v1/auth/logout.
func (a *AuthClient) Logout(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/auth/logout", nil)
	if err != nil {
		return err
	}
	a.setAuth(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	resp.Body.Close()
	return nil
}

// Me calls GET /v1/auth/me.
func (a *AuthClient) Me(ctx context.Context) (*MeResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/v1/auth/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	a.setAuth(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("me: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("me: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData)))
	}

	var out MeResponse
	if err := json.Unmarshal(respData, &out); err != nil {
		return nil, fmt.Errorf("me: decode: %w", err)
	}
	return &out, nil
}

func (a *AuthClient) setAuth(req *http.Request) {
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
}
