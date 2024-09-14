package known

import (
	"vcassist-backend/cmd/linker-cli/commands/known/key"
	"vcassist-backend/cmd/linker-cli/commands/known/set"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "known",
	Short: "The 'known' subcommand allows you to access all known keys and sets.",
}

func init() {
	RootCmd.AddCommand(key.RootCmd)
	RootCmd.AddCommand(set.RootCmd)
}
