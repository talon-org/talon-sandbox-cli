package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/apiclient"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewLogsCmd returns the `tsb logs` command.
func NewLogsCmd(cfg *config.Config) *cobra.Command {
	var (
		follow bool
		tail   int
	)

	cmd := &cobra.Command{
		Use:   "logs <id> <pid>",
		Short: "Fetch logs for a spawned process",
		Long: `Fetch the combined stdout+stderr log for a spawned process.

Use --follow to poll until the process exits.
Use --tail N to limit the number of bytes returned.

Examples:
  tsb logs sb-123 proc-abc
  tsb logs sb-123 proc-abc --follow
  tsb logs sb-123 proc-abc --tail 1024`,
		Args: cobra.ExactArgs(2),
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

			if !follow {
				data, err := pc.GetProcessLogs(cmd.Context(), sandboxID, procID, tail)
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), string(data))
				return nil
			}

			// --follow: poll at 1s intervals, printing newly appended bytes.
			seen := 0
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				proc, err := pc.GetProcess(cmd.Context(), sandboxID, procID)
				if err != nil {
					return err
				}

				data, err := pc.GetProcessLogs(cmd.Context(), sandboxID, procID, 0)
				if err != nil {
					return err
				}

				if len(data) > seen {
					fmt.Fprint(cmd.OutOrStdout(), string(data[seen:]))
					seen = len(data)
				}

				terminal := proc.State == "exited" || proc.State == "killed" || proc.State == "failed"
				if terminal {
					return nil
				}

				select {
				case <-cmd.Context().Done():
					return nil
				case <-ticker.C:
				}
			}
		},
	}

	cmd.Flags().BoolVar(&follow, "follow", false, "Poll log output until process exits (1s interval)")
	cmd.Flags().IntVar(&tail, "tail", 0, "Limit to last N bytes (0 = server default)")
	return cmd
}
