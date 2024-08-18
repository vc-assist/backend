package link

import (
	"vcassist-backend/cmd/linker-cli/cmd/link/edit"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "link",
	Short: "The 'link' subcommand allows for basic CRUD operations on the explicit links stored in the linker.",
}

func init() {
	RootCmd.AddCommand(edit.RootCmd)
}
