package edit

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"vcassist-backend/cmd/linker-cli/globals"
	linkerv1 "vcassist-backend/proto/vcassist/services/linker/v1"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply <file path>",
	Short: "Applies the actions in an edit action file.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := globals.Get(cmd.Context())
		client := ctx.Client

		path := args[0]
		contents, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}

		actionFile, err := newActionFile(bytes.NewBuffer(contents))
		if err != nil {
			log.Fatal(err)
		}

		wg := sync.WaitGroup{}
		for i, line := range actionFile.actions {
			if line.directive == action_keep || line.directive == action_sets {
				continue
			}

			fmt.Printf("%s (#%d)\n", line.String(), i+1)

			wg.Add(1)
			go func(line actionLine, i int) {
				defer func() {
					fmt.Printf("complete action (#%d)\n", i+1)
				}()
				defer wg.Done()

				left := &linkerv1.ExplicitKey{
					Set: actionFile.leftSet,
					Key: line.keyLeft,
				}
				right := &linkerv1.ExplicitKey{
					Set: actionFile.rightSet,
					Key: line.keyRight,
				}

				switch line.directive {
				case action_add:
					_, err := client.AddExplicitLink(cmd.Context(), &connect.Request[linkerv1.AddExplicitLinkRequest]{
						Msg: &linkerv1.AddExplicitLinkRequest{
							Left:  left,
							Right: right,
						},
					})
					if err != nil {
						fmt.Printf(
							"[ERROR] failed to create explicit link (%s - %s):\n%v\n",
							line.keyLeft, line.keyRight, err.Error(),
						)
					}
				case action_delete:
					_, err := client.DeleteExplicitLink(cmd.Context(), &connect.Request[linkerv1.DeleteExplicitLinkRequest]{
						Msg: &linkerv1.DeleteExplicitLinkRequest{
							Left:  left,
							Right: right,
						},
					})
					if err != nil {
						fmt.Printf(
							"[ERROR] failed to delete explicit link (%s - %s):\n%v\n",
							line.keyLeft, line.keyRight, err.Error(),
						)
					}
				}
			}(line, i)
		}

		wg.Wait()
		fmt.Println("complete.")
	},
}
