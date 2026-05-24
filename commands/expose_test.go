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

func TestExposeCmd_PrintsURL(t *testing.T) {
	mux := http.NewServeMux()

	// GET sandbox.
	mux.HandleFunc("/v1/sandboxes/sb-1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-1", "state": "running"})
	})

	// POST expose.
	mux.HandleFunc("/v1/sandboxes/sb-1/expose", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			writeJSON(w, http.StatusOK, map[string]any{
				"port": 5173,
				"url":  "https://preview.example.com/sb-1-5173",
			})
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"ports": []any{}})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewExposeCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-1", "5173"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "https://preview.example.com/sb-1-5173" {
		t.Errorf("output = %q; want expose URL", got)
	}
}

func TestExposeCmd_WithSign(t *testing.T) {
	var capturedBody map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-2", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-2", "state": "running"})
	})
	mux.HandleFunc("/v1/sandboxes/sb-2/expose", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			writeJSON(w, http.StatusOK, map[string]any{
				"port":   8080,
				"url":    "https://signed.example.com?token=abc",
				"signed": true,
			})
		}
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewExposeCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"sb-2", "8080", "--sign", "--ttl", "1h"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if v, ok := capturedBody["sign"].(bool); !ok || !v {
		t.Errorf("expected sign=true in request body, got %v", capturedBody)
	}
	if v, ok := capturedBody["ttl"].(string); !ok || v != "1h" {
		t.Errorf("expected ttl=1h in request body, got %v", capturedBody["ttl"])
	}
}

func TestExposeCmd_NotImplemented(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes/sb-3", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "sb-3", "state": "running"})
	})
	mux.HandleFunc("/v1/sandboxes/sb-3/expose", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var errBuf bytes.Buffer
	cmd := commands.NewExposeCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"sb-3", "3000"})

	// Should succeed (ErrNotImplemented is surfaced as warning, not error).
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success for ErrNotImplemented, got: %v", err)
	}

	if !strings.Contains(errBuf.String(), "not yet available") {
		t.Errorf("expected 'not yet available' warning, got: %s", errBuf.String())
	}
}
