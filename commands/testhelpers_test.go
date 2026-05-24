package commands_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// sandboxJSON is a minimal sandbox JSON response.
func sandboxJSON(id, state string) []byte {
	b, _ := json.Marshal(map[string]any{
		"id":    id,
		"state": state,
	})
	return b
}

// newTestServer returns a test HTTP server with a pre-wired mux.
// The caller registers handlers on the mux.
func newTestServer(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newTestConfig builds a minimal config pointing at serverURL.
func newTestConfig(t *testing.T, serverURL string) *config.Config {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")

	cfg := &config.Config{}
	cfg.UpsertContext(config.Context{
		Name:   config.DefaultContextName,
		Server: serverURL,
		Auth: config.Auth{
			Type:   config.AuthTypeAPIKey,
			Cookie: "test-api-key",
		},
	})
	cfg.CurrentContext = config.DefaultContextName

	// Write config so Load can read it back.
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	// Patch cfg path using Save.
	// We'll use env override instead since we can't set cfg.path directly.
	// Instead, set TALON_SANDBOX_API_KEY and TALON_SANDBOX_SERVER via env.
	t.Setenv("TALON_SANDBOX_SERVER", serverURL)
	t.Setenv("TALON_SANDBOX_API_KEY", "test-api-key")

	return cfg
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
