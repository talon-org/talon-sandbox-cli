package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/apiclient"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewEnvCmd returns the `tsb env` command with sub-commands.
func NewEnvCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environment variables inside a sandbox",
	}

	cmd.AddCommand(
		newEnvGetCmd(cfg),
		newEnvSetCmd(cfg),
		newEnvListCmd(cfg),
		newEnvRmCmd(cfg),
	)

	return cmd
}

func newEnvGetCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id> <KEY>",
		Short: "Get an environment variable value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, key := args[0], args[1]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			val, err := sb.Env().Get(cmd.Context(), key)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

func newEnvSetCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "set <id> KEY=VALUE [KEY2=VALUE2 ...]",
		Short: "Set one or more environment variables",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]
			kvs := args[1:]

			cfgCtx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			ec, err := apiclient.NewEnvClientFromConfig(cfg, cfgCtx)
			if err != nil {
				return err
			}

			for _, kv := range kvs {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid KEY=VALUE %q", kv)
				}
				if err := ec.Set(cmd.Context(), sandboxID, parts[0], parts[1]); err != nil {
					return fmt.Errorf("set %s: %w", parts[0], err)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "set %d variable(s) on sandbox %s\n", len(kvs), sandboxID)
			return nil
		},
	}
}

func newEnvListCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "list <id>",
		Short: "List environment variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]

			cfgCtx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			ec, err := apiclient.NewEnvClientFromConfig(cfg, cfgCtx)
			if err != nil {
				return err
			}

			envVars, err := ec.List(cmd.Context(), sandboxID)
			if err != nil {
				return err
			}

			outFmt, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}
			w := output.New(cmd.OutOrStdout(), outFmt)

			rows := make([]output.EnvRow, 0, len(envVars))
			for k, v := range envVars {
				rows = append(rows, output.EnvRow{Key: k, Value: v})
			}

			return w.PrintEnv(rows, envVars)
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json")
	return cmd
}

func newEnvRmCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id> <KEY>",
		Short: "Remove an environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, key := args[0], args[1]

			cfgCtx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			ec, err := apiclient.NewEnvClientFromConfig(cfg, cfgCtx)
			if err != nil {
				return err
			}

			if err := ec.Remove(cmd.Context(), sandboxID, key); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "removed %s from sandbox %s\n", key, sandboxID)
			return nil
		},
	}
}
