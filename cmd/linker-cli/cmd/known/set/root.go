package set

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "set",
	Short: "The 'known set' subcommand works with known sets.",
}
