package commands_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"x.xgit.pro/dark/talon-sandbox-cli/commands"
)

// TestImagesCmd_TableOutput 验证 `tsb images` 以表格格式列出 image。
func TestImagesCmd_TableOutput(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/images", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"images": []map[string]any{
				{
					"id":          "img-abc",
					"name":        "node:20-bookworm",
					"url":         "registry.example.com/node:20",
					"sha256":      "abc123",
					"os":          "linux",
					"arch":        "amd64",
					"source":      "builtin",
					"is_default":  true,
					"description": "Node.js 20 LTS",
					"created_at":  1700000000,
				},
				{
					"id":         "img-xyz",
					"name":       "python:3.12-slim",
					"url":        "registry.example.com/python:3.12",
					"sha256":     "def456",
					"os":         "linux",
					"arch":       "amd64",
					"source":     "builtin",
					"is_default": false,
					"created_at": 1700100000,
				},
			},
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewImagesCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "img-abc") {
		t.Errorf("输出缺少 img-abc，got:\n%s", out)
	}
	if !strings.Contains(out, "node:20-bookworm") {
		t.Errorf("输出缺少 node:20-bookworm，got:\n%s", out)
	}
	if !strings.Contains(out, "python:3.12-slim") {
		t.Errorf("输出缺少 python:3.12-slim，got:\n%s", out)
	}
	// 默认镜像标记
	if !strings.Contains(out, "yes") {
		t.Errorf("输出缺少默认标记 'yes'，got:\n%s", out)
	}
}

// TestImagesCmd_JSONOutput 验证 `tsb images -o json` 返回 JSON 数组。
func TestImagesCmd_JSONOutput(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/images", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"images": []map[string]any{
				{"id": "img-json", "name": "alpine:3.20", "os": "linux", "arch": "amd64",
					"source": "builtin", "is_default": false, "created_at": 0},
			},
		})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	var buf bytes.Buffer
	cmd := commands.NewImagesCmd(cfg)
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"-o", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"img-json"`) {
		t.Errorf("JSON 输出缺少 img-json，got:\n%s", out)
	}
}

// TestImagesCmd_Empty 验证镜像列表为空时不报错。
func TestImagesCmd_Empty(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/images", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"images": []any{}})
	})

	srv := newTestServer(t, mux)
	cfg := newTestConfig(t, srv.URL)

	cmd := commands.NewImagesCmd(cfg)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
}
