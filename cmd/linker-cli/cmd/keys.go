package cmd

import (
	"fmt"
	"log"
	"os"
	"time"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(keysCmd)
}

type setKey struct {
	setName string
	keys    []*linkerv1.KnownKey
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Prints the keys known to the sets given as positional arguments.",
	Run: func(cmd *cobra.Command, args []string) {
		setKeys := []setKey{}
		for _, setName := range args {
			res, err := client.GetKnownKeys(cmd.Context(), &connect.Request[linkerv1.GetKnownKeysRequest]{
				Msg: &linkerv1.GetKnownKeysRequest{
					Set: setName,
				},
			})
			if err != nil {
				log.Fatal(err)
			}
			setKeys = append(setKeys, setKey{
				setName: setName,
				keys:    res.Msg.Keys,
			})
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		header := table.Row{}
		for _, set := range setKeys {
			header = append(header, fmt.Sprintf("Key: %s", set.setName))
			header = append(header, fmt.Sprintf("Last seen: %s", set.setName))
		}
		t.AppendHeader(header)

		maxKeyCount := 0
		for _, set := range setKeys {
			if len(set.keys) > maxKeyCount {
				maxKeyCount = len(set.keys)
			}
		}

		rowLength := len(setKeys) * 2
		rows := make([]table.Row, maxKeyCount)
		for i := 0; i < len(rows); i++ {
			rows[i] = make(table.Row, rowLength)
		}

		for setIdx, set := range setKeys {
			rowOffset := setIdx * 2
			for keyIdx, key := range set.keys {
				lastSeen := time.Unix(key.LastSeen, 0).Format(time.ANSIC)

				rows[keyIdx][rowOffset] = key.Key
				rows[keyIdx][rowOffset+1] = lastSeen
			}
		}

		t.AppendRows(rows)
		t.SetStyle(table.StyleRounded)
		t.Render()
	},
}
