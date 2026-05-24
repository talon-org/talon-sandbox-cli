package commands

import (
	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewGetCmd returns the `tsb get` command.
func NewGetCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a sandbox by ID",
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

			outFmt, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}
			w := output.New(cmd.OutOrStdout(), outFmt)

			info := sb.Info()
			return w.PrintSandbox(output.SandboxRow{
				ID:        info.ID,
				State:     info.State,
				Image:     info.Image,
				Network:   info.NetworkPolicy,
				CPU:       output.FormatCPU(info.CPUMillis),
				Memory:    output.FormatMemory(info.MemoryBytes),
				CreatedAt: info.CreatedAt,
			}, info)
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json|id")
	return cmd
}
