package commands

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	xterm "golang.org/x/term"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/apiclient"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/keyring"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewLoginCmd returns the `tsb login` command.
func NewLoginCmd(cfg *config.Config) *cobra.Command {
	var (
		server   string
		username string
		tenant   string
		apiKey   string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to a talon-sandbox server",
		Long: `Authenticate with a talon-sandbox-api server.

Use --api-key to store an API key in the OS keyring (recommended for CI).
Omit --api-key to use interactive username/password login.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}
			// --server flag overrides the context server.
			if server != "" {
				ctx.Server = server
			}
			if ctx.Server == "" {
				return fmt.Errorf("no server specified — use --server URL")
			}

			// API key path: store in keyring, no network call needed.
			if apiKey != "" {
				kr := keyring.New()
				if err := kr.Set(ctx.Name, apiKey); err != nil {
					return fmt.Errorf("login: store api key: %w", err)
				}
				ctx.Auth = config.Auth{
					Type:      config.AuthTypeAPIKey,
					APIKeyRef: keyring.RefForContext(ctx.Name),
				}
				cfg.UpsertContext(*ctx)
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "API key saved for context %q\n", ctx.Name)
				return nil
			}

			// Interactive login.
			if username == "" {
				fmt.Fprint(cmd.OutOrStdout(), "Username: ")
				fmt.Fscan(cmd.InOrStdin(), &username)
			}
			password, err := readPassword(cmd)
			if err != nil {
				return fmt.Errorf("login: %w", err)
			}

			ac := apiclient.NewAuthClient(ctx.Server, "")
			resp, err := ac.Login(cmd.Context(), username, password, tenant)
			if err != nil {
				return err
			}

			ctx.Auth = config.Auth{
				Type:      config.AuthTypeCookie,
				Cookie:    resp.Token,
				ExpiresAt: time.Unix(resp.ExpiresAt, 0),
			}
			if resp.TenantID != "" {
				ctx.Tenant = resp.TenantID
			}
			cfg.UpsertContext(*ctx)
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in to %s (tenant: %s)\n", ctx.Server, ctx.Tenant)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Server URL (overrides current context)")
	cmd.Flags().StringVarP(&username, "username", "u", "", "Username")
	cmd.Flags().StringVar(&tenant, "tenant", "", "Tenant name (optional)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Store an API key instead of interactive login")

	return cmd
}

// NewLogoutCmd returns the `tsb logout` command.
func NewLogoutCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out from the current server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			// Best-effort server-side logout.
			if ctx.Server != "" && (ctx.Auth.Cookie != "" || ctx.Auth.APIKeyRef != "") {
				var authKey string
				if ctx.Auth.Cookie != "" {
					authKey = ctx.Auth.Cookie
				} else if ctx.Auth.APIKeyRef != "" {
					ctxName, ok := keyring.ContextFromRef(ctx.Auth.APIKeyRef)
					if ok {
						k, _ := keyring.New().Get(ctxName)
						authKey = k
					}
				}
				ac := apiclient.NewAuthClient(ctx.Server, authKey)
				if err := ac.Logout(cmd.Context()); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: server logout failed: %v\n", err)
				}
			}

			// Clear API key from keyring if applicable.
			if ctx.Auth.Type == config.AuthTypeAPIKey {
				kr := keyring.New()
				if err := kr.Delete(ctx.Name); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: keyring delete: %v\n", err)
				}
			}

			ctx.Auth = config.Auth{}
			cfg.UpsertContext(*ctx)
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out")
			return nil
		},
	}
}

// NewWhoamiCmd returns the `tsb whoami` command.
func NewWhoamiCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current authenticated identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgCtx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			var authKey string
			if cfgCtx.Auth.Cookie != "" {
				authKey = cfgCtx.Auth.Cookie
			} else if cfgCtx.Auth.APIKeyRef != "" {
				ctxName, ok := keyring.ContextFromRef(cfgCtx.Auth.APIKeyRef)
				if ok {
					k, _ := keyring.New().Get(ctxName)
					authKey = k
				}
			}
			// Env key override.
			if k := os.Getenv("TALON_SANDBOX_API_KEY"); k != "" {
				authKey = k
			}

			ac := apiclient.NewAuthClient(cfgCtx.Server, authKey)
			resp, err := ac.Me(cmd.Context())
			if err != nil {
				return err
			}

			outFmt, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}
			w := output.New(cmd.OutOrStdout(), outFmt)
			return w.PrintWhoami(output.WhoamiRow{
				TenantID: resp.TenantID,
				Role:     resp.Role,
				Server:   cfgCtx.Server,
			}, resp)
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json")
	return cmd
}

// readPassword reads a password from the terminal (no echo) or from stdin.
func readPassword(cmd *cobra.Command) (string, error) {
	fd := int(syscall.Stdin)
	if xterm.IsTerminal(fd) {
		fmt.Fprint(cmd.OutOrStdout(), "Password: ")
		pw, err := xterm.ReadPassword(fd)
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return string(pw), nil
	}
	var pw string
	_, err := fmt.Fscan(cmd.InOrStdin(), &pw)
	if err != nil && !errors.Is(err, os.ErrClosed) {
		return "", fmt.Errorf("read password: %w", err)
	}
	return pw, nil
}
