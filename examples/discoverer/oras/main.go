package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:          "oras [command]",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		copyCmd(),
		discoverCmd(),
	)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
