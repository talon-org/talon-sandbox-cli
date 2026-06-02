package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/output"
)

// NewCreateCmd returns the `tsb create` command.
func NewCreateCmd(cfg *config.Config) *cobra.Command {
	var (
		image       string
		resources   string // dict-style: cpu=2,memory=4GiB
		network     string
		idleTimeout string
		ttl         string
		waitState   string
		spawnCmd    string
		exposePort  int
		printURL    bool
		outputFmt   string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new sandbox",
		Long: `Create a new sandbox and optionally wait for it to reach a target state.

Examples:
  tsb create --image talon-alpine --wait running
  tsb create --resources cpu=2,memory=4GiB --network allowlist -o id
  tsb create --wait running --spawn "npm run dev" --expose 5173 --print-url`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			// Parse resources dict.
			cpu, memory, err := output.ParseResources(resources)
			if err != nil {
				return fmt.Errorf("--resources: %w", err)
			}

			opts := talonsandbox.Opts{
				Image:   image,
				Network: network,
				Resources: talonsandbox.Resources{
					CPU:    cpu,
					Memory: memory,
				},
				Timeout: idleTimeout,
				TTL:     ttl,
			}

			// If --wait is explicitly set to empty string, skip waiting.
			// Default "running" = the SDK will POST with ?wait=running.
			// For other states we print a warning since the SDK only supports "running".
			if waitState != "" && waitState != "running" {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: --wait %q not supported by SDK; only \"running\" is supported\n", waitState)
			}

			sb, err := talonsandbox.Create(cmd.Context(), opts, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			// If --spawn provided, spawn a process after create.
			if spawnCmd != "" {
				if _, err := sb.Spawn(cmd.Context(), spawnCmd); err != nil {
					return fmt.Errorf("spawn: %w", wrapErr(err))
				}
			}

			// If --expose provided, expose the port.
			var exposeURL string
			if exposePort > 0 {
				exposeURL, err = sb.Expose(cmd.Context(), exposePort)
				if err != nil && !isNotImplemented(err) {
					return fmt.Errorf("expose: %w", wrapErr(err))
				}
				if isNotImplemented(err) {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: expose not yet available on this server\n")
				}
			}

			// Output.
			outFmt, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}
			w := output.New(cmd.OutOrStdout(), outFmt)

			if outFmt == output.FormatID {
				w.PrintID(sb.ID())
			} else {
				info := sb.Info()
				if err := w.PrintSandbox(output.SandboxRow{
					ID:        info.ID,
					State:     info.State,
					Image:     info.Image,
					Network:   info.NetworkPolicy,
					CPU:       output.FormatCPU(info.CPUMillis),
					Memory:    output.FormatMemory(info.MemoryBytes),
					CreatedAt: info.CreatedAt,
				}, info); err != nil {
					return err
				}
			}

			// Print expose URL on its own line (after sandbox output).
			if printURL && exposeURL != "" {
				fmt.Fprintln(cmd.OutOrStdout(), exposeURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&image, "image", "", "Image name or ID")
	cmd.Flags().StringVar(&resources, "resources", "", `Resource allocation, e.g. "cpu=2,memory=4GiB"`)
	cmd.Flags().StringVar(&network, "network", "", "Network policy: allowlist|open|sealed")
	cmd.Flags().StringVar(&idleTimeout, "idle-timeout", "", "Idle timeout before auto-pause (e.g. 30m)")
	cmd.Flags().StringVar(&ttl, "ttl", "", "Hard time-to-live from creation (e.g. 6h)")
	cmd.Flags().StringVar(&waitState, "wait", "running", "Wait until sandbox reaches this state (running)")
	cmd.Flags().StringVar(&spawnCmd, "spawn", "", `Spawn a process after create+wait (e.g. "npm run dev")`)
	cmd.Flags().IntVar(&exposePort, "expose", 0, "Expose a port after create+wait (e.g. 5173)")
	cmd.Flags().BoolVar(&printURL, "print-url", false, "Print the expose URL on stdout (requires --expose)")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "Output format: table|json|id")

	return cmd
}
