package commands

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewFsCmd 返回 `tsb fs` 命令组，包含 read/write/ls/rm 子命令。
// 这是对 `tsb cp` 的轻量补充：cp 做本地↔远端文件传输，
// fs 子命令侧重对沙箱内文件系统的直接操作（内容读写、目录列举、删除）。
func NewFsCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "Manage files in a sandbox filesystem",
		Long: `Inspect and manipulate files inside a sandbox.

Sub-commands:
  read   Print a file's contents to stdout
  write  Write stdin or a local file into the sandbox
  ls     List directory entries
  rm     Delete a file or directory

For copying files between local disk and sandbox use 'tsb cp'.`,
	}

	cmd.AddCommand(
		newFsReadCmd(cfg),
		newFsWriteCmd(cfg),
		newFsLsCmd(cfg),
		newFsRmCmd(cfg),
	)

	return cmd
}

// newFsReadCmd 返回 `tsb fs read <id> <remote-path>` 命令。
// 对应 GET /v1/sandboxes/{id}/fs/{path}，将文件内容输出到 stdout。
func newFsReadCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "read <id> <path>",
		Short: "Print a sandbox file to stdout",
		Long: `Read a file from the sandbox filesystem and print its contents to stdout.

Examples:
  tsb fs read sb-123 /app/config.json
  tsb fs read sb-123 /app/output.log > local.log`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, remotePath := args[0], args[1]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			data, err := sb.FS().Read(cmd.Context(), remotePath)
			if err != nil {
				return fmt.Errorf("fs read: %w", err)
			}

			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

// newFsWriteCmd 返回 `tsb fs write <id> <remote-path> [local-file]` 命令。
// 对应 PUT /v1/sandboxes/{id}/fs/{path}。
// local-file 省略时从 stdin 读取。
func newFsWriteCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "write <id> <path> [local-file]",
		Short: "Write a file into the sandbox",
		Long: `Write content into the sandbox filesystem.

If local-file is given, its content is written to the sandbox path.
If local-file is omitted, stdin is used as the content source.

Examples:
  tsb fs write sb-123 /app/config.json ./config.json
  echo '{"debug":true}' | tsb fs write sb-123 /app/config.json`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, remotePath := args[0], args[1]

			var data []byte
			var err error

			if len(args) == 3 {
				// 从指定本地文件读取
				data, err = os.ReadFile(args[2])
				if err != nil {
					return fmt.Errorf("fs write: read local file: %w", err)
				}
			} else {
				// 从 stdin 读取（适合管道用法）
				data, err = io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("fs write: read stdin: %w", err)
				}
			}

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			if err := sb.FS().Write(cmd.Context(), remotePath, data); err != nil {
				return fmt.Errorf("fs write: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "written %d bytes to %s:%s\n", len(data), sandboxID, remotePath)
			return nil
		},
	}
}

// newFsLsCmd 返回 `tsb fs ls <id> <remote-path>` 命令。
// 对应 GET /v1/sandboxes/{id}/fs-list/{path}，列出目录内容。
func newFsLsCmd(cfg *config.Config) *cobra.Command {
	var long bool

	cmd := &cobra.Command{
		Use:   "ls <id> <path>",
		Short: "List a directory in the sandbox",
		Long: `List the entries of a sandbox directory.

Use -l to show size and modification time.

Examples:
  tsb fs ls sb-123 /app
  tsb fs ls sb-123 /app -l`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, remotePath := args[0], args[1]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			entries, err := sb.FS().List(cmd.Context(), remotePath)
			if err != nil {
				return fmt.Errorf("fs ls: %w", err)
			}

			if long {
				// 长格式：带大小和修改时间
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "NAME\tSIZE\tMODIFIED\tTYPE")
				for _, e := range entries {
					typ := "file"
					if e.IsDir {
						typ = "dir"
					}
					modTime := "-"
					if e.ModTime > 0 {
						modTime = time.Unix(e.ModTime, 0).UTC().Format("2006-01-02 15:04:05")
					}
					sizeStr := fmt.Sprintf("%d", e.Size)
					if e.IsDir {
						sizeStr = "-"
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.Name, sizeStr, modTime, typ)
				}
				return tw.Flush()
			}

			// 短格式：仅名称（目录追加 /）
			for _, e := range entries {
				name := e.Name
				if e.IsDir {
					name += "/"
				}
				fmt.Fprintln(cmd.OutOrStdout(), name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&long, "long", "l", false, "显示详细信息（大小、修改时间、类型）")
	return cmd
}

// newFsRmCmd 返回 `tsb fs rm <id> <remote-path>` 命令。
// 对应 DELETE /v1/sandboxes/{id}/fs/{path}，删除文件或目录。
func newFsRmCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id> <path>",
		Short: "Remove a file or directory in the sandbox",
		Long: `Delete a file or directory from the sandbox filesystem.

Examples:
  tsb fs rm sb-123 /app/tmp/cache
  tsb fs rm sb-123 /app/old-build`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID, remotePath := args[0], args[1]

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			if err := sb.FS().Remove(cmd.Context(), remotePath); err != nil {
				return fmt.Errorf("fs rm: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "removed %s:%s\n", sandboxID, remotePath)
			return nil
		},
	}
}
