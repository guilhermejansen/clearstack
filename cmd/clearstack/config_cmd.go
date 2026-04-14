package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/guilhermejansen/clearstack/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the clearstack configuration file",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "path",
			Short: "Print the active config file path",
			RunE: func(cmd *cobra.Command, _ []string) error {
				p := globalFlags.ConfigPath
				if p == "" {
					p = config.DefaultConfigPath()
				}
				printfln(cmd.OutOrStdout(), "%s", p)
				return nil
			},
		},
		&cobra.Command{
			Use:   "init",
			Short: "Write a starter config file to the default location",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg := &config.Config{
					Version: 1,
					Profile: config.ProfileBalanced,
					Dormancy: config.Dormancy{
						MinAge:   "14d",
						CheckGit: true,
					},
					Docker: config.Docker{Enabled: true, BuildCache: true},
					UI:     config.UI{Theme: "auto", DefaultSort: "size"},
				}
				path := globalFlags.ConfigPath
				if path == "" {
					path = config.DefaultConfigPath()
				}
				if err := config.Save(path, cfg); err != nil {
					return err
				}
				printfln(cmd.OutOrStdout(), "wrote %s", path)
				return nil
			},
		},
		&cobra.Command{
			Use:   "show",
			Short: "Print the currently loaded configuration as YAML",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg, err := config.Load(globalFlags.ConfigPath)
				if err != nil {
					return err
				}
				if globalFlags.JSON {
					return writeJSON(cmd.OutOrStdout(), cfg)
				}
				b, err := yaml.Marshal(cfg)
				if err != nil {
					return fmt.Errorf("marshal: %w", err)
				}
				_, _ = cmd.OutOrStdout().Write(b)
				return nil
			},
		},
	)
	return cmd
}
