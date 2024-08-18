package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"vcassist-backend/cmd/linker-cli/cmd"
	"vcassist-backend/cmd/linker-cli/globals"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"

	"connectrpc.com/connect"
)

func authInterceptor(accessToken string) connect.UnaryInterceptorFunc {
	authHeader := fmt.Sprintf("Bearer %s", accessToken)
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", authHeader)
			return next(ctx, req)
		}
	}
}

func main() {
	accessToken, _ := os.LookupEnv("LINKER_ACCESS_TOKEN")
	baseUrl, ok := os.LookupEnv("LINKER_BASE_URL")
	if !ok {
		fmt.Println("You should specify the base url of the linker service in the environment variable LINKER_BASE_URL.")
		os.Exit(1)
	}

	client := linkerv1connect.NewLinkerServiceClient(
		http.DefaultClient,
		baseUrl,
		connect.WithInterceptors(authInterceptor(accessToken)),
	)

	cmd.ExecuteContext(globals.Set(
		context.Background(),
		&globals.Value{Client: client},
	))
}
