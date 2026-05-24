package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewContextCmd returns the `tsb context` command with sub-commands.
func NewContextCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage named server contexts",
		Long:  `Create, list, delete, and switch between named server contexts.`,
	}

	cmd.AddCommand(
		newContextListCmd(cfg),
		newContextUseCmd(cfg),
		newContextCreateCmd(cfg),
		newContextDeleteCmd(cfg),
	)

	return cmd
}

func newContextListCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}
			w := output.New(cmd.OutOrStdout(), fmt)

			rows := make([]output.ContextRow, len(cfg.Contexts))
			for i, ctx := range cfg.Contexts {
				rows[i] = output.ContextRow{
					Name:    ctx.Name,
					Server:  ctx.Server,
					Current: ctx.Name == cfg.CurrentContext,
				}
			}
			return w.PrintContexts(rows, cfg.Contexts)
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json")
	return cmd
}

func newContextUseCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			// Verify it exists.
			found := false
			for _, ctx := range cfg.Contexts {
				if ctx.Name == name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("context %q not found — run `tsb context create %s` first", name, name)
			}
			cfg.CurrentContext = name
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Switched to context %q\n", name)
			return nil
		},
	}
}

func newContextCreateCmd(cfg *config.Config) *cobra.Command {
	var server string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := config.Context{
				Name:   name,
				Server: server,
			}
			cfg.UpsertContext(ctx)
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Context %q created\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Server URL for this context")
	return cmd
}

func newContextDeleteCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if !cfg.RemoveContext(name) {
				return fmt.Errorf("context %q not found", name)
			}
			// If we deleted the current context, reset to default.
			if cfg.CurrentContext == name {
				cfg.CurrentContext = config.DefaultContextName
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Context %q deleted\n", name)
			return nil
		},
	}
}
