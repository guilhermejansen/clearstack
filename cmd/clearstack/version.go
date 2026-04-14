package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guilhermejansen/clearstack/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if globalFlags.JSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
					"version": version.Version,
					"commit":  version.Commit,
					"date":    version.Date,
				})
			}
			fmt.Fprintln(cmd.OutOrStdout(), version.Full())
			return nil
		},
	}
}
