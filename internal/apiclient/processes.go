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

// ProcessInfo is a minimal process descriptor from the API.
type ProcessInfo struct {
	ID        string   `json:"id"`
	SandboxID string   `json:"sandbox_id"`
	Command   []string `json:"command"`
	PID       int32    `json:"pid"`
	State     string   `json:"state"`
	ExitCode  int32    `json:"exit_code"`
	StartedAt int64    `json:"started_at"`
	ExitedAt  int64    `json:"exited_at"`
}

// ProcessClient is a minimal direct HTTP client for process operations.
// Used when the SDK does not expose sufficient API surface (e.g. process ID).
type ProcessClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewProcessClient creates a ProcessClient from server URL and API key.
func NewProcessClient(baseURL, apiKey string) *ProcessClient {
	return &ProcessClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SpawnProcess calls POST /v1/sandboxes/{id}/processes and returns the ProcessInfo.
// exposePorts 为进程声明对外暴露的容器端口列表（如 []int{5173, 3000}）；
// 空/nil 时不发送该字段，保持向后兼容。
func (c *ProcessClient) SpawnProcess(ctx context.Context, sandboxID, command string, exposePorts []int) (*ProcessInfo, error) {
	body := map[string]any{
		"command": strings.Fields(command),
	}
	// 仅非空时写入，与服务端 StartProcessRequest.ExposePorts 语义对齐
	if len(exposePorts) > 0 {
		body["expose_ports"] = exposePorts
	}
	data, _ := json.Marshal(body)

	u := fmt.Sprintf("%s/v1/sandboxes/%s/processes", c.baseURL, sandboxID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spawn process: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("spawn process: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData)))
	}

	var info ProcessInfo
	if err := json.Unmarshal(respData, &info); err != nil {
		return nil, fmt.Errorf("spawn process: decode: %w", err)
	}
	return &info, nil
}

// GetProcess calls GET /v1/sandboxes/{id}/processes/{procID}.
func (c *ProcessClient) GetProcess(ctx context.Context, sandboxID, procID string) (*ProcessInfo, error) {
	u := fmt.Sprintf("%s/v1/sandboxes/%s/processes/%s", c.baseURL, sandboxID, procID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get process: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get process: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData)))
	}

	var info ProcessInfo
	if err := json.Unmarshal(respData, &info); err != nil {
		return nil, fmt.Errorf("get process: decode: %w", err)
	}
	return &info, nil
}

// GetProcessLogs calls GET /v1/sandboxes/{id}/processes/{procID}/logs.
func (c *ProcessClient) GetProcessLogs(ctx context.Context, sandboxID, procID string, tail int) ([]byte, error) {
	u := fmt.Sprintf("%s/v1/sandboxes/%s/processes/%s/logs", c.baseURL, sandboxID, procID)
	if tail > 0 {
		u += fmt.Sprintf("?tail=%d", tail)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/plain")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get process logs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read logs: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get process logs: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// KillProcess calls DELETE /v1/sandboxes/{id}/processes/{procID}.
func (c *ProcessClient) KillProcess(ctx context.Context, sandboxID, procID string) error {
	u := fmt.Sprintf("%s/v1/sandboxes/%s/processes/%s", c.baseURL, sandboxID, procID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kill process: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("kill process: HTTP %d", resp.StatusCode)
	}
	return nil
}
