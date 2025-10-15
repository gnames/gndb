// Package main provides the gndb CLI application.
// gndb manages the lifecycle of the GNverifier PostgreSQL database.
package main

import (
	"os"
)

func main() {
	if err := getRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
