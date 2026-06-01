package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	// exposeRaw 收集 --expose 的原始字符串值（可重复使用，也支持逗号分隔）
	var exposeRaw []string

	cmd := &cobra.Command{
		Use:   "spawn <id> <cmd>",
		Short: "Spawn a process asynchronously inside a sandbox",
		Long: `Start a long-running process inside a sandbox and return immediately.

The process ID is printed to stdout. Use 'tsb logs' to view output.

Examples:
  tsb spawn sb-123 "npm run dev"
  tsb spawn sb-123 "npm run dev" --expose 5173
  tsb spawn sb-123 "python server.py" --expose 5173 --expose 3000
  tsb spawn sb-123 "vite" --expose 5173,3000
  PID=$(tsb spawn sb-123 "python server.py" --expose 8080)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, spawnCmd := args[0], args[1]

			// 解析 --expose：支持逗号分隔（--expose 5173,3000）和多次重复（--expose 5173 --expose 3000）
			exposePorts, err := parseExposeFlag(exposeRaw)
			if err != nil {
				return fmt.Errorf("--expose: %w", err)
			}

			cfgCtx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}

			pc, err := apiclient.NewProcessClientFromConfig(cfg, cfgCtx)
			if err != nil {
				return err
			}

			proc, err := pc.SpawnProcess(cmd.Context(), sandboxID, spawnCmd, exposePorts)
			if err != nil {
				return fmt.Errorf("spawn: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), proc.ID)
			return nil
		},
	}

	// StringArrayVar 支持多次 --expose（每次一个值）；parseExposeFlag 再处理逗号分隔
	cmd.Flags().StringArrayVar(&exposeRaw, "expose", nil,
		"声明进程对外暴露的端口，可重复或逗号分隔（如 --expose 5173 或 --expose 5173,3000）")

	return cmd
}

// parseExposeFlag 将 --expose 原始字符串列表解析为 int 端口切片。
// 支持两种写法：
//   - 多次重复：--expose 5173 --expose 3000 → exposeRaw = ["5173", "3000"]
//   - 逗号分隔：--expose 5173,3000         → exposeRaw = ["5173,3000"]
//   - 混合：    --expose 5173,3000 --expose 4000
//
// 不传时返回 nil（向后兼容，POST body 不含 expose_ports 字段）。
func parseExposeFlag(raw []string) ([]int, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var ports []int
	for _, item := range raw {
		// 每个 item 可能本身是逗号分隔列表
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			p, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("无效端口 %q：必须是正整数", part)
			}
			if p <= 0 || p > 65535 {
				return nil, fmt.Errorf("端口 %d 超出范围（1-65535）", p)
			}
			ports = append(ports, p)
		}
	}
	return ports, nil
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
