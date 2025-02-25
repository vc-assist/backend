package fuzzing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"vcassist-backend/internal/telemetry"

	"github.com/hujun-open/myflags/v2"
)

// Target is represents an class/object of some sort, it contains state
// and contains methods for mutating its state.
//
// Any possible mutation to the state of the target is called a "step",
// fuzzing works by deterministically randomly choosing steps (and their inputs)
// to perform, then examining the state to determine if any invariants are violated.
//
// As such, all possible steps should be exposed to the fuzzer as methods satisfying
// the signature:
//
// `Step*(ctx context.Context, res *Results) error`
//
// Invariants can be verified within a step itself, and if one is violated, the step should return an error describing the violation.
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
//
// If a method matching the signature:
//
// `OnEnd(ctx context.Context, res *Results)`
//
// is present, it will be called at the end of the fuzz path.
type Target interface{}

type step = func(ctx context.Context, res *Results) error

type onEnd = func(ctx context.Context, res *Results)

func getTargetMethods(target Target) (steps []reflect.Method, onEnd reflect.Method) {
	t := reflect.TypeOf(target)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		methodType := method.Type

		if methodType.NumIn() != 3 {
			continue
		}
		hasCtx := methodType.In(1) == reflect.TypeOf((*context.Context)(nil)).Elem()
		hasRes := methodType.In(2) == reflect.TypeOf(&Results{})

		if method.Name == "OnEnd" {
			onEnd = method
			continue
		}

		if methodType.NumOut() != 1 {
			continue
		}
		hasReturn := methodType.Out(0) == reflect.TypeOf((*error)(nil)).Elem()

		if !hasCtx || !hasRes || !hasReturn {
			continue
		}
		if !strings.HasPrefix(method.Name, "Step") {
			continue
		}

		steps = append(steps, method)
	}

	return steps, onEnd
}

// Results contains the state of the fuzzing.
type Results struct {
	failures []error
}

func (r *Results) Fail(err error) {
	r.failures = append(r.failures, err)
}

func (r *Results) formatFails() string {
	var out strings.Builder

	out.WriteString("====== CHECKS FAILED ======\n\n")
	for _, err := range r.failures {
		out.WriteString(fmt.Sprintf("\t- %v\n", err))
	}

	return out.String()
}

type TargetProvider interface {
	CreateTarget(tel telemetry.API, rndm *rand.Rand) (Target, error)
}

// F is a fuzzing job on a given fuzz target.
type F struct {
	tel telemetry.API

	provider TargetProvider
	steps    []reflect.Method
	onEnd    reflect.Method

	minSteps uint64
	maxSteps uint64
	usePath  bool
	path     Path
}

// New creates a new fuzzing job.
//
// note: path is an optional parameter (it can be provided the zero value),
// if it has more than 0 steps, it will skip exploring various fuzzing paths
// and only execute the one given.
func New(
	tel telemetry.API,
	provider TargetProvider,
	minSteps, maxSteps uint64,
	path Path,
) (F, error) {
	f := F{
		tel:      tel,
		provider: provider,
		minSteps: minSteps,
		maxSteps: maxSteps,
		path:     path,
		usePath:  path.Steps > 0,
	}

	f.tel = telemetry.NewScopedAPI("fuzzer", f.tel)

	target, err := provider.CreateTarget(tel, rand.New(rand.NewSource(0)))
	if err != nil {
		return F{}, err
	}
	f.steps, f.onEnd = getTargetMethods(target)

	return f, nil
}

func (f F) runStep(target Target, stepIdx int, ctx context.Context, results *Results) error {
	step := f.steps[stepIdx]
	outs := step.Func.Call([]reflect.Value{
		reflect.ValueOf(target),
		reflect.ValueOf(ctx),
		reflect.ValueOf(results),
	})
	val := outs[0].Interface()
	if val == nil {
		return nil
	}
	return val.(error)
}

func (f F) runOnEnd(target Target, ctx context.Context, results *Results) {
	if f.onEnd.Func.Interface() == nil {
		return
	}
	f.onEnd.Func.Call([]reflect.Value{
		reflect.ValueOf(target),
		reflect.ValueOf(ctx),
		reflect.ValueOf(results),
	})
}

// runPath runs a specific Path (that is, a seed and a step count) on a given fuzz target.
func (f F) runPath(ctx context.Context, target Target, rndm *rand.Rand, stepCount int) (*Results, error) {
	results := &Results{}

	for i := 0; i < stepCount; i++ {
		stepIdx := rndm.Intn(len(f.steps))
		f.runStep(target, stepIdx, ctx, results)
	}
	f.runOnEnd(target, ctx, results)

	return results, nil
}

// fuzzWorker does the job of exploring the state space of a given fuzz target.
func (f F) fuzzWorker(ctx context.Context, cancel func(), count *uint64) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		seed := rand.Int63()
		rndm := rand.New(rand.NewSource(seed))

		target, err := f.provider.CreateTarget(telemetry.NoopAPI{}, rndm)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			f.tel.ReportBroken("failed to setup fuzz target", "err", err)
			cancel()
			return
		}

		stepCount := int(f.minSteps) + rndm.Intn(int(f.maxSteps-f.minSteps))

		results, err := f.runPath(ctx, target, rndm, stepCount)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			f.tel.ReportBroken("encountered fatal error", "err", err)
			cancel()
			return
		}

		atomic.AddUint64(count, 1)

		if len(results.failures) == 0 {
			continue
		}

		f.tel.ReportBroken(fmt.Sprintf(
			"%s\npath: %d:%d\n",
			results.formatFails(),
			seed,
			stepCount,
		))
		cancel()
		return
	}
}

// StartFuzzTest does not block, but spawns various goroutines to explore the
// given fuzz target's state space.
func (f F) StartFuzzTest(ctx context.Context) {
	if f.usePath {
		f.tel.ReportDebug("running single fuzz target", "seed", f.path.Seed)

		rndm := rand.New(rand.NewSource(f.path.Seed))
		target, err := f.provider.CreateTarget(f.tel, rndm)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			f.tel.ReportBroken("failed to setup fuzz target", "err", err)
			return
		}

		results, err := f.runPath(ctx, target, rndm, int(f.path.Steps))
		if err != nil {
			f.tel.ReportBroken("encountered fatal error", "err", err)
			return
		}
		if len(results.failures) == 0 {
			f.tel.ReportDebug("no failures")
			return
		}

		f.tel.ReportBroken(results.formatFails())
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cpus := runtime.NumCPU()
	f.tel.ReportDebug("starting fuzzing on all threads", "count", cpus)

	var count uint64
	countPtr := &count

	for range cpus {
		go f.fuzzWorker(ctx, cancel, countPtr)
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

		f.tel.ReportDebug(
			"run paths count",
			"count", atomic.LoadUint64(countPtr),
		)
	}
}

// Path represents a series of steps (and the number of steps to take) to execute
// on a given fuzz target.
type Path struct {
	Seed  int64
	Steps int64
}

func (Path) ToStr(in any, tag reflect.StructTag) string {
	path := in.(Path)
	return fmt.Sprintf("%d:%d", path.Seed, path.Steps)
}

func (Path) FromStr(text string, tag reflect.StructTag) (any, error) {
	segments := strings.Split(text, ":")
	if len(segments) != 2 {
		return nil, fmt.Errorf("parse FuzzPath: more than one separator '%s' present", text)
	}

	seed, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse FuzzPath: %w", err)
	}
	steps, err := strconv.ParseInt(segments[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse FuzzPath: %w", err)
	}

	return Path{
		Seed:  seed,
		Steps: steps,
	}, nil
}

func init() {
	myflags.Register[Path](Path{})
}
