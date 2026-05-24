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

	// POST processes: immediately return exited process.
	mux.HandleFunc("/v1/sandboxes/sb-run/processes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":        "proc-1",
			"state":     "exited",
			"exit_code": 0,
		})
	})

	// GET process: return exited.
	mux.HandleFunc("/v1/sandboxes/sb-run/processes/proc-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"id":        "proc-1",
			"state":     "exited",
			"exit_code": 0,
		})
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
	})
	mux.HandleFunc("/v1/sandboxes/sb-sh/processes/proc-sh", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "proc-sh", "state": "exited", "exit_code": 0})
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
