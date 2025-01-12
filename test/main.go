package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"vcassist-backend/internal/components/telemetry"
)

var tel = telemetry.NewSlogAPI(slog.LevelDebug)
var log = tel.Logger()

type cli struct {
}

var tests = map[string]func(context.Context){
	"test-integration-moodle": func(ctx context.Context) {
		runTests("IntegrationTestMoodle", IntegrationTestMoodle)
	},
	"fuzz-snapshot": func(ctx context.Context) {
		StartFuzzTest(ctx, NewSnapshotTarget, tel)
	},
}

func main() {
	flag.Parse()
	req := flag.Arg(0)

	callback, ok := tests[req]
	if !ok {
		fmt.Printf("\"%s\" is not a known test, the known tests are as follows:\n\n", req)
		for key := range tests {
			fmt.Printf("\t%s\n", key)
		}
		fmt.Println()

		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	exit := make(chan os.Signal, 1)
	go func() {
		<-exit
		cancel()
	}()
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	callback(ctx)
}
