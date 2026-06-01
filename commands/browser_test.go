package commands_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"x.xgit.pro/dark/talon-sandbox-cli/commands"
)

// TestBrowserStartCmd 验证 `tsb browser start` 打印 cdp_ws_url。
func TestBrowserStartCmd(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-br", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-br", "state": "running"})
	})

	mux.HandleFunc("/v1/sandboxes/sb-br/browser", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			writeJSON(w, http.StatusCreated, map[string]any{
				"sandbox_id": "sb-br",
				"process_id": "proc-cdp",
				"cdp_port":   9222,
				"cdp_path":   "/devtools/browser/abc",
				"cdp_ws_url": "wss://api.example.com/v1/sandboxes/sb-br/preview/9222/devtools/browser/abc",
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewBrowserCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"start", "sb-br"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "wss://") {
		t.Errorf("输出应为 wss:// URL，got: %s", out)
	}
}

// TestBrowserGetCmd 验证 `tsb browser get` 输出 session 信息。
func TestBrowserGetCmd(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-brg", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-brg", "state": "running"})
	})

	mux.HandleFunc("/v1/sandboxes/sb-brg/browser", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"sandbox_id": "sb-brg",
			"process_id": "proc-brg",
			"cdp_port":   9222,
			"cdp_path":   "/devtools/browser/xyz",
			"cdp_ws_url": "wss://api.example.com/preview/9222/devtools/browser/xyz",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewBrowserCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"get", "sb-brg"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "cdp_ws_url") {
		t.Errorf("输出缺少 cdp_ws_url，got: %s", out)
	}
	if !strings.Contains(out, "proc-brg") {
		t.Errorf("输出缺少 process_id proc-brg，got: %s", out)
	}
}

// TestBrowserStopCmd 验证 `tsb browser stop` 发出 DELETE 请求并打印确认。
func TestBrowserStopCmd(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-brs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-brs", "state": "running"})
	})

	stopCalled := false
	mux.HandleFunc("/v1/sandboxes/sb-brs/browser", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		stopCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewBrowserCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"stop", "sb-brs"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !stopCalled {
		t.Error("DELETE /browser 未被调用")
	}

	if !strings.Contains(buf.String(), "stopped") {
		t.Errorf("输出中缺少 'stopped'，got: %s", buf.String())
	}
}
