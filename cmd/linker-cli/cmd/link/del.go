package link

import (
	"log"
	"vcassist-backend/cmd/linker-cli/globals"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(delCmd)
}

var delCmd = &cobra.Command{
	Use:   "del <set 1> <key 1> <set 2> <key 2>",
	Short: "Deletes an explicit link from left to right.",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		set1 := args[0]
		key1 := args[1]
		set2 := args[2]
		key2 := args[3]

		_, err := client.DeleteExplicitLink(cmd.Context(), &connect.Request[linkerv1.DeleteExplicitLinkRequest]{
			Msg: &linkerv1.DeleteExplicitLinkRequest{
				Left: &linkerv1.ExplicitKey{
					Set: set1,
					Key: key1,
				},
				Right: &linkerv1.ExplicitKey{
					Set: set2,
					Key: key2,
				},
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	},
}
