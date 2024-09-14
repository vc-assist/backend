package set

import (
	"fmt"
	"os"
	"vcassist-backend/cmd/linker-cli/globals"
	"vcassist-backend/cmd/linker-cli/utils"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(setsCmd)
}

var setsCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists the sets known to the linker.",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		res, err := client.GetKnownSets(
			cmd.Context(),
			&connect.Request[linkerv1.GetKnownSetsRequest]{
				Msg: &linkerv1.GetKnownSetsRequest{},
			},
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		t := utils.NewTable()
		t.AppendHeader(table.Row{"Set"})

		for _, s := range res.Msg.GetSets() {
			t.AppendRow(table.Row{s})
		}

		t.Render()
	},
}
