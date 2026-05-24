package apiclient

import (
	"errors"
	"fmt"
	"os"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/keyring"
)

// resolveServerKey extracts the server URL and API key from a config context.
func resolveServerKey(ctx *config.Context) (server, apiKey string, err error) {
	server = ctx.Server
	if s := os.Getenv("TALON_SANDBOX_SERVER"); s != "" && server == "" {
		server = s
	}
	if server == "" {
		return "", "", fmt.Errorf("no server configured — run `tsb login --server URL` or set TALON_SANDBOX_SERVER")
	}

	// API key: env takes precedence.
	if key := os.Getenv("TALON_SANDBOX_API_KEY"); key != "" {
		return server, key, nil
	}

	switch ctx.Auth.Type {
	case config.AuthTypeAPIKey:
		ctxName, ok := keyring.ContextFromRef(ctx.Auth.APIKeyRef)
		if !ok {
			return "", "", fmt.Errorf("context %q: unrecognised api-key-ref", ctx.Name)
		}
		kr := keyring.New()
		key, kerr := kr.Get(ctxName)
		if kerr != nil {
			if errors.Is(kerr, keyring.ErrNotFound) {
				return "", "", fmt.Errorf("context %q: api key not found in keyring", ctx.Name)
			}
			return "", "", fmt.Errorf("context %q: keyring get: %w", ctx.Name, kerr)
		}
		return server, key, nil

	case config.AuthTypeCookie:
		if ctx.Auth.Cookie == "" {
			return "", "", fmt.Errorf("context %q: not logged in — run `tsb login`", ctx.Name)
		}
		return server, ctx.Auth.Cookie, nil

	default:
		if ctx.Auth.Cookie != "" {
			return server, ctx.Auth.Cookie, nil
		}
		return server, "", nil
	}
}

// NewProcessClientFromConfig builds a ProcessClient from the config context.
func NewProcessClientFromConfig(cfg *config.Config, ctx *config.Context) (*ProcessClient, error) {
	server, key, err := resolveServerKey(ctx)
	if err != nil {
		return nil, err
	}
	return NewProcessClient(server, key), nil
}

// NewEnvClientFromConfig builds an EnvClient from the config context.
func NewEnvClientFromConfig(cfg *config.Config, ctx *config.Context) (*EnvClient, error) {
	server, key, err := resolveServerKey(ctx)
	if err != nil {
		return nil, err
	}
	return NewEnvClient(server, key), nil
}
