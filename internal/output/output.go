// Package output renders command results in table / json / id formats.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
)

// Format is the output format requested by the user.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatID    Format = "id"
)

// ParseFormat parses a format string. Returns an error for unknown formats.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatTable, FormatJSON, FormatID:
		return Format(s), nil
	case "":
		return FormatTable, nil
	default:
		return "", fmt.Errorf("unknown output format %q; use table|json|id", s)
	}
}

// Writer writes formatted output to an io.Writer.
type Writer struct {
	w      io.Writer
	format Format
}

// New creates a new Writer.
func New(w io.Writer, format Format) *Writer {
	return &Writer{w: w, format: format}
}

// PrintJSON writes v as indented JSON.
func (w *Writer) PrintJSON(v any) error {
	enc := json.NewEncoder(w.w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintLine writes a single line terminated with newline.
func (w *Writer) PrintLine(s string) {
	fmt.Fprintln(w.w, s)
}

// PrintID prints an id and newline.
func (w *Writer) PrintID(id string) {
	fmt.Fprintln(w.w, id)
}

// ─── Sandbox ──────────────────────────────────────────────────────────────────

// SandboxRow is a single row in the sandbox list/get table.
type SandboxRow struct {
	ID        string
	State     string
	Image     string
	Network   string
	CPU       string
	Memory    string
	CreatedAt int64 // Unix seconds
}

// PrintSandboxes renders a sandbox list.
func (w *Writer) PrintSandboxes(rows []SandboxRow, raw any) error {
	switch w.format {
	case FormatJSON:
		return w.PrintJSON(raw)
	case FormatID:
		for _, r := range rows {
			fmt.Fprintln(w.w, r.ID)
		}
		return nil
	default: // table
		tw := tabwriter.NewWriter(w.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tSTATE\tIMAGE\tNETWORK\tCPU\tMEMORY\tCREATED")
		for _, r := range rows {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				r.ID, r.State, r.Image, r.Network, r.CPU, r.Memory, formatAge(r.CreatedAt))
		}
		return tw.Flush()
	}
}

// PrintSandbox renders a single sandbox.
func (w *Writer) PrintSandbox(row SandboxRow, raw any) error {
	switch w.format {
	case FormatJSON:
		return w.PrintJSON(raw)
	case FormatID:
		fmt.Fprintln(w.w, row.ID)
		return nil
	default: // table
		tw := tabwriter.NewWriter(w.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "FIELD\tVALUE")
		fmt.Fprintf(tw, "id\t%s\n", row.ID)
		fmt.Fprintf(tw, "state\t%s\n", row.State)
		fmt.Fprintf(tw, "image\t%s\n", row.Image)
		fmt.Fprintf(tw, "network\t%s\n", row.Network)
		fmt.Fprintf(tw, "cpu\t%s\n", row.CPU)
		fmt.Fprintf(tw, "memory\t%s\n", row.Memory)
		fmt.Fprintf(tw, "created\t%s\n", formatAge(row.CreatedAt))
		return tw.Flush()
	}
}

// ─── Whoami ───────────────────────────────────────────────────────────────────

// WhoamiRow is data for the whoami table.
type WhoamiRow struct {
	TenantID string
	Role     string
	Server   string
}

// PrintWhoami renders whoami output.
func (w *Writer) PrintWhoami(row WhoamiRow, raw any) error {
	switch w.format {
	case FormatJSON:
		return w.PrintJSON(raw)
	default:
		tw := tabwriter.NewWriter(w.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "FIELD\tVALUE")
		fmt.Fprintf(tw, "tenant\t%s\n", row.TenantID)
		fmt.Fprintf(tw, "role\t%s\n", row.Role)
		fmt.Fprintf(tw, "server\t%s\n", row.Server)
		return tw.Flush()
	}
}

// ─── Exposed ports ────────────────────────────────────────────────────────────

// ExposedRow is a single exposed port row.
type ExposedRow struct {
	Port      int
	URL       string
	Signed    bool
	ExpiresAt time.Time
	Source    string
}

// PrintExposed renders exposed-port list.
func (w *Writer) PrintExposed(rows []ExposedRow, raw any) error {
	switch w.format {
	case FormatJSON:
		return w.PrintJSON(raw)
	default:
		tw := tabwriter.NewWriter(w.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "PORT\tURL\tSIGNED\tSOURCE\tEXPIRES")
		for _, r := range rows {
			exp := "-"
			if !r.ExpiresAt.IsZero() {
				exp = r.ExpiresAt.Format(time.RFC3339)
			}
			fmt.Fprintf(tw, "%d\t%s\t%v\t%s\t%s\n",
				r.Port, r.URL, r.Signed, r.Source, exp)
		}
		return tw.Flush()
	}
}

// ─── Env vars ─────────────────────────────────────────────────────────────────

// EnvRow is a single env var row.
type EnvRow struct {
	Key   string
	Value string
}

// PrintEnv renders env var list.
func (w *Writer) PrintEnv(rows []EnvRow, raw any) error {
	switch w.format {
	case FormatJSON:
		return w.PrintJSON(raw)
	default:
		tw := tabwriter.NewWriter(w.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "KEY\tVALUE")
		for _, r := range rows {
			fmt.Fprintf(tw, "%s\t%s\n", r.Key, r.Value)
		}
		return tw.Flush()
	}
}

// ─── Context list ─────────────────────────────────────────────────────────────

// ContextRow is a single context row.
type ContextRow struct {
	Name    string
	Server  string
	Current bool
}

// PrintContexts renders context list.
func (w *Writer) PrintContexts(rows []ContextRow, raw any) error {
	switch w.format {
	case FormatJSON:
		return w.PrintJSON(raw)
	default:
		tw := tabwriter.NewWriter(w.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "\tNAME\tSERVER")
		for _, r := range rows {
			cur := " "
			if r.Current {
				cur = "*"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\n", cur, r.Name, r.Server)
		}
		return tw.Flush()
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// FormatCPU converts millicores to a human-readable string.
func FormatCPU(millis int64) string {
	if millis == 0 {
		return "-"
	}
	if millis%1000 == 0 {
		return fmt.Sprintf("%d", millis/1000)
	}
	return fmt.Sprintf("%dm", millis)
}

// FormatMemory converts bytes to a human-readable string.
func FormatMemory(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const (
		GiB = int64(1 << 30)
		MiB = int64(1 << 20)
		KiB = int64(1 << 10)
	)
	switch {
	case bytes >= GiB && bytes%GiB == 0:
		return fmt.Sprintf("%dGi", bytes/GiB)
	case bytes >= GiB:
		return fmt.Sprintf("%.1fGi", float64(bytes)/float64(GiB))
	case bytes >= MiB && bytes%MiB == 0:
		return fmt.Sprintf("%dMi", bytes/MiB)
	case bytes >= MiB:
		return fmt.Sprintf("%.1fMi", float64(bytes)/float64(MiB))
	case bytes >= KiB:
		return fmt.Sprintf("%dKi", bytes/KiB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// formatAge returns a human-readable age string from a Unix timestamp.
func formatAge(unixSec int64) string {
	if unixSec == 0 {
		return "-"
	}
	t := time.Unix(unixSec, 0)
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return t.Format("2006-01-02")
	}
}

// ParseResources parses a dict-style resources flag like "cpu=2,memory=4GiB".
func ParseResources(s string) (cpu float64, memory string, err error) {
	if s == "" {
		return 0, "", nil
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return 0, "", fmt.Errorf("invalid resources entry %q (expected key=value)", part)
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch strings.ToLower(k) {
		case "cpu":
			_, scanErr := fmt.Sscanf(v, "%f", &cpu)
			if scanErr != nil {
				return 0, "", fmt.Errorf("invalid cpu value %q", v)
			}
		case "memory", "mem":
			memory = v
		default:
			return 0, "", fmt.Errorf("unknown resource key %q (supported: cpu, memory)", k)
		}
	}
	return cpu, memory, nil
}
