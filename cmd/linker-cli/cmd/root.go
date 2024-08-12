package cmd

import (
	"fmt"
	"net/http"
	"os"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
)

var BaseUrl string
var AccessToken string

var client linkerv1connect.LinkerServiceClient

var rootCmd = &cobra.Command{
	Use:   "linker-cli",
	Short: "linker-cli is a CLI interface for the VC Assist data linking service.",
}

func Execute() {
	client = linkerv1connect.NewLinkerServiceClient(http.DefaultClient, BaseUrl)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func authRequest[T any](req *connect.Request[T]) *connect.Request[T] {
	if AccessToken == "" {
		return req
	}
	req.Header().Set("Authorization", "Bearer "+AccessToken)
	return req
}
