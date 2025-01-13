package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"vcassist-backend/internal/components/telemetry"

	"github.com/hujun-open/cobra"
	"github.com/hujun-open/myflags/v2"
)

var tel = telemetry.NewSlogAPI(slog.LevelDebug)
var log = tel.Logger()

type fuzzCli struct {
	Seed     int64  `short:"s" usage:"replay a fuzzer with a given seed"`
	MinSteps uint64 `usage:"the fuzzer will generate steps within the interval [min, max)"`
	MaxSteps uint64 `usage:"the fuzzer will generate steps within the interval [min, max)"`

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
	c.runTests("IntegrationTestMoodle", IntegrationTestMoodle)
}

func (c cli) runFuzzing(ctx context.Context, mkTarget MkFuzzTarget) {
	StartFuzzTest(ctx, tel, mkTarget, c.Fuzz.Seed, c.Fuzz.MinSteps, c.Fuzz.MaxSteps)
}

func (c cli) FuzzSnapshot(cmd *cobra.Command, args []string) {
	c.runFuzzing(cmd.Context(), MakeSnapshotFuzzTarget)
}

func main() {
	input := cli{
		Fuzz: fuzzCli{
			Seed:     -1,
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
		log.Error("cli arguments error", "err", err)
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
		log.Error("exec err", "err", err)
		os.Exit(1)
	}
}
