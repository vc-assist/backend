package key

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "key",
	Short: "The 'known key' subcommand works with known keys.",
}
