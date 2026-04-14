package main

import (
	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/detectors"
)

func newCategoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "Inspect cleanup categories supported by clearstack",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List every registered detector category",
			RunE: func(cmd *cobra.Command, _ []string) error {
				all := detectors.Default.All()
				if globalFlags.JSON {
					out := make([]map[string]any, 0, len(all))
					for _, d := range all {
						out = append(out, map[string]any{
							"id":          d.ID(),
							"description": d.Description(),
							"safety":      d.Safety().String(),
							"strategy":    d.DefaultStrategy(),
							"supported":   d.PlatformSupported(),
						})
					}
					return writeJSON(cmd.OutOrStdout(), out)
				}
				for _, d := range all {
					printfln(cmd.OutOrStdout(), "%-22s [%s] %s", d.ID(), d.Safety(), d.Description())
				}
				return nil
			},
		},
	)
	return cmd
}
