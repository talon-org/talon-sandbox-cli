package output_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    output.Format
		wantErr bool
	}{
		{"table", output.FormatTable, false},
		{"json", output.FormatJSON, false},
		{"id", output.FormatID, false},
		{"", output.FormatTable, false},
		{"yaml", "", true},
		{"invalid", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := output.ParseFormat(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ParseFormat(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatCPU(t *testing.T) {
	tests := []struct{ millis int64; want string }{
		{0, "-"},
		{1000, "1"},
		{2000, "2"},
		{500, "500m"},
		{1500, "1500m"},
	}
	for _, tc := range tests {
		got := output.FormatCPU(tc.millis)
		if got != tc.want {
			t.Errorf("FormatCPU(%d) = %q; want %q", tc.millis, got, tc.want)
		}
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct{ bytes int64; want string }{
		{0, "-"},
		{1 << 30, "1Gi"},
		{4 << 30, "4Gi"},
		{1 << 20, "1Mi"},
		{512 << 20, "512Mi"},
		{1024, "1Ki"},
		{100, "100B"},
	}
	for _, tc := range tests {
		got := output.FormatMemory(tc.bytes)
		if got != tc.want {
			t.Errorf("FormatMemory(%d) = %q; want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestPrintSandboxes_Table(t *testing.T) {
	var buf bytes.Buffer
	w := output.New(&buf, output.FormatTable)

	rows := []output.SandboxRow{
		{ID: "sb-1", State: "running", Image: "alpine", Network: "allowlist", CPU: "2", Memory: "4Gi", CreatedAt: 0},
	}
	if err := w.PrintSandboxes(rows, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "sb-1") {
		t.Errorf("expected sb-1 in output:\n%s", out)
	}
	if !strings.Contains(out, "running") {
		t.Errorf("expected 'running' in output:\n%s", out)
	}
}

func TestPrintSandboxes_ID(t *testing.T) {
	var buf bytes.Buffer
	w := output.New(&buf, output.FormatID)

	rows := []output.SandboxRow{
		{ID: "sb-aaa"},
		{ID: "sb-bbb"},
	}
	if err := w.PrintSandboxes(rows, nil); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), buf.String())
	}
	if lines[0] != "sb-aaa" || lines[1] != "sb-bbb" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestPrintSandboxes_JSON(t *testing.T) {
	var buf bytes.Buffer
	w := output.New(&buf, output.FormatJSON)

	if err := w.PrintSandboxes(nil, map[string]any{"test": "value"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"test"`) {
		t.Errorf("expected JSON output, got: %s", buf.String())
	}
}

func TestPrintExposed_Table(t *testing.T) {
	var buf bytes.Buffer
	w := output.New(&buf, output.FormatTable)

	rows := []output.ExposedRow{
		{Port: 5173, URL: "https://preview.example.com", Signed: false, Source: "explicit", ExpiresAt: time.Time{}},
	}
	if err := w.PrintExposed(rows, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "5173") {
		t.Errorf("expected port 5173 in output: %s", buf.String())
	}
}

func TestParseResources(t *testing.T) {
	tests := []struct {
		input   string
		wantCPU float64
		wantMem string
		wantErr bool
	}{
		{"", 0, "", false},
		{"cpu=2,memory=4GiB", 2, "4GiB", false},
		{"cpu=0.5,memory=512MiB", 0.5, "512MiB", false},
		{"cpu=1", 1, "", false},
		{"memory=8GiB", 0, "8GiB", false},
		{"bad", 0, "", true},
		{"cpu=abc", 0, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			cpu, mem, err := output.ParseResources(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err != nil {
				return
			}
			if cpu != tc.wantCPU {
				t.Errorf("cpu = %v; want %v", cpu, tc.wantCPU)
			}
			if mem != tc.wantMem {
				t.Errorf("mem = %q; want %q", mem, tc.wantMem)
			}
		})
	}
}
