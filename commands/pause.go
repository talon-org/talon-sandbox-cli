package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewPauseCmd returns the `tsb pause` command.
func NewPauseCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "pause <id>",
		Short: "Freeze all processes in a sandbox",
		Args:  cobra.ExactArgs(1),
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

			if err := sb.Pause(cmd.Context()); err != nil {
				return wrapErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "sandbox %s paused\n", id)
			return nil
		},
	}
}

// NewResumeCmd returns the `tsb resume` command.
func NewResumeCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <id>",
		Short: "Resume a paused sandbox",
		Args:  cobra.ExactArgs(1),
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

			if err := sb.Resume(cmd.Context()); err != nil {
				return wrapErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "sandbox %s resumed\n", id)
			return nil
		},
	}
}
