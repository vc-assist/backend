package edit

import (
	"fmt"
	"log"
	"os"
	"sort"
	"vcassist-backend/cmd/linker-cli/globals"
	"vcassist-backend/cmd/linker-cli/utils"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	suggestLinksCmd.Flags().Bool("write", false, "Write the suggestions to an edit file by the name of 'edit_suggestions.txt'")
	suggestLinksCmd.Flags().Float32("threshold", 0.75, "The threshold to filter suggestions by.")

	RootCmd.AddCommand(suggestLinksCmd)
}

var suggestLinksCmd = &cobra.Command{
	Use:   "suggest <set 1> <set 2> [--write]",
	Short: "Suggest links between the two given sets.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		write, err := cmd.Flags().GetBool("write")
		if err != nil {
			log.Fatal(err)
		}
		threshold, err := cmd.Flags().GetFloat32("threshold")
		if err != nil {
			log.Fatal(err)
		}

		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		set1 := args[0]
		set2 := args[1]

		res, err := client.SuggestLinks(cmd.Context(), &connect.Request[linkerv1.SuggestLinksRequest]{
			Msg: &linkerv1.SuggestLinksRequest{
				SetLeft:   set1,
				SetRight:  set2,
				Threshold: threshold,
			},
		})
		if err != nil {
			log.Fatal(err)
		}

		if write {
			_, err = os.Stat("edit_suggestions.txt")
			if err == nil {
				log.Fatal("A file called 'edit_suggestions.txt' already exists, rather than overwrite it, I am aborting now...")
			}

			file := actionFile{
				leftSet:  set1,
				rightSet: set2,
			}
			for _, suggest := range res.Msg.GetSuggestions() {
				file.actions = append(file.actions, actionLine{
					directive: action_add,
					keyLeft:   suggest.GetLeftKey(),
					keyRight:  suggest.GetRightKey(),
					comment:   fmt.Sprintf("correlation: %f", suggest.GetCorrelation()),
				})
			}

			f, err := os.Create("edit_suggestions.txt")
			if err != nil {
				log.Fatal(err)
			}
			_, err = f.WriteString(file.String())
			f.Close()
			if err != nil {
				log.Fatal(err)
			}
		}

		sort.Slice(res.Msg.GetSuggestions(), func(i, j int) bool {
			// sort descending
			return res.Msg.GetSuggestions()[i].GetCorrelation() > res.Msg.GetSuggestions()[j].GetCorrelation()
		})

		t := utils.NewTable()
		t.AppendHeader(table.Row{set1, set2, "Correlation"})

		for _, suggest := range res.Msg.GetSuggestions() {
			t.AppendRow(table.Row{suggest.GetLeftKey(), suggest.GetRightKey(), suggest.GetCorrelation()})
		}

		t.Render()
	},
}
