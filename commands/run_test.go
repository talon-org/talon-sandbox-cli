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

// TestRunCmd_OutputPrinted verifies that stdout from the remote process
// is printed to the CLI's stdout.
func TestRunCmd_OutputPrinted(t *testing.T) {
	mux := http.NewServeMux()

	// GET sandbox.
	mux.HandleFunc("/v1/sandboxes/sb-run", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-run", "state": "running"})
	})

	// POST processes: create. GET processes: list (SDK v0.1.1+ polls via LIST).
	mux.HandleFunc("/v1/sandboxes/sb-run/processes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":        "proc-1",
				"state":     "exited",
				"exit_code": 0,
			})
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{
				"processes": []map[string]any{
					{"id": "proc-1", "state": "exited", "exit_code": 0},
				},
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET process logs.
	mux.HandleFunc("/v1/sandboxes/sb-run/processes/proc-1/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world\n"))
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewRunCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-run", "echo hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("expected 'hello world' in output, got: %s", buf.String())
	}
}

// TestRunCmd_CommandPassedToShell verifies the command is wrapped in /bin/sh -c.
func TestRunCmd_CommandPassedToShell(t *testing.T) {
	var capturedCmd []string

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-sh", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-sh", "state": "running"})
	})
	mux.HandleFunc("/v1/sandboxes/sb-sh/processes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if cmds, ok := body["command"].([]interface{}); ok {
				for _, c := range cmds {
					capturedCmd = append(capturedCmd, c.(string))
				}
			}
			writeJSON(w, http.StatusCreated, map[string]any{
				"id": "proc-sh", "state": "exited", "exit_code": 0,
			})
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{
				"processes": []map[string]any{
					{"id": "proc-sh", "state": "exited", "exit_code": 0},
				},
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/sandboxes/sb-sh/processes/proc-sh/logs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewRunCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-sh", "echo test"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// The SDK wraps Run commands in /bin/sh -c.
	if len(capturedCmd) < 3 || capturedCmd[0] != "/bin/sh" || capturedCmd[1] != "-c" {
		t.Errorf("expected /bin/sh -c <cmd>, got %v", capturedCmd)
	}
	if capturedCmd[2] != "echo test" {
		t.Errorf("capturedCmd[2] = %q; want %q", capturedCmd[2], "echo test")
	}
}

// TestSpawnCmd_PrintsProcessID verifies that `tsb spawn` prints the process ID.
func TestSpawnCmd_PrintsProcessID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-spn/processes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    "proc-xyz",
			"state": "running",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewSpawnCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-spn", "npm run dev"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "proc-xyz" {
		t.Errorf("spawn output = %q; want proc-xyz", got)
	}
}

// TestSpawnCmd_ExposePorts_SingleFlag 验证单个 --expose 5173 正确写入请求体。
func TestSpawnCmd_ExposePorts_SingleFlag(t *testing.T) {
	var capturedBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-exp/processes", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody) //nolint:errcheck
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    "proc-exp1",
			"state": "running",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewSpawnCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-exp", "npm run dev", "--expose", "5173"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	raw, ok := capturedBody["expose_ports"]
	if !ok {
		t.Fatalf("请求体缺少 expose_ports 字段，body=%v", capturedBody)
	}
	ports, ok := raw.([]any)
	if !ok {
		t.Fatalf("expose_ports 类型错误，got %T", raw)
	}
	if len(ports) != 1 || ports[0] != float64(5173) {
		t.Errorf("expose_ports = %v; want [5173]", ports)
	}
}

// TestSpawnCmd_ExposePorts_MultiFlag 验证 --expose 5173 --expose 3000（多次重复）。
func TestSpawnCmd_ExposePorts_MultiFlag(t *testing.T) {
	var capturedBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-exp2/processes", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody) //nolint:errcheck
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    "proc-exp2",
			"state": "running",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewSpawnCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-exp2", "vite", "--expose", "5173", "--expose", "3000"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	raw, ok := capturedBody["expose_ports"]
	if !ok {
		t.Fatalf("请求体缺少 expose_ports 字段，body=%v", capturedBody)
	}
	ports, ok := raw.([]any)
	if !ok {
		t.Fatalf("expose_ports 类型错误，got %T", raw)
	}
	if len(ports) != 2 || ports[0] != float64(5173) || ports[1] != float64(3000) {
		t.Errorf("expose_ports = %v; want [5173 3000]", ports)
	}
}

// TestSpawnCmd_ExposePorts_CommaSeparated 验证 --expose 5173,3000（逗号分隔）。
func TestSpawnCmd_ExposePorts_CommaSeparated(t *testing.T) {
	var capturedBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-exp3/processes", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody) //nolint:errcheck
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    "proc-exp3",
			"state": "running",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewSpawnCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-exp3", "vite", "--expose", "5173,3000"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	raw, ok := capturedBody["expose_ports"]
	if !ok {
		t.Fatalf("请求体缺少 expose_ports 字段，body=%v", capturedBody)
	}
	ports, ok := raw.([]any)
	if !ok {
		t.Fatalf("expose_ports 类型错误，got %T", raw)
	}
	if len(ports) != 2 || ports[0] != float64(5173) || ports[1] != float64(3000) {
		t.Errorf("expose_ports = %v; want [5173 3000]", ports)
	}
}

// TestSpawnCmd_NoExpose_BodyOmitsField 验证不传 --expose 时请求体不含 expose_ports（向后兼容）。
func TestSpawnCmd_NoExpose_BodyOmitsField(t *testing.T) {
	var capturedBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-noexp/processes", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody) //nolint:errcheck
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    "proc-noexp",
			"state": "running",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewSpawnCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-noexp", "npm run dev"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, has := capturedBody["expose_ports"]; has {
		t.Errorf("不应包含 expose_ports 字段，body=%v", capturedBody)
	}
}
