package edit

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "edit",
	Short: "The 'link edit' subcommand allows you to edit many explicit links at once in a git rebase-esque style.",
}
