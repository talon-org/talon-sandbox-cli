package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewBrowserCmd 返回 `tsb browser` 命令组，包含 start/get/stop 子命令。
// 对应后端 Spec 34 的 Browser CDP 端点。
func NewBrowserCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browser",
		Short: "Manage the headless browser session in a sandbox",
		Long: `Control the headless Chromium browser inside a sandbox (Spec 34).

Sub-commands:
  start  Launch a headless Chromium and return its CDP WebSocket URL
  get    Get the current browser session info
  stop   Stop and destroy the browser session`,
	}

	cmd.AddCommand(
		newBrowserStartCmd(cfg),
		newBrowserGetCmd(cfg),
		newBrowserStopCmd(cfg),
	)

	return cmd
}

// newBrowserStartCmd 返回 `tsb browser start <id>` 命令。
// 对应 POST /v1/sandboxes/{id}/browser，启动 headless Chromium 并返回 CDP URL。
func newBrowserStartCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "start <id>",
		Short: "Launch a headless Chromium in the sandbox",
		Long: `Start a headless Chromium instance inside a sandbox.

On success, prints the CDP WebSocket URL (cdp_ws_url) which can be used with
puppeteer, playwright, or any CDP-compatible client.

Examples:
  tsb browser start sb-123
  WS_URL=$(tsb browser start sb-123)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			sess, err := sb.Browser().Start(cmd.Context())
			if err != nil {
				return fmt.Errorf("browser start: %w", err)
			}

			// 输出 cdp_ws_url，供脚本直接 $(tsb browser start ...) 捕获
			fmt.Fprintln(cmd.OutOrStdout(), sess.CDPURL)
			return nil
		},
	}
}

// newBrowserGetCmd 返回 `tsb browser get <id>` 命令。
// 对应 GET /v1/sandboxes/{id}/browser，查询当前浏览器 session 状态。
func newBrowserGetCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get the current browser session info",
		Long: `Retrieve information about the running headless browser session.

Returns the CDP WebSocket URL and related details. Returns an error if no
browser session is running in the sandbox.

Examples:
  tsb browser get sb-123
  tsb browser get sb-123 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			sess, err := sb.Browser().Get(cmd.Context())
			if err != nil {
				return fmt.Errorf("browser get: %w", err)
			}

			if outputFmt == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(sess)
			}

			// 默认文字格式
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "sandbox_id:  %s\n", sess.SandboxID)
			fmt.Fprintf(w, "process_id:  %s\n", sess.ProcessID)
			fmt.Fprintf(w, "cdp_port:    %d\n", sess.CDPPort)
			fmt.Fprintf(w, "cdp_path:    %s\n", sess.CDPPath)
			fmt.Fprintf(w, "cdp_ws_url:  %s\n", sess.CDPURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "text", "输出格式：text|json")
	return cmd
}

// newBrowserStopCmd 返回 `tsb browser stop <id>` 命令。
// 对应 DELETE /v1/sandboxes/{id}/browser，停止并销毁浏览器 session。
func newBrowserStopCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop the browser session in a sandbox",
		Long: `Stop and destroy the headless browser session in a sandbox.

Examples:
  tsb browser stop sb-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			if err := sb.Browser().Stop(cmd.Context()); err != nil {
				return fmt.Errorf("browser stop: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "browser session stopped in sandbox %s\n", sandboxID)
			return nil
		},
	}
}
