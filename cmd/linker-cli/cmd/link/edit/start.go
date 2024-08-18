package edit

import (
	"log"
	"os"
	"vcassist-backend/cmd/linker-cli/globals"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start <set 1> <set 2> <output file path>",
	Short: "Creates an edit actions file.",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		left := args[0]
		right := args[1]
		outputFilePath := args[2]

		res, err := client.GetExplicitLinks(cmd.Context(), &connect.Request[linkerv1.GetExplicitLinksRequest]{
			Msg: &linkerv1.GetExplicitLinksRequest{
				LeftSet:  left,
				RightSet: right,
			},
		})
		if err != nil {
			log.Fatal(err)
		}

		file := actionFile{
			leftSet:  left,
			rightSet: right,
		}
		file.actions = make([]actionLine, len(res.Msg.LeftKeys))
		for i := 0; i < len(res.Msg.LeftKeys); i++ {
			file.actions[i] = actionLine{
				directive: action_keep,
				keyLeft:   res.Msg.LeftKeys[i],
				keyRight:  res.Msg.RightKeys[i],
			}
		}
		f, err := os.Create(outputFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		_, err = f.WriteString(file.String())
		if err != nil {
			log.Fatal(err)
		}
	},
}
