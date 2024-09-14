package key

import (
	"log"
	"strconv"
	"vcassist-backend/cmd/linker-cli/globals"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(delCmd)
}

var delCmd = &cobra.Command{
	Use:   "del <set> <before (unix seconds)>",
	Short: "Deletes all the keys of a certain set before a given unix time.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		before, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			log.Fatal(err)
		}

		_, err = client.DeleteKnownKeys(cmd.Context(), &connect.Request[linkerv1.DeleteKnownKeysRequest]{
			Msg: &linkerv1.DeleteKnownKeysRequest{
				Set:    args[0],
				Before: before,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	},
}
