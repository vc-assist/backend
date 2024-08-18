package set

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
	Use:   "del <set 1> <set 2> ...",
	Short: "Deletes all the given sets.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		_, err := client.DeleteKnownSets(cmd.Context(), &connect.Request[linkerv1.DeleteKnownSetsRequest]{
			Msg: &linkerv1.DeleteKnownSetsRequest{
				Sets: args,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	},
}
