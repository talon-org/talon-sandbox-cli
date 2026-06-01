package commands

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewAgentRunCmd 返回 `tsb agent-run <id> --goal "..."` 命令。
// 对应 POST /v1/sandboxes/{id}/agent/run（Spec 38）。
// 同步阻塞，最长 5 分钟。
func NewAgentRunCmd(cfg *config.Config) *cobra.Command {
	var (
		goal      string
		maxSteps  int
		llmModel  string
		outputFmt string
	)

	cmd := &cobra.Command{
		Use:   "agent-run <id>",
		Short: "Run a high-level agent task in a sandbox",
		Long: `Execute a natural-language goal inside a sandbox using the browser-harness agent.

The command blocks until the agent finishes (up to 5 minutes). The result
and step trace are printed on completion.

--max-steps defaults to 20 (hard limit 100 on the server).
--model  sets the LLM hint, e.g. "anthropic:claude-sonnet-4-6".

Examples:
  tsb agent-run sb-123 --goal "open https://example.com and screenshot the page"
  tsb agent-run sb-123 --goal "get the Go latest version" --max-steps 5
  tsb agent-run sb-123 --goal "..." -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sandboxID := args[0]

			if goal == "" {
				return fmt.Errorf("--goal is required")
			}

			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			sb, err := talonsandbox.Get(cmd.Context(), sandboxID, clientOpts...)
			if err != nil {
				return wrapErr(err)
			}

			opts := talonsandbox.AgentRunOpts{
				MaxSteps: maxSteps,
				LLMModel: llmModel,
			}

			result, err := sb.AgentRun(cmd.Context(), goal, opts)
			if err != nil {
				return fmt.Errorf("agent-run: %w", err)
			}

			if outputFmt == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			// 文字汇总格式
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "run_id:     %s\n", result.RunID)
			fmt.Fprintf(w, "status:     %s\n", result.Status)
			fmt.Fprintf(w, "duration:   %dms\n", result.DurationMs)
			fmt.Fprintf(w, "exit_code:  %d\n", result.ExitCode)
			if result.Result != "" {
				fmt.Fprintf(w, "result:     %s\n", result.Result)
			}
			if result.Stderr != "" {
				fmt.Fprintf(w, "stderr:     %s\n", result.Stderr)
			}

			if len(result.Steps) > 0 {
				fmt.Fprintln(w, "\nsteps:")
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "  #\tACTION\tTHOUGHT")
				for _, s := range result.Steps {
					thought := s.Thought
					if len(thought) > 60 {
						thought = thought[:57] + "..."
					}
					fmt.Fprintf(tw, "  %d\t%s\t%s\n", s.Step, s.Action, thought)
				}
				tw.Flush() //nolint:errcheck
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&goal, "goal", "", "自然语言目标描述（必填）")
	cmd.Flags().IntVar(&maxSteps, "max-steps", 0, "最大步骤数（默认 20，上限 100）")
	cmd.Flags().StringVar(&llmModel, "model", "", "LLM 模型提示，如 anthropic:claude-sonnet-4-6")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "text", "输出格式：text|json")

	// goal 设为必填（配合自定义校验而非 MarkRequired，确保错误信息友好）
	return cmd
}
