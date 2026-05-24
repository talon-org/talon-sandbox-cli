package commands

import (
	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewListCmd returns the `tsb list` command.
func NewListCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sandboxes",
		Long:    `List all sandboxes in the current tenant.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sandboxes, err := talonsandbox.List(cmd.Context(), talonsandbox.ListOpts{}, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			outFmt, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}
			w := output.New(cmd.OutOrStdout(), outFmt)

			rows := make([]output.SandboxRow, len(sandboxes))
			infos := make([]talonsandbox.SandboxInfo, len(sandboxes))
			for i, sb := range sandboxes {
				info := sb.Info()
				infos[i] = info
				rows[i] = output.SandboxRow{
					ID:        info.ID,
					State:     info.State,
					Image:     info.Image,
					Network:   info.NetworkPolicy,
					CPU:       output.FormatCPU(info.CPUMillis),
					Memory:    output.FormatMemory(info.MemoryBytes),
					CreatedAt: info.CreatedAt,
				}
			}

			return w.PrintSandboxes(rows, infos)
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json|id")
	return cmd
}
