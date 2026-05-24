package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewExposeCmd returns the `tsb expose` command.
func NewExposeCmd(cfg *config.Config) *cobra.Command {
	var (
		sign      bool
		ttl       string
		subdomain string
	)

	cmd := &cobra.Command{
		Use:   "expose <id> <port>",
		Short: "Expose a port for external access",
		Long: `Register a port for external access and print the preview URL.

Use --sign to request a signed (expiring) URL.
Use --ttl to set the signed URL lifetime (e.g. "1h", "30m").
Use --subdomain to specify a custom subdomain prefix.

Note: if the server does not yet implement the expose endpoint, a warning
is printed and the command exits successfully.

Examples:
  tsb expose sb-123 5173
  tsb expose sb-123 5173 --sign --ttl 1h
  tsb expose sb-123 3000 --subdomain myapp`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]
			var port int
			if _, err := fmt.Sscanf(args[1], "%d", &port); err != nil {
				return fmt.Errorf("invalid port %q: %w", args[1], err)
			}

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			opts := talonsandbox.ExposeOpts{
				Sign:      sign,
				TTL:       ttl,
				Subdomain: subdomain,
			}

			url, err := sb.Expose(cmd.Context(), port, opts)
			if err != nil {
				if isNotImplemented(err) {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: expose endpoint not yet available on this server\n")
					return nil
				}
				return wrapErr(err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), url)
			return nil
		},
	}

	cmd.Flags().BoolVar(&sign, "sign", false, "Request a signed (expiring) preview URL")
	cmd.Flags().StringVar(&ttl, "ttl", "", "Signed URL lifetime (e.g. 1h). Only meaningful with --sign")
	cmd.Flags().StringVar(&subdomain, "subdomain", "", "Custom subdomain prefix (default: random)")
	return cmd
}

// NewUnexposeCmd returns the `tsb unexpose` command.
func NewUnexposeCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unexpose <id> <port>",
		Short: "Remove an explicit port exposure",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]
			var port int
			if _, err := fmt.Sscanf(args[1], "%d", &port); err != nil {
				return fmt.Errorf("invalid port %q: %w", args[1], err)
			}

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			if err := sb.Unexpose(cmd.Context(), port); err != nil {
				if isNotImplemented(err) {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: expose endpoint not yet available on this server\n")
					return nil
				}
				return wrapErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "port %d unexposed\n", port)
			return nil
		},
	}

	return cmd
}

// NewExposedCmd returns the `tsb exposed` command.
func NewExposedCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "exposed <id>",
		Short: "List exposed ports for a sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			ports, err := sb.Exposed(cmd.Context())
			if err != nil {
				if isNotImplemented(err) {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: expose endpoint not yet available on this server\n")
					return nil
				}
				return wrapErr(err)
			}

			outFmt, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}
			w := output.New(cmd.OutOrStdout(), outFmt)

			rows := make([]output.ExposedRow, len(ports))
			for i, p := range ports {
				rows[i] = output.ExposedRow{
					Port:      p.Port,
					URL:       p.URL,
					Signed:    p.Signed,
					ExpiresAt: p.ExpiresAt,
					Source:    p.Source,
				}
			}

			return w.PrintExposed(rows, ports)
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json")
	return cmd
}
