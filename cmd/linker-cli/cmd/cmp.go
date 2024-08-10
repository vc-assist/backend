package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(cmpCmd)
}

var cmpCmd = &cobra.Command{
	Use:   "cmp <set 1> <set 2>",
	Short: "Show the probable links between two sets.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
		}
		// left := args[0]
		// todo: finish this command
	},
}
