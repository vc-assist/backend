package link

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
	RootCmd.AddCommand(listLinksCmd)
}

var listLinksCmd = &cobra.Command{
	Use:   "list <set 1> <set 2>",
	Short: "List existing explicit links between the two sets.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		set1 := args[0]
		set2 := args[1]

		res, err := client.GetExplicitLinks(cmd.Context(), &connect.Request[linkerv1.GetExplicitLinksRequest]{
			Msg: &linkerv1.GetExplicitLinksRequest{
				LeftSet:  set1,
				RightSet: set2,
			},
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		leftKeys := res.Msg.GetLeftKeys()
		rightKeys := res.Msg.GetRightKeys()

		t := utils.NewTable()
		t.AppendHeader(table.Row{set1, set2})
		for i := 0; i < len(leftKeys); i++ {
			t.AppendRow(table.Row{leftKeys[i], rightKeys[i]})
		}
		t.Render()
	},
}
