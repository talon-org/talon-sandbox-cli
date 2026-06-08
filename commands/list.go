package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewListCmd returns the `tsb list` command.
func NewListCmd(cfg *config.Config) *cobra.Command {
	var (
		outputFmt  string
		labelPairs []string // --label key=value，可重复;按 labels 过滤
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sandboxes",
		Long:    `List all sandboxes in the current tenant, optionally filtered by labels.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			// --label key=value 过滤(AND)。SDK 会把它们作为服务端 query 参数发出 +
			// 客户端兜底过滤。
			labels, err := parseLabels(labelPairs)
			if err != nil {
				return fmt.Errorf("--label: %w", err)
			}

			sandboxes, err := talonsandbox.List(cmd.Context(), talonsandbox.ListOpts{Labels: labels}, clientOpts...)
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
	cmd.Flags().StringArrayVar(&labelPairs, "label", nil, `Filter by label key=value (repeatable, AND), e.g. --label end_user_id=u_8821`)
	return cmd
}
