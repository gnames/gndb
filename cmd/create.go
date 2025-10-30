/*
Copyright © 2025 Dmitry Mozzherin <dmozzherin@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gnames/gn"
	"github.com/gnames/gndb/internal/iodb"
	"github.com/gnames/gndb/internal/ioschema"
	"github.com/spf13/cobra"
)

// getCreateCmd returns the create command.
// Extracted as a function to facilitate testing and dynamic
// command registration.
func getCreateCmd() *cobra.Command {
	var forceCreate bool

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create database schema",
		Long: `Create the GNverifier database schema from scratch.

This command:
  1. Connects to PostgreSQL using configuration settings
  2. Checks for existing tables and prompts for confirmation
  3. Creates all base tables using GORM AutoMigrate
  4. Sets collation for scientific name sorting

Note: Fuzzy matching is handled by gnmatcher (external).
This database stores canonical forms for exact lookups.

Use --force to skip confirmation and drop existing tables.

Examples:
  gndb create
  gndb create --force
  gndb create -f`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, args, forceCreate)
		},
	}

	createCmd.Flags().BoolVarP(&forceCreate, "force", "f",
		false, "drop existing tables without confirmation")

	return createCmd
}

func runCreate(
	_ *cobra.Command,
	_ []string,
	force bool,
) error {
	ctx := context.Background()

	// Create database operator
	op := iodb.NewPgxOperator()
	if err := op.Connect(ctx, &cfg.Database); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}
	defer op.Close()

	gn.Info("Connected to database: %s@%s:%d/%s",
		cfg.Database.User, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.Database)

	// Check if database has existing tables
	hasTables, err := op.HasTables(ctx)
	if err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	// Handle existing tables
	if hasTables {
		if force {
			// Force flag set - drop without prompting
			gn.Info("Dropping all existing tables " +
				"(--force enabled)...")
			if err := op.DropAllTables(ctx); err != nil {
				gn.PrintErrorMessage(err)
				return err
			}
			gn.Info("All tables dropped")
		} else {
			// Prompt user for confirmation
			gn.Warn("\nWarning: Database contains " +
				"existing tables.")
			gn.Warn("Creating schema will drop ALL " +
				"existing tables and data.")
			fmt.Print("\nDo you want to continue? (yes/no): ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				gn.Warn("Failed to read user input")
				return err
			}

			response = strings.TrimSpace(
				strings.ToLower(response))
			if response != "yes" && response != "y" {
				gn.Info("Aborted. No changes made.")
				return nil
			}

			// User confirmed - drop tables
			gn.Info("Dropping all existing tables...")
			if err := op.DropAllTables(ctx); err != nil {
				gn.PrintErrorMessage(err)
				return err
			}
			gn.Info("All tables dropped")
		}
	}

	// Create schema manager
	sm := ioschema.NewManager(op)

	// Run GORM AutoMigrate to create schema
	gn.Info("Creating schema using GORM AutoMigrate...")
	if err := sm.Create(ctx, cfg); err != nil {
		gn.PrintErrorMessage(err)
		return err
	}

	gn.Info("\nDatabase schema creation complete!")
	gn.Info("\nNext steps:")
	gn.Info("  - Run 'gndb populate' to import data")
	gn.Info("  - Run 'gndb optimize' to create indexes")

	return nil
}
