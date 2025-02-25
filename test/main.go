package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"vcassist-backend/internal/telemetry"
	"vcassist-backend/test/fuzzing"
	"vcassist-backend/test/integration"

	"github.com/hujun-open/cobra"
	"github.com/hujun-open/myflags/v2"
)

var tel = telemetry.NewSlogAPI(slog.LevelDebug)

type fuzzCli struct {
	Path     fuzzing.Path `short:"p" usage:"replay a fuzzer with a given fuzzing path"`
	MinSteps uint64       `usage:"the minimum amount of steps that must be executed on any given fuzz target"`
	MaxSteps uint64       `usage:"the maximum amount of steps that can be executed on any given fuzz target"`

	Snapshot struct{} `action:"FuzzSnapshot"`
}

type integrationCli struct {
	Moodle struct{} `action:"TestIntegrateMoodle"`
}

type cli struct {
	Integration integrationCli `usage:"run integration tests for ..."`
	Fuzz        fuzzCli        `usage:"run fuzzer ..."`
}

func (c cli) runTests(name string, body func(t *testing.T)) {
	testing.Main(
		func(pat, str string) (bool, error) {
			return true, nil
		},
		[]testing.InternalTest{
			{name, body},
		},
		nil,
		nil,
	)
}

func (c cli) TestIntegrateMoodle(cmd *cobra.Command, args []string) {
	c.runTests("IntegrationTestMoodle", integration.IntegrationTestMoodle)
}

func (c cli) runFuzzing(ctx context.Context, provider fuzzing.TargetProvider) {
	f, err := fuzzing.New(tel, provider, c.Fuzz.MinSteps, c.Fuzz.MaxSteps, c.Fuzz.Path)
	if err != nil {
		tel.ReportBroken("fuzzing.New: %w", err)
	}
	f.StartFuzzTest(ctx)
}

func (c cli) FuzzSnapshot(cmd *cobra.Command, args []string) {
	c.runFuzzing(cmd.Context(), fuzzing.SnapshotProvider{})
}

func main() {
	input := cli{
		Fuzz: fuzzCli{
			MinSteps: 10,
			MaxSteps: 100,
		},
	}

	filler := myflags.NewFiller(
		"test",
		"the vcassist backend test runner",
		myflags.WithSummaryHelp(),
	)

	err := filler.Fill(&input)
	if err != nil {
		tel.ReportBroken("cli arguments error", "err", err)
		os.Exit(22)
	}

	ctx, cancel := context.WithCancel(context.Background())
	exit := make(chan os.Signal, 1)
	go func() {
		<-exit
		cancel()
	}()
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	err = filler.ExecuteContext(ctx)
	if err != nil {
		tel.ReportBroken("exec err", "err", err)
		os.Exit(1)
	}
}
