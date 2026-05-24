package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	xterm "golang.org/x/term"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewPtyCmd returns the `tsb pty` command.
func NewPtyCmd(cfg *config.Config) *cobra.Command {
	var shellCmd string

	cmd := &cobra.Command{
		Use:   "pty <id>",
		Short: "Open an interactive terminal (PTY) inside a sandbox",
		Long: `Open a raw PTY session inside a sandbox.

The current terminal is put into raw mode while the session is active.
SIGWINCH is forwarded to the server for terminal resize support.
Press Ctrl+C or Ctrl+D to exit.

Examples:
  tsb pty sb-123
  tsb pty sb-123 --cmd "/bin/sh"`,
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

			termHandle := sb.Terminal()

			// Create a cancellable context.
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// Open the WebSocket PTY session.
			ptySession, err := termHandle.Open(ctx)
			if err != nil {
				return fmt.Errorf("pty: open: %w", err)
			}
			defer ptySession.Close(ctx)

			// Put stdin into raw mode.
			fd := int(os.Stdin.Fd())
			if !xterm.IsTerminal(fd) {
				return fmt.Errorf("pty: stdin is not a TTY")
			}

			oldState, err := xterm.MakeRaw(fd)
			if err != nil {
				return fmt.Errorf("pty: enter raw mode: %w", err)
			}
			defer xterm.Restore(fd, oldState)

			// Forward SIGWINCH for terminal resize.
			winchCh := make(chan os.Signal, 1)
			signal.Notify(winchCh, syscall.SIGWINCH)
			defer signal.Stop(winchCh)

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-winchCh:
						cols, rows, err := xterm.GetSize(fd)
						if err == nil {
							_ = ptySession.Resize(ctx, rows, cols)
						}
					}
				}
			}()

			// Send initial terminal size.
			if cols, rows, err := xterm.GetSize(fd); err == nil {
				_ = ptySession.Resize(ctx, rows, cols)
			}

			// Write incoming PTY data to stdout.
			closedCh := make(chan struct{})
			ptySession.OnData(func(b []byte) {
				os.Stdout.Write(b) //nolint:errcheck
			})
			ptySession.OnClose(func() {
				cancel()
				select {
				case <-closedCh:
				default:
					close(closedCh)
				}
			})

			// If a shell command was specified, send it as the first line.
			if shellCmd != "" {
				if err := ptySession.Write(ctx, []byte(shellCmd+"\n")); err != nil {
					return fmt.Errorf("pty: write initial cmd: %w", err)
				}
			}

			// Read stdin and forward to PTY in a goroutine.
			stdinDone := make(chan struct{})
			go func() {
				defer close(stdinDone)
				buf := make([]byte, 1024)
				for {
					n, err := os.Stdin.Read(buf)
					if n > 0 {
						if writeErr := ptySession.Write(ctx, buf[:n]); writeErr != nil {
							return
						}
					}
					if err != nil {
						return
					}
				}
			}()

			// Wait until PTY closes, stdin EOF, or context done.
			select {
			case <-closedCh:
			case <-stdinDone:
			case <-ctx.Done():
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&shellCmd, "cmd", "", "Command to run instead of server default shell")
	return cmd
}
