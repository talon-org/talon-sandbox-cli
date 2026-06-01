package commands_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"x.xgit.pro/dark/talon-sandbox-cli/commands"
)

// TestFsReadCmd 验证 `tsb fs read` 将远端文件内容输出到 stdout。
func TestFsReadCmd(t *testing.T) {
	mux := http.NewServeMux()

	// GET sandbox。
	mux.HandleFunc("/v1/sandboxes/sb-fs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-fs", "state": "running"})
	})

	// GET fs 文件。
	mux.HandleFunc("/v1/sandboxes/sb-fs/fs/app/config.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"debug":true}`)) //nolint:errcheck
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewFsCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"read", "sb-fs", "/app/config.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(buf.String(), `"debug":true`) {
		t.Errorf("输出内容有误，got: %s", buf.String())
	}
}

// TestFsLsCmd 验证 `tsb fs ls` 输出目录条目。
func TestFsLsCmd(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-ls", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-ls", "state": "running"})
	})

	mux.HandleFunc("/v1/sandboxes/sb-ls/fs-list/app", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": []map[string]any{
				{"name": "main.go", "size": 1234, "mod_time": 1700000000, "is_dir": false},
				{"name": "vendor", "size": 0, "mod_time": 1700000000, "is_dir": true},
			},
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewFsCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"ls", "sb-ls", "/app"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "main.go") {
		t.Errorf("输出缺少 main.go，got: %s", out)
	}
	// 目录应追加 /
	if !strings.Contains(out, "vendor/") {
		t.Errorf("输出缺少 vendor/，got: %s", out)
	}
}

// TestFsRmCmd 验证 `tsb fs rm` 发出 DELETE 请求并打印确认。
func TestFsRmCmd(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-rm", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-rm", "state": "running"})
	})

	rmCalled := false
	mux.HandleFunc("/v1/sandboxes/sb-rm/fs/tmp/cache", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rmCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewFsCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"rm", "sb-rm", "/tmp/cache"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !rmCalled {
		t.Error("DELETE /fs/tmp/cache 未被调用")
	}

	if !strings.Contains(buf.String(), "removed") {
		t.Errorf("输出中缺少 'removed'，got: %s", buf.String())
	}
}

// TestFsWriteCmd 验证 `tsb fs write <id> <path> <local-file>` 发出 PUT 请求并写入正确内容。
func TestFsWriteCmd(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/sandboxes/sb-wr", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-wr", "state": "running"})
	})

	var receivedBody []byte
	mux.HandleFunc("/v1/sandboxes/sb-wr/fs/app/test.txt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	})

	// 创建临时本地文件
	localFile := t.TempDir() + "/test.txt"
	if err := os.WriteFile(localFile, []byte("hello from test"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewFsCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"write", "sb-wr", "/app/test.txt", localFile})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if string(receivedBody) != "hello from test" {
		t.Errorf("服务端收到内容 %q，期望 %q", string(receivedBody), "hello from test")
	}

	if !strings.Contains(buf.String(), "written") {
		t.Errorf("输出中缺少 'written'，got: %s", buf.String())
	}
}
