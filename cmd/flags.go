package cmd

import (
	"fmt"
	"os"

	gndb "github.com/gnames/gndb/pkg"
	"github.com/spf13/cobra"
)

type funcFlag func(cmd *cobra.Command)

func versionFlag(cmd *cobra.Command) {
	hasVersionFlag, _ := cmd.Flags().GetBool("version")
	if hasVersionFlag {
		fmt.Printf("\nversion: %s\nbuild: %s\n\n", gndb.Version, gndb.Build)
		os.Exit(0)
	}
}
