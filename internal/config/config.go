// Package config manages the talon-sandbox CLI configuration file at
// ~/.config/talon-sandbox/config.yaml (XDG base dir spec).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultContextName is the name of the default context.
	DefaultContextName = "default"

	// DefaultServer 是未经 login / 未设 TALON_SANDBOX_SERVER 时回落的官方托管
	// 端点。让用户只配 TALON_SANDBOX_API_KEY(或 login 拿到 key)即可直接使用,
	// 与各语言 SDK 的默认值保持一致。自部署用户用 `tsb login --server URL` 或
	// TALON_SANDBOX_SERVER 覆盖。
	DefaultServer = "https://api.sandbox.talon.net.cn"

	// AuthTypeCookie is cookie-based auth (username/password login).
	AuthTypeCookie = "cookie"
	// AuthTypeAPIKey is API key-based auth.
	AuthTypeAPIKey = "api-key"
)

// Auth holds authentication state for a context.
type Auth struct {
	Type      string    `yaml:"type"`
	Cookie    string    `yaml:"cookie,omitempty"`      // JWT token
	CSRF      string    `yaml:"csrf,omitempty"`        // CSRF token
	ExpiresAt time.Time `yaml:"expires-at,omitempty"`  // token expiry
	APIKeyRef string    `yaml:"api-key-ref,omitempty"` // keyring ref, never raw key
}

// Context represents a named server configuration.
type Context struct {
	Name   string `yaml:"name"`
	Server string `yaml:"server"`
	Auth   Auth   `yaml:"auth,omitempty"`
	Tenant string `yaml:"tenant,omitempty"`
}

// Config is the top-level configuration structure.
type Config struct {
	CurrentContext string    `yaml:"current-context"`
	Contexts       []Context `yaml:"contexts,omitempty"`

	path string // unexported: file path this config was loaded from
}

// Path returns the config file path.
func (c *Config) Path() string { return c.path }

// DefaultPath returns the default config file path under XDG_CONFIG_HOME.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "talon-sandbox", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "talon-sandbox", "config.yaml")
	}
	return filepath.Join(home, ".config", "talon-sandbox", "config.yaml")
}

// Load reads the config from path. If the file does not exist, a minimal
// default config is returned (not saved to disk).
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultConfig(path), nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: stat %s: %w", path, err)
	}

	// Enforce 0600 — world-readable config is dangerous.
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		return nil, fmt.Errorf(
			"config: %s has permissions %o; expected 0600 — fix with: chmod 600 %s",
			path, mode, path,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	cfg.path = path
	return &cfg, nil
}

// Save writes the config to its path with 0600 permissions.
func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(c.path), err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", c.path, err)
	}
	return nil
}

// CurrentCtx returns the active context by name.
// Returns an error if the context is not found (except for "default" which is auto-created).
func (c *Config) CurrentCtx() (*Context, error) {
	name := c.CurrentContext
	if name == "" {
		name = DefaultContextName
	}
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i], nil
		}
	}
	if name == DefaultContextName {
		ctx := Context{
			Name:   DefaultContextName,
			Server: "", // empty: must be set via login or env
		}
		c.Contexts = append(c.Contexts, ctx)
		c.CurrentContext = DefaultContextName
		return &c.Contexts[len(c.Contexts)-1], nil
	}
	return nil, fmt.Errorf("config: context %q not found", name)
}

// UpsertContext creates or replaces a context by name.
func (c *Config) UpsertContext(ctx Context) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == ctx.Name {
			c.Contexts[i] = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, ctx)
}

// RemoveContext deletes a context by name. Returns false if not found.
func (c *Config) RemoveContext(name string) bool {
	for i, ctx := range c.Contexts {
		if ctx.Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)
			return true
		}
	}
	return false
}

// defaultConfig returns a new Config with a single default context (no server).
func defaultConfig(path string) *Config {
	return &Config{
		CurrentContext: DefaultContextName,
		Contexts: []Context{
			{
				Name:   DefaultContextName,
				Server: "",
			},
		},
		path: path,
	}
}
