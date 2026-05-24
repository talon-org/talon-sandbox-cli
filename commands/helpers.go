// Package commands implements the talon-sandbox CLI subcommands.
package commands

import (
	"errors"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/apiclient"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// sdkOpts resolves the current context and returns SDK client options.
func sdkOpts(cfg *config.Config) ([]talonsandbox.Option, error) {
	ctx, err := cfg.CurrentCtx()
	if err != nil {
		return nil, err
	}
	return apiclient.Opts(cfg, ctx)
}

// sdkClient resolves the current context and builds an SDK client.
func sdkClient(cfg *config.Config) (*talonsandbox.Client, error) {
	ctx, err := cfg.CurrentCtx()
	if err != nil {
		return nil, err
	}
	return apiclient.NewClient(cfg, ctx)
}

// wrapErr wraps an SDK error with a CLI exit code.
func wrapErr(err error) error {
	return apiclient.Wrap(err)
}

// ExitCode extracts the exit code from an error.
func ExitCode(err error) int {
	return apiclient.ExitCode(err)
}

// isNotImplemented reports whether err wraps ErrNotImplemented.
func isNotImplemented(err error) bool {
	return errors.Is(err, talonsandbox.ErrNotImplemented)
}
