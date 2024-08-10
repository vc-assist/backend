package main

import (
	"fmt"
	"os"
	"vcassist-backend/cmd/linker-cli/cmd"
)

func main() {
	baseUrl, ok := os.LookupEnv("LINKER_BASE_URL")
	if !ok {
		fmt.Println("You should specify the base url of the linker service in the environment variable LINKER_BASE_URL.")
		os.Exit(1)
	}
	cmd.BaseUrl = baseUrl

	cmd.Execute()
}
