package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewStartCmd 返回 `tsb start <id>` 命令。
// 对应后端 POST /v1/sandboxes/{id}/start（stopped→running，204 无 body）。
func NewStartCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "start <id>",
		Short: "Start a stopped sandbox",
		Long: `Transition a stopped sandbox back to the running state.

This is the inverse of 'tsb stop'. The sandbox processes are not restored
automatically; use 'tsb spawn' to restart any services.

Examples:
  tsb start sb-123`,
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

			if err := sb.Start(cmd.Context()); err != nil {
				return wrapErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "sandbox %s started\n", id)
			return nil
		},
	}
}

// NewStopCmd 返回 `tsb stop <id>` 命令。
// 对应后端 POST /v1/sandboxes/{id}/stop（running→stopped，204 无 body）。
// 与 pause 不同：stop 彻底停止沙箱，资源回收但数据保留；pause 只冻结进程不回收资源。
func NewStopCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a running sandbox",
		Long: `Gracefully stop a running sandbox (running → stopped).

Unlike 'tsb pause', stopping a sandbox reclaims compute resources while
keeping its data intact. Use 'tsb start' to bring it back up.

Examples:
  tsb stop sb-123`,
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

			if err := sb.Stop(cmd.Context()); err != nil {
				return wrapErr(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "sandbox %s stopped\n", id)
			return nil
		},
	}
}
