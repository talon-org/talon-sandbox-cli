package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/apiclient"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewRunCmd returns the `tsb run` command.
// It executes a command synchronously, prints combined stdout/stderr, and
// exits with the process exit code.
func NewRunCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <id> <cmd>",
		Short: "Run a command synchronously inside a sandbox",
		Long: `Execute a shell command inside a sandbox and wait for it to complete.

The exit code of the remote process is propagated as the CLI exit code, making
this command composable in shell pipelines.

Examples:
  tsb run sb-123 "echo hello"
  tsb run sb-123 "npm test" && echo "tests passed"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, shellCmd := args[0], args[1]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), id, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			result, err := sb.Run(cmd.Context(), shellCmd)
			if err != nil {
				return wrapErr(err)
			}

			if result.Combined != "" {
				fmt.Fprint(cmd.OutOrStdout(), result.Combined)
			}

			if result.ExitCode != 0 {
				// Exit directly — cobra would print "Error:" prefix otherwise.
				os.Exit(result.ExitCode)
			}
			return nil
		},
	}

	return cmd
}

// NewSpawnCmd returns the `tsb spawn` command.
func NewSpawnCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn <id> <cmd>",
		Short: "Spawn a process asynchronously inside a sandbox",
		Long: `Start a long-running process inside a sandbox and return immediately.

The process ID is printed to stdout. Use 'tsb logs' to view output.

Examples:
  tsb spawn sb-123 "npm run dev"
  PID=$(tsb spawn sb-123 "python server.py")`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, spawnCmd := args[0], args[1]

			cfgCtx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			pc, err := apiclient.NewProcessClientFromConfig(cfg, cfgCtx)
			if err != nil {
				return err
			}

			proc, err := pc.SpawnProcess(cmd.Context(), sandboxID, spawnCmd)
			if err != nil {
				return fmt.Errorf("spawn: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), proc.ID)
			return nil
		},
	}

	return cmd
}

// NewKillCmd returns the `tsb kill` command.
func NewKillCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill <id> <pid>",
		Short: "Kill a spawned process",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, procID := args[0], args[1]

			cfgCtx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			pc, err := apiclient.NewProcessClientFromConfig(cfg, cfgCtx)
			if err != nil {
				return err
			}

			if err := pc.KillProcess(cmd.Context(), sandboxID, procID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "process %s killed\n", procID)
			return nil
		},
	}

	return cmd
}
