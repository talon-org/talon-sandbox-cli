package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewRmCmd returns the `tsb rm` command.
func NewRmCmd(cfg *config.Config) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "rm <id>",
		Aliases: []string{"delete", "destroy"},
		Short:   "Remove (destroy) a sandbox",
		Long: `Permanently delete a sandbox.

Use --force to skip confirmation when deleting a running sandbox.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), id, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			if !force && sb.State() == "running" {
				fmt.Fprintf(cmd.ErrOrStderr(), "sandbox %s is running — use --force to destroy anyway\n", id)
				return fmt.Errorf("sandbox %s is running", id)
			}

			if err := sb.Kill(cmd.Context()); err != nil {
				return wrapErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "sandbox %s deleted\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force deletion without confirmation (even if running)")
	return cmd
}
