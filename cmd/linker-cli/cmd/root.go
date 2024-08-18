package cmd

import (
	"context"
	"fmt"
	"os"
	"vcassist-backend/cmd/linker-cli/cmd/known"
	"vcassist-backend/cmd/linker-cli/cmd/link"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "linker-cli",
	Short: "linker-cli is a CLI interface for the VC Assist data linking service.",
}

func init() {
	rootCmd.AddCommand(link.RootCmd)
	rootCmd.AddCommand(known.RootCmd)
}

func ExecuteContext(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
