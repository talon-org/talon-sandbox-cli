// Command talon-sandbox (also aliased as tsb) is the CLI for the talon-sandbox
// platform.
//
// Usage:
//
//	tsb [--server URL] [--api-key KEY] [--context NAME] [-o FORMAT] <command> [flags]
//
// Run `tsb --help` for the full command list.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"x.xgit.pro/dark/talon-sandbox-cli/commands"
	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

func main() {
	// Handle SIGINT/SIGTERM → exit 130.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		os.Exit(130)
	}()

	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var (
		serverOverride  string
		apiKeyOverride  string
		contextOverride string
		configPath      string
	)

	// Binary name detection: both "talon-sandbox" and "tsb" work as the binary name.
	// The root command's Use is always "tsb" for brevity in help output.
	binaryName := filepath.Base(os.Args[0])
	useName := "tsb"
	if binaryName == "talon-sandbox" {
		useName = "talon-sandbox"
	}

	// Apply env var defaults.
	serverOverride = os.Getenv("TALON_SANDBOX_SERVER")
	apiKeyOverride = os.Getenv("TALON_SANDBOX_API_KEY")
	contextOverride = os.Getenv("TALON_SANDBOX_CONTEXT")
	configPath = os.Getenv("TALON_SANDBOX_CONFIG")

	root := &cobra.Command{
		Use:   useName,
		Short: "CLI for the talon-sandbox platform",
		Long: `tsb controls sandbox lifecycle, authentication, and configuration.

Environment variables:
  TALON_SANDBOX_SERVER   — server URL (overrides --server)
  TALON_SANDBOX_API_KEY  — API key (overrides config/keyring)
  TALON_SANDBOX_CONTEXT  — config context name (overrides --context)
  TALON_SANDBOX_CONFIG   — config file path (overrides --config)`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetArgs(args)

	root.PersistentFlags().StringVar(&serverOverride, "server", serverOverride, "Server URL (overrides config context)")
	root.PersistentFlags().StringVar(&apiKeyOverride, "api-key", apiKeyOverride, "API key (overrides config/keyring)")
	root.PersistentFlags().StringVar(&contextOverride, "context", contextOverride, "Config context name to use")
	root.PersistentFlags().StringVar(&configPath, "config", configPath, "Config file path (default: ~/.config/talon-sandbox/config.yaml)")

	cfg := &config.Config{}

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// Re-apply env vars in case they were set after init.
		if s := os.Getenv("TALON_SANDBOX_SERVER"); s != "" && serverOverride == "" {
			serverOverride = s
		}
		if k := os.Getenv("TALON_SANDBOX_API_KEY"); k != "" && apiKeyOverride == "" {
			apiKeyOverride = k
		}

		loaded, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		*cfg = *loaded

		// Apply context override.
		if contextOverride != "" {
			cfg.CurrentContext = contextOverride
		}

		// Apply server override to current context.
		if serverOverride != "" {
			ctx, err := cfg.CurrentCtx()
			if err != nil {
				return err
			}
			ctx.Server = serverOverride
			cfg.UpsertContext(*ctx)
		}

		// Propagate API key override via env so apiclient picks it up.
		if apiKeyOverride != "" {
			os.Setenv("TALON_SANDBOX_API_KEY", apiKeyOverride)
		}

		return nil
	}

	root.AddCommand(
		commands.NewLoginCmd(cfg),
		commands.NewLogoutCmd(cfg),
		commands.NewWhoamiCmd(cfg),
		commands.NewContextCmd(cfg),

		commands.NewCreateCmd(cfg),
		commands.NewListCmd(cfg),
		commands.NewGetCmd(cfg),
		commands.NewRmCmd(cfg),

		commands.NewRunCmd(cfg),
		commands.NewSpawnCmd(cfg),
		commands.NewLogsCmd(cfg),
		commands.NewKillCmd(cfg),

		commands.NewExposeCmd(cfg),
		commands.NewUnexposeCmd(cfg),
		commands.NewExposedCmd(cfg),

		commands.NewPtyCmd(cfg),
		commands.NewCpCmd(cfg),
		commands.NewEnvCmd(cfg),

		commands.NewPauseCmd(cfg),
		commands.NewResumeCmd(cfg),

		commands.NewVersionCmd(cfg),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return commands.ExitCode(err)
	}
	return 0
}
