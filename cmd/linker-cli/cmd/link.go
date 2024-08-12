package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(linkCmd)
}

var linkCmd = &cobra.Command{
	Use:   "link <set 1> <set 2>",
	Short: "Run linking between the known keys of two known sets.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "incorrect number of arguments")
			os.Exit(1)
		}

		leftRes, err := client.GetKnownKeys(
			cmd.Context(),
			authRequest(&connect.Request[linkerv1.GetKnownKeysRequest]{
				Msg: &linkerv1.GetKnownKeysRequest{
					Set: args[0],
				},
			}),
		)
		if err != nil {
			log.Fatal(err)
		}
		rightRes, err := client.GetKnownKeys(
			cmd.Context(),
			authRequest(&connect.Request[linkerv1.GetKnownKeysRequest]{
				Msg: &linkerv1.GetKnownKeysRequest{
					Set: args[1],
				},
			}),
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
			context.Background(),
			authRequest(&connect.Request[linkerv1.LinkRequest]{
				Msg: &linkerv1.LinkRequest{
					Src: &linkerv1.Set{
						Name: args[0],
						Keys: leftKeys,
					},
					Dst: &linkerv1.Set{
						Name: args[1],
						Keys: rightKeys,
					},
					Threshold: 0,
				},
			}),
		)
		if err != nil {
			log.Fatal(err)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{args[0], args[1]})

		for src, dst := range result.Msg.GetSrcToDst() {
			t.AppendRow(table.Row{src, dst})
		}

		t.Render()
	},
}
