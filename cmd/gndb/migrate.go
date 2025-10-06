package main

import "github.com/spf13/cobra"

func getMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Applies database migrations",
		Long:  "Applies all pending database migrations to bring the schema to the latest version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement migration logic
			return nil
		},
	}
	return cmd
}
