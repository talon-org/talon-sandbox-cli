package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewCpCmd returns the `tsb cp` command.
//
// Direction inferred from colon position:
//
//	"id:/remote/path" ./local  → download from sandbox
//	./local "id:/remote/path"  → upload to sandbox
func NewCpCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cp <src> <dst>",
		Short: "Copy files between local and sandbox filesystem",
		Long: `Copy files between local filesystem and a sandbox.

Use <id>:<path> to denote a sandbox path. The colon position determines
the direction of transfer.

Examples:
  tsb cp sb-123:/app/output.log ./output.log    # download from sandbox
  tsb cp ./data.json sb-123:/app/data.json      # upload to sandbox`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, dst := args[0], args[1]

			srcID, srcPath := parseCpPath(src)
			dstID, dstPath := parseCpPath(dst)

			switch {
			case srcID != "" && dstID != "":
				return fmt.Errorf("cp: remote-to-remote is not supported")
			case srcID == "" && dstID == "":
				return fmt.Errorf("cp: neither src nor dst is a sandbox path (use <id>:<path>)")
			}

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			if srcID != "" {
				// Download: sandbox → local.
				sb, err := talonsandbox.Get(cmd.Context(), srcID, clientOpts...)
				if err != nil {
					return wrapErr(err)
				}
				return cpDownload(cmd, sb, srcPath, dst)
			}

			// Upload: local → sandbox.
			sb, err := talonsandbox.Get(cmd.Context(), dstID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}
			return cpUpload(cmd, sb, src, dstPath)
		},
	}

	return cmd
}

// parseCpPath splits "id:/path" into (id, path). Returns ("", s) for local paths.
func parseCpPath(s string) (sandboxID, path string) {
	idx := strings.IndexByte(s, ':')
	if idx < 0 {
		return "", s
	}
	return s[:idx], s[idx+1:]
}

// cpDownload downloads remotePath from the sandbox to localPath.
func cpDownload(cmd *cobra.Command, sb *talonsandbox.Sandbox, remotePath, localPath string) error {
	fsHandle := sb.FS()

	// Try listing to detect directory.
	entries, listErr := fsHandle.List(cmd.Context(), remotePath)
	if listErr == nil {
		// It's a directory — recursive copy.
		if mkErr := os.MkdirAll(localPath, 0o755); mkErr != nil {
			return mkErr
		}
		for _, entry := range entries {
			remoteSub := strings.TrimRight(remotePath, "/") + "/" + entry.Name
			localSub := filepath.Join(localPath, entry.Name)
			if entry.IsDir {
				if err := cpDownload(cmd, sb, remoteSub, localSub); err != nil {
					return err
				}
			} else {
				if err := cpDownloadFile(cmd, sb, remoteSub, localSub); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Single file.
	return cpDownloadFile(cmd, sb, remotePath, localPath)
}

// cpDownloadFile downloads a single remote file to a local path.
func cpDownloadFile(cmd *cobra.Command, sb *talonsandbox.Sandbox, remotePath, localPath string) error {
	data, err := sb.FS().Read(cmd.Context(), remotePath)
	if err != nil {
		return fmt.Errorf("cp read %s: %w", remotePath, err)
	}
	// If local path is an existing directory, write inside it.
	if stat, statErr := os.Stat(localPath); statErr == nil && stat.IsDir() {
		localPath = filepath.Join(localPath, filepath.Base(remotePath))
	}
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		return fmt.Errorf("cp write %s: %w", localPath, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s:%s → %s\n", sb.ID(), remotePath, localPath)
	return nil
}

// cpUpload uploads localPath to the sandbox at remotePath.
func cpUpload(cmd *cobra.Command, sb *talonsandbox.Sandbox, localPath, remotePath string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("cp: %w", err)
	}
	if info.IsDir() {
		return cpUploadDir(cmd, sb, localPath, remotePath)
	}
	return cpUploadFile(cmd, sb, localPath, remotePath)
}

// cpUploadFile uploads a single local file.
func cpUploadFile(cmd *cobra.Command, sb *talonsandbox.Sandbox, localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("cp read %s: %w", localPath, err)
	}
	// If remotePath ends with /, append the local filename.
	if strings.HasSuffix(remotePath, "/") {
		remotePath += filepath.Base(localPath)
	}
	if err := sb.FS().Write(cmd.Context(), remotePath, data); err != nil {
		return fmt.Errorf("cp write %s:%s: %w", sb.ID(), remotePath, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s → %s:%s\n", localPath, sb.ID(), remotePath)
	return nil
}

// cpUploadDir recursively uploads a local directory.
func cpUploadDir(cmd *cobra.Command, sb *talonsandbox.Sandbox, localDir, remoteDir string) error {
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("cp readdir %s: %w", localDir, err)
	}
	for _, entry := range entries {
		localSub := filepath.Join(localDir, entry.Name())
		remoteSub := strings.TrimRight(remoteDir, "/") + "/" + entry.Name()
		if entry.IsDir() {
			if err := cpUploadDir(cmd, sb, localSub, remoteSub); err != nil {
				return err
			}
		} else {
			if err := cpUploadFile(cmd, sb, localSub, remoteSub); err != nil {
				return err
			}
		}
	}
	return nil
}
