package commands_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"x.xgit.pro/dark/talon-sandbox-cli/commands"
)

// TestAgentRunCmd_BasicGoal 验证 `tsb agent-run` 向正确端点 POST 并展示结果。
func TestAgentRunCmd_BasicGoal(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-agent", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-agent", "state": "running"})
	})

	var capturedGoal string
	mux.HandleFunc("/v1/sandboxes/sb-agent/agent/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		if g, ok := body["goal"].(string); ok {
			capturedGoal = g
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"run_id":      "run-001",
			"status":      "completed",
			"duration_ms": 1234,
			"exit_code":   0,
			"result":      "任务已完成",
			"steps": []map[string]any{
				{"step": 1, "action": "Page.navigate", "thought": "导航到目标页面"},
			},
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewAgentRunCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-agent", "--goal", "打开 https://example.com"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if capturedGoal != "打开 https://example.com" {
		t.Errorf("goal 传递有误，got: %q", capturedGoal)
	}

	out := buf.String()
	if !strings.Contains(out, "run-001") {
		t.Errorf("输出缺少 run_id，got:\n%s", out)
	}
	if !strings.Contains(out, "completed") {
		t.Errorf("输出缺少 status=completed，got:\n%s", out)
	}
}

// TestAgentRunCmd_RequiresGoal 验证未传 --goal 时返回错误。
func TestAgentRunCmd_RequiresGoal(t *testing.T) {
	cfg := newTestConfig(t, "http://localhost:9999")
	cmd := commands.NewAgentRunCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-xxx"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("期望缺少 --goal 时返回错误")
	}
}

// TestAgentRunCmd_MaxSteps 验证 --max-steps 正确传入请求体。
func TestAgentRunCmd_MaxSteps(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-ms", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-ms", "state": "running"})
	})

	var capturedMaxSteps float64
	mux.HandleFunc("/v1/sandboxes/sb-ms/agent/run", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		if v, ok := body["max_steps"].(float64); ok {
			capturedMaxSteps = v
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"run_id":      "run-ms",
			"status":      "completed",
			"duration_ms": 100,
			"exit_code":   0,
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewAgentRunCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-ms", "--goal", "test", "--max-steps", "5"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if capturedMaxSteps != 5 {
		t.Errorf("max_steps = %v，期望 5", capturedMaxSteps)
	}
}

// TestAgentRunCmd_JSONOutput 验证 `-o json` 输出完整 JSON 结构。
func TestAgentRunCmd_JSONOutput(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-aj", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-aj", "state": "running"})
	})

	mux.HandleFunc("/v1/sandboxes/sb-aj/agent/run", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"run_id":      "run-json",
			"status":      "failed",
			"duration_ms": 500,
			"exit_code":   1,
			"stderr":      "error detail",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewAgentRunCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-aj", "--goal", "test", "-o", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"run_id"`) {
		t.Errorf("JSON 输出缺少 run_id，got:\n%s", out)
	}
	if !strings.Contains(out, `"run-json"`) {
		t.Errorf("JSON 输出缺少 run-json，got:\n%s", out)
	}
}
