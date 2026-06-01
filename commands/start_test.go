package commands_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"x.xgit.pro/dark/talon-sandbox-cli/commands"
)

// TestStartCmd_SendsPost 验证 `tsb start` 向正确端点发 POST 并打印成功消息。
func TestStartCmd_SendsPost(t *testing.T) {
	mux := http.NewServeMux()

	// GET sandbox（SDK 先 Get 再调 Start）。
	mux.HandleFunc("/v1/sandboxes/sb-start", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-start", "state": "stopped"})
	})

	// POST start。
	startCalled := false
	mux.HandleFunc("/v1/sandboxes/sb-start/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		startCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewStartCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-start"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !startCalled {
		t.Error("POST /start 未被调用")
	}

	if !strings.Contains(buf.String(), "started") {
		t.Errorf("输出中缺少 'started'，got: %s", buf.String())
	}
}

// TestStopCmd_SendsPost 验证 `tsb stop` 向正确端点发 POST 并打印成功消息。
func TestStopCmd_SendsPost(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-stop", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-stop", "state": "running"})
	})

	stopCalled := false
	mux.HandleFunc("/v1/sandboxes/sb-stop/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		stopCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewStopCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-stop"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !stopCalled {
		t.Error("POST /stop 未被调用")
	}

	if !strings.Contains(buf.String(), "stopped") {
		t.Errorf("输出中缺少 'stopped'，got: %s", buf.String())
	}
}

// TestStartCmd_RequiresArg 验证缺少 id 参数时返回错误。
func TestStartCmd_RequiresArg(t *testing.T) {
	cfg := newTestConfig(t, "http://localhost:9999")
	cmd := commands.NewStartCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err == nil {
		t.Fatal("期望缺少参数时返回错误")
	}
}

// TestStopCmd_RequiresArg 验证缺少 id 参数时返回错误。
func TestStopCmd_RequiresArg(t *testing.T) {
	cfg := newTestConfig(t, "http://localhost:9999")
	cmd := commands.NewStopCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err == nil {
		t.Fatal("期望缺少参数时返回错误")
	}
}
