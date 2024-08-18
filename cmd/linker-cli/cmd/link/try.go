package link

import (
	"log"
	"vcassist-backend/cmd/linker-cli/globals"
	"vcassist-backend/cmd/linker-cli/utils"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(linkCmd)
}

var linkCmd = &cobra.Command{
	Use:   "try <set 1> <set 2>",
	Short: "Run linking between the known keys of two known sets.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		left := args[0]
		right := args[1]

		leftRes, err := client.GetKnownKeys(
			cmd.Context(),
			&connect.Request[linkerv1.GetKnownKeysRequest]{
				Msg: &linkerv1.GetKnownKeysRequest{
					Set: left,
				},
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		rightRes, err := client.GetKnownKeys(
			cmd.Context(),
			&connect.Request[linkerv1.GetKnownKeysRequest]{
				Msg: &linkerv1.GetKnownKeysRequest{
					Set: right,
				},
			},
		)
		if err != nil {
			log.Fatal(err)
		}

		leftKeys := make([]string, len(leftRes.Msg.GetKeys()))
		for i, val := range leftRes.Msg.GetKeys() {
			leftKeys[i] = val.GetKey()
		}
		rightKeys := make([]string, len(rightRes.Msg.GetKeys()))
		for i, val := range rightRes.Msg.GetKeys() {
			rightKeys[i] = val.GetKey()
		}

		result, err := client.Link(
			cmd.Context(),
			&connect.Request[linkerv1.LinkRequest]{
				Msg: &linkerv1.LinkRequest{
					Src: &linkerv1.Set{
						Name: left,
						Keys: leftKeys,
					},
					Dst: &linkerv1.Set{
						Name: right,
						Keys: rightKeys,
					},
				},
			},
		)
		if err != nil {
			log.Fatal(err)
		}

		t := utils.NewTable()
		t.AppendHeader(table.Row{left, right})

		for src, dst := range result.Msg.GetSrcToDst() {
			t.AppendRow(table.Row{src, dst})
		}

		t.Render()
	},
}
