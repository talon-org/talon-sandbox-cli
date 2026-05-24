package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/apiclient"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// BuildVersion is set via -ldflags at build time.
// go build -ldflags "-X x.xgit.pro/dark/talon-sandbox-cli/commands.BuildVersion=v1.0.0"
var BuildVersion = "dev"

// NewVersionCmd returns the `tsb version` command.
func NewVersionCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print client version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "talon-sandbox version %s (%s/%s)\n",
				BuildVersion, runtime.GOOS, runtime.GOARCH)

			ctx, err := cfg.CurrentCtx()
			if err != nil || ctx.Server == "" {
				return nil
			}

			// Best-effort server ping: call /v1/auth/me (always present, returns 401 if unauthed).
			var authKey string
			if ctx.Auth.Cookie != "" {
				authKey = ctx.Auth.Cookie
			}
			ac := apiclient.NewAuthClient(ctx.Server, authKey)
			if _, err := ac.Me(cmd.Context()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: cannot reach server %s: %v\n", ctx.Server, err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "server %s (reachable)\n", ctx.Server)
			}

			return nil
		},
	}
}
