// Package apiclient wraps the talon-sandbox SDK client with CLI-friendly helpers.
package apiclient

import (
	"errors"
	"fmt"
	"os"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/keyring"
)

// Opts returns the set of talonsandbox.Option values needed to build a client
// for the given config context. The caller can pass these to any SDK function
// that accepts ...Option, or use NewClient to build a Client directly.
func Opts(cfg *config.Config, ctx *config.Context) ([]talonsandbox.Option, error) {
	server := ctx.Server

	// Env override.
	if s := os.Getenv("TALON_SANDBOX_SERVER"); s != "" && server == "" {
		server = s
	}
	if server == "" {
		return nil, fmt.Errorf("no server configured — run `tsb login --server URL` or set TALON_SANDBOX_SERVER")
	}

	opts := []talonsandbox.Option{talonsandbox.WithBaseURL(server)}

	// API key: env takes precedence.
	if key := os.Getenv("TALON_SANDBOX_API_KEY"); key != "" {
		opts = append(opts, talonsandbox.WithAPIKey(key))
		return opts, nil
	}

	switch ctx.Auth.Type {
	case config.AuthTypeAPIKey:
		if ctx.Auth.APIKeyRef == "" {
			return nil, fmt.Errorf("context %q: api-key-ref is empty", ctx.Name)
		}
		ctxName, ok := keyring.ContextFromRef(ctx.Auth.APIKeyRef)
		if !ok {
			return nil, fmt.Errorf("context %q: unrecognised api-key-ref %q", ctx.Name, ctx.Auth.APIKeyRef)
		}
		kr := keyring.New()
		key, err := kr.Get(ctxName)
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				return nil, fmt.Errorf("context %q: api key not found in keyring — run `tsb login`", ctx.Name)
			}
			return nil, fmt.Errorf("context %q: keyring get: %w", ctx.Name, err)
		}
		opts = append(opts, talonsandbox.WithAPIKey(key))

	case config.AuthTypeCookie:
		if ctx.Auth.Cookie == "" {
			return nil, fmt.Errorf("context %q: not logged in — run `tsb login`", ctx.Name)
		}
		// JWT stored from login; use as Bearer.
		opts = append(opts, talonsandbox.WithAPIKey(ctx.Auth.Cookie))

	default:
		if ctx.Auth.Cookie != "" {
			opts = append(opts, talonsandbox.WithAPIKey(ctx.Auth.Cookie))
		}
		// Unauthenticated — some commands (version) don't need auth.
	}

	return opts, nil
}

// NewClient builds a talonsandbox.Client from the config context.
func NewClient(cfg *config.Config, ctx *config.Context) (*talonsandbox.Client, error) {
	opts, err := Opts(cfg, ctx)
	if err != nil {
		return nil, err
	}
	return talonsandbox.New("", opts...), nil
}

// ─── Error helpers ────────────────────────────────────────────────────────────

const (
	ExitCodeOK       = 0
	ExitCodeUser     = 1
	ExitCodeServer   = 2
	ExitCodeAuth     = 3
	ExitCodeNotFound = 4
)

// CLIError enriches an SDK error with an exit code.
type CLIError struct {
	err  error
	code int
}

func (e *CLIError) Error() string { return e.err.Error() }
func (e *CLIError) Unwrap() error { return e.err }
func (e *CLIError) Code() int     { return e.code }

// Wrap wraps an error with a CLI exit code inferred from the SDK error type.
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	code := exitCodeFor(err)
	return &CLIError{err: err, code: code}
}

// ExitCode extracts the exit code from an error, defaulting to 1.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var e *CLIError
	if errors.As(err, &e) {
		return e.code
	}
	return ExitCodeUser
}

func exitCodeFor(err error) int {
	if errors.Is(err, talonsandbox.ErrAuth) {
		return ExitCodeAuth
	}
	if errors.Is(err, talonsandbox.ErrNotFound) {
		return ExitCodeNotFound
	}
	if errors.Is(err, talonsandbox.ErrServer) {
		return ExitCodeServer
	}
	return ExitCodeUser
}
