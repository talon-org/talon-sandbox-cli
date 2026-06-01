package commands

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"

	"x.xgit.pro/dark/talon-sandbox-cli/internal/config"
)

// NewImagesCmd 返回 `tsb images` 命令。
// 对应 GET /v1/images，列出平台所有可用 baseimage。
func NewImagesCmd(cfg *config.Config) *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "images",
		Short: "List available sandbox base images",
		Long: `List all base images available on the platform.

The image ID or name can be passed to 'tsb create --image' when creating
a new sandbox.

Examples:
  tsb images
  tsb images -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientOpts, err := sdkOpts(cfg)
			if err != nil {
				return err
			}

			images, err := talonsandbox.ListImages(cmd.Context(), clientOpts...)
			if err != nil {
				return fmt.Errorf("images: %w", err)
			}

			switch outputFmt {
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(images)

			default:
				// 表格格式，对齐展示关键字段
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "ID\tNAME\tOS/ARCH\tSOURCE\tDEFAULT\tCREATED")
				for _, img := range images {
					def := ""
					if img.IsDefault {
						def = "yes"
					}
					osArch := fmt.Sprintf("%s/%s", img.OS, img.Arch)
					created := "-"
					if img.CreatedAt > 0 {
						created = time.Unix(img.CreatedAt, 0).UTC().Format("2006-01-02")
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
						img.ID, img.Name, osArch, img.Source, def, created)
				}
				return tw.Flush()
			}
		},
	}

	cmd.Flags().StringVarP(&outputFmt, "output", "o", "table", "输出格式：table|json")
	return cmd
}
