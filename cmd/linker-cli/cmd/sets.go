package cmd

import (
	"fmt"
	"os"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setsCmd)
}

var setsCmd = &cobra.Command{
	Use:   "sets",
	Short: "Prints the sets known to the linker.",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := client.GetKnownSets(
			cmd.Context(),
			authRequest(&connect.Request[linkerv1.GetKnownSetsRequest]{
				Msg: &linkerv1.GetKnownSetsRequest{},
			}),
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Set"})

		for _, s := range res.Msg.GetSets() {
			t.AppendRow(table.Row{s})
		}

		t.SetStyle(table.StyleRounded)
		t.Render()
	},
}
