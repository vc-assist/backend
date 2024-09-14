package link

import (
	"fmt"
	"os"
	"vcassist-backend/cmd/linker-cli/globals"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <set 1> <key 1> <set 2> <key 2>",
	Short: "Add an explicit link from left to right.",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		set1 := args[0]
		key1 := args[1]
		set2 := args[2]
		key2 := args[3]

		_, err := client.AddExplicitLink(
			cmd.Context(),
			&connect.Request[linkerv1.AddExplicitLinkRequest]{
				Msg: &linkerv1.AddExplicitLinkRequest{
					Left: &linkerv1.ExplicitKey{
						Set: set1,
						Key: key1,
					},
					Right: &linkerv1.ExplicitKey{
						Set: set2,
						Key: key2,
					},
				},
			},
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}
