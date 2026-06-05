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

// init 把 CLI 真实版本号同步给 apiclient,供出站请求的 User-Agent
// (talon-sandbox-cli/<version>)使用。commands.BuildVersion 仍是唯一的
// -ldflags 注入点;反向让 apiclient import commands 会构成循环,故由这里单向
// 推送。BuildVersion 还是默认 "dev" 时不覆盖,让 apiclient 自行回落到
// go module 的 build info。
func init() {
	if BuildVersion != "" && BuildVersion != "dev" {
		apiclient.Version = BuildVersion
	}
}

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
