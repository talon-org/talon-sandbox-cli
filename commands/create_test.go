package commands_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"x.xgit.pro/dark/talon-sandbox-cli/commands"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

func TestCreateCmd_BasicFlags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    "sb-test",
			"state": "running",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewCreateCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"-o", "id"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "sb-test" {
		t.Errorf("output = %q; want sb-test", got)
	}
}

func TestCreateCmd_Resources(t *testing.T) {
	var capturedBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		writeJSON(w, http.StatusCreated, map[string]any{"id": "sb-res", "state": "running"})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewCreateCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--resources", "cpu=2,memory=4GiB", "--network", "allowlist", "-o", "id"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// cpu_millis should be 2000 (2 cores × 1000).
	if v, ok := capturedBody["cpu_millis"].(float64); !ok || v != 2000 {
		t.Errorf("cpu_millis = %v; want 2000", capturedBody["cpu_millis"])
	}

	// memory_bytes should be 4GiB = 4294967296.
	if v, ok := capturedBody["memory_bytes"].(float64); !ok || v != 4294967296 {
		t.Errorf("memory_bytes = %v; want 4294967296", capturedBody["memory_bytes"])
	}

	// network_policy should be mapped from "allowlist".
	if v, ok := capturedBody["network_policy"].(string); !ok || v != "restricted-egress" {
		t.Errorf("network_policy = %v; want restricted-egress", capturedBody["network_policy"])
	}
}

func TestCreateCmd_InvalidResources(t *testing.T) {
	cfg := &config.Config{}
	cfg.UpsertContext(config.Context{Name: config.DefaultContextName, Server: "http://localhost"})
	cfg.CurrentContext = config.DefaultContextName

	cmd := commands.NewCreateCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--resources", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid resources")
	}
}

func TestCreateCmd_TableOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":            "sb-tbl",
			"state":         "running",
			"cpu_millis":    1000,
			"memory_bytes":  1 << 30,
			"network_policy": "full-egress",
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewCreateCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	// Default output is table.
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "sb-tbl") {
		t.Errorf("expected 'sb-tbl' in table output:\n%s", out)
	}
}

func TestCreateCmd_WaitFlag(t *testing.T) {
	// --wait with a state other than "running" should not cause a crash.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]any{"id": "sb-wait", "state": "running"})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var errBuf bytes.Buffer
	cmd := commands.NewCreateCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--wait", "ready", "-o", "id"})

	// Even with an unsupported wait state, the command should complete (with a warning).
	_ = cmd.Execute()
}

// Compile-time check: NewCreateCmd must accept *config.Config.
var _ = func() *cobra.Command { return commands.NewCreateCmd(&config.Config{}) }
