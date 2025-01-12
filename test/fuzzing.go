package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"vcassist-backend/internal/components/telemetry"
)

type failure struct {
	name string
	err  error
}

// FuzzStep returns an error if a fatal error is incurred. It must also return a string that
// contains the name of the step and the inputs generated for that step. ex. GetSnapshots(0, some_course_id)
//
// Such errors include:
//   - Logic errors.
//   - Test setup errors. (if you have to generate a random number or something and that fails)
//
// Errors should not include:
//   - Dependency errors. (a db failing to read)
//   - Inadequate latency.
//
// Dependency errors are outside the scope of the test and will often be triggered as part of
// fault injection.
//
// Overall latency evaluation should be addressed by checks.
type FuzzStep func(ctx context.Context, f *FuzzResults) (str string, err error)

type FuzzTarget struct {
	Steps []FuzzStep
	OnEnd func(ctx context.Context, f *FuzzResults)
}

type NewFuzzTarget = func(rndm *rand.Rand, tel telemetry.API) (FuzzTarget, error)

type FuzzResults struct {
	failures []failure
	steps    []string
}

func (r *FuzzResults) Fail(name string, err error) {
	r.failures = append(r.failures, failure{
		name: name,
		err:  err,
	})
}

func (r *FuzzResults) formatFailures() string {
	var out strings.Builder

	out.WriteString("====== CHECKS FAILED ======\n\n")
	for _, fail := range r.failures {
		out.WriteString(fmt.Sprintf("\t- %s: %v\n", fail.name, fail.err))
	}

	return out.String()
}

func (r *FuzzResults) formatSteps() string {
	var out strings.Builder
	for i, step := range r.steps {
		out.WriteString(fmt.Sprintf("%d. %s\t", i+1, step))
	}
	return out.String()
}

func RunFuzzPath(ctx context.Context, target FuzzTarget, rndm *rand.Rand) (*FuzzResults, error) {
	results := &FuzzResults{}

	noSteps := rndm.Intn(100)
	for i := 0; i < noSteps; i++ {
		stepIdx := rndm.Intn(len(target.Steps))
		step := target.Steps[stepIdx]

		str, err := step(ctx, results)
		if err != nil {
			return nil, err
		}
		results.steps = append(results.steps, str)
	}

	if target.OnEnd != nil {
		target.OnEnd(ctx, results)
	}

	return results, nil
}

// StartFuzzTest does not block, but spawns various goroutines to start fuzz testing.
func StartFuzzTest(ctx context.Context, newTarget NewFuzzTarget, tel telemetry.API) {
	specificSeed := flag.Arg(1)
	if specificSeed != "" {
		seed, err := strconv.ParseInt(specificSeed, 10, 64)
		if err != nil {
			log.Error("failed to parse seed", "err", err)
			return
		}

		log.Debug("running fuzz target", "seed", seed)

		rndm := rand.New(rand.NewSource(int64(seed)))
		target, err := newTarget(rndm, tel)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			log.Error("failed to setup fuzz target", "err", err)
			return
		}

		results, err := RunFuzzPath(ctx, target, rndm)
		if err != nil {
			log.Error("encountered fatal error", "err", err)
			return
		}
		if len(results.failures) == 0 {
			log.Debug("no failures")
			return
		}

		log.Error(fmt.Sprintf(
			"%s\nsteps:\n%s",
			results.formatFailures(),
			results.formatSteps(),
		))
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cpus := runtime.NumCPU()
	log.Debug("starting fuzzing on all threads", "count", cpus)

	var count uint64
	countPtr := &count

	for range cpus {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				seed := rand.Int63()
				rndm := rand.New(rand.NewSource(seed))

				target, err := newTarget(rndm, tel)
				if errors.Is(err, context.Canceled) {
					return
				}
				if err != nil {
					log.Error("failed to setup fuzz target", "err", err)
					cancel()
					return
				}

				results, err := RunFuzzPath(ctx, target, rndm)
				if errors.Is(err, context.Canceled) {
					return
				}
				if err != nil {
					log.Error("encountered fatal error", "err", err)
					cancel()
					return
				}

				atomic.AddUint64(countPtr, 1)

				if len(results.failures) == 0 {
					continue
				}

				log.Error(fmt.Sprintf(
					"%s\nseed: %d\nsteps:\n%s",
					results.formatFailures(),
					seed,
					results.formatSteps(),
				))
				cancel()
				return
			}
		}()
	}

	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		<-timer.C

		count := atomic.LoadUint64(countPtr)
		log.Debug("run paths count", "count", count)
	}
}
