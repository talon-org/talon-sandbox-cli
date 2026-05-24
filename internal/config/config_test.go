package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

func TestDefaultPath(t *testing.T) {
	// Unset XDG_CONFIG_HOME to get the home-dir based path.
	os.Unsetenv("XDG_CONFIG_HOME")

	got := config.DefaultPath()
	if got == "" {
		t.Fatal("DefaultPath returned empty string")
	}
	if filepath.Base(filepath.Dir(got)) != "talon-sandbox" {
		t.Errorf("DefaultPath = %q; want .../talon-sandbox/config.yaml", got)
	}
}

func TestLoad_NotExist(t *testing.T) {
	cfg, err := config.Load("/tmp/talon-sandbox-cli-test-nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("Load non-existent: %v", err)
	}
	if cfg.CurrentContext != config.DefaultContextName {
		t.Errorf("CurrentContext = %q; want %q", cfg.CurrentContext, config.DefaultContextName)
	}
}

func TestLoadSaveRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")

	// Start from default.
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	// Add a context.
	cfg.UpsertContext(config.Context{
		Name:   "prod",
		Server: "https://api.example.com",
	})
	cfg.CurrentContext = "prod"

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload.
	cfg2, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg2.CurrentContext != "prod" {
		t.Errorf("CurrentContext = %q; want prod", cfg2.CurrentContext)
	}

	ctx, err := cfg2.CurrentCtx()
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Server != "https://api.example.com" {
		t.Errorf("ctx.Server = %q; want https://api.example.com", ctx.Server)
	}
}

func TestUpsertContext(t *testing.T) {
	cfg := &config.Config{}

	cfg.UpsertContext(config.Context{Name: "alpha", Server: "http://a"})
	cfg.UpsertContext(config.Context{Name: "beta", Server: "http://b"})
	cfg.UpsertContext(config.Context{Name: "alpha", Server: "http://a2"}) // overwrite

	if len(cfg.Contexts) != 2 {
		t.Errorf("Contexts len = %d; want 2", len(cfg.Contexts))
	}

	for _, ctx := range cfg.Contexts {
		if ctx.Name == "alpha" && ctx.Server != "http://a2" {
			t.Errorf("alpha server = %q; want http://a2", ctx.Server)
		}
	}
}

func TestRemoveContext(t *testing.T) {
	cfg := &config.Config{}
	cfg.UpsertContext(config.Context{Name: "x"})

	if !cfg.RemoveContext("x") {
		t.Fatal("RemoveContext returned false for existing context")
	}
	if cfg.RemoveContext("x") {
		t.Fatal("RemoveContext returned true for already-removed context")
	}
}

func TestCurrentCtx_AutoCreate(t *testing.T) {
	cfg := &config.Config{}
	ctx, err := cfg.CurrentCtx()
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Name != config.DefaultContextName {
		t.Errorf("ctx.Name = %q; want %q", ctx.Name, config.DefaultContextName)
	}
}

func TestCurrentCtx_NotFound(t *testing.T) {
	cfg := &config.Config{CurrentContext: "does-not-exist"}
	_, err := cfg.CurrentCtx()
	if err == nil {
		t.Fatal("expected error for missing non-default context")
	}
}
