package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
	"vcassist-backend/internal/application/snapshot"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"
	"vcassist-backend/pkg/migrations"
)

// steps:
// - GetSnapshots(accountId, courseId)
//   - accountId/courseId can either be:
//     - an id that matches a random existing row (80%)
//     - an id that doesn't exist (20%)
// - MakeSnapshot(accountId, courseId, value)
//   - accountId/courseId can either be:
//     - an id that matches an existing row (80%)
//     - an id that doesn't exist (20%)
//   - value can be any float value in [0, 1]
// - add ps_account with a given accountId
// - fault inject: time passing by faster than usual / reversing
// - fault inject: failed db query (TODO)

// properties of the system:
// - making a snapshot should take no more than 10ms (on the p95, so no more than 5%
//     of MakeSnapshot should be more than 10ms)
// - getting a snapshot should take no more than 10ms (on the p95, so no more than 5%
//     of GetSnapshot should be more than 10ms)
// - a snapshot added to an account should be added to that account and no others

// randomly generate steps to try and find a situation where a property of the system is invalidated

type timeShim struct {
	current   time.Time
	increment func(rndm *rand.Rand) int
	rndm      *rand.Rand
}

func newTimeShim(rndm *rand.Rand) *timeShim {
	return &timeShim{
		current: time.Now(),
		// 0: add anywhere from (0, 60) seconds
		// 1: add anywhere from (0, 60) minutes
		// 2: add anywhere from (0, 24) hours
		// 3: add anywhere from (0, 30) days
		increment: RandomSwitch(3, 3, 3, 1),
		rndm:      rndm,
	}
}

func (s timeShim) randDuration() time.Duration {
	var dur time.Duration
	switch s.increment(s.rndm) {
	case 0:
		dur = time.Duration(s.rndm.Intn(59)+1) * time.Second
	case 1:
		dur = time.Duration(s.rndm.Intn(59)+1) * time.Minute
	case 2:
		dur = time.Duration(s.rndm.Intn(23)+1) * time.Hour
	case 3:
		dur = time.Duration(s.rndm.Intn(29)+1) * time.Hour * 24
	}
	return dur
}

func (s timeShim) Now() time.Time {
	return s.current
}

func NewSnapshotTarget(rndm *rand.Rand, tel telemetry.API) (target FuzzTarget, err error) {
	dbtx, err := migrations.OpenDB(":memory:")
	if err != nil {
		return
	}
	_, err = dbtx.Exec(db.Schema)
	if err != nil {
		return
	}

	timeshim := newTimeShim(rndm)

	qry := db.New(dbtx)
	maketx := db.NewMakeTx(dbtx)
	snapshotter := snapshot.NewSnapshot(qry, maketx, timeshim, tel)

	// 0: id matches random existing row
	// 1: new id
	idAction := RandomSwitch(4, 1)

	randomAccountId := func(ctx context.Context, rndm *rand.Rand) (int64, error) {
		switch idAction(rndm) {
		case 0:
			accounts, err := qry.GetAllPSAccounts(ctx)
			if err != nil {
				return 0, err
			}

			if len(accounts) == 0 {
				id, err := qry.AddPSAccount(ctx, db.AddPSAccountParams{
					Email: "fake@email.com",
				})
				return id, err
			}

			i := rndm.Intn(len(accounts))
			return accounts[i].ID, nil
		default:
			return rndm.Int63(), nil
		}
	}

	randomCourseId := func(ctx context.Context, rndm *rand.Rand, accountId int64) (string, error) {
		switch idAction(rndm) {
		case 0:
			ids, err := qry.GetSnapshotSeriesCourseIds(ctx, accountId)
			if err != nil {
				return "", err
			}

			if len(ids) == 0 {
				return RandomString(rndm, 20), nil
			}

			i := rndm.Intn(len(ids))
			return ids[i], nil
		default:
			return RandomString(rndm, 20), nil
		}
	}

	added := map[string][]float32{}

	getSnapshotsTimeExceeded := 0
	getSnapshotsCount := 0
	makeSnapshotTimeExceeded := 0
	makeSnapshotCount := 0

	// clear && go run . fuzz-snapshot 5675861429899494614

	return FuzzTarget{
		OnEnd: func(ctx context.Context, f *FuzzResults) {
			if float64(getSnapshotsTimeExceeded)/float64(getSnapshotsCount) > 0.05 {
				f.Fail("getsnapshots.time", fmt.Errorf(
					"more than 5%% of GetSnapshots took too longer than 10ms",
				))
			}

			if float64(makeSnapshotTimeExceeded)/float64(makeSnapshotCount) > 0.05 {
				f.Fail("makesnapshot.time", fmt.Errorf(
					"more than 5%% of MakeSnapshot took too longer than 10ms",
				))
			}
		},
		Steps: []FuzzStep{
			func(ctx context.Context, f *FuzzResults) (string, error) {
				accountId, err := randomAccountId(ctx, rndm)
				if err != nil {
					return "", err
				}
				courseId, err := randomCourseId(ctx, rndm, accountId)
				if err != nil {
					return "", err
				}

				getSnapshotsCount++

				t1 := time.Now()
				values, _, err := snapshotter.GetSnapshots(ctx, accountId, courseId)
				if err == nil {
					expected := added[fmt.Sprintf("%d:%s", accountId, courseId)]
					for i := range values {
						if values[i] != expected[i] {
							f.Fail("getsnapshots.correct-values", fmt.Errorf(
								"return values of GetSnapshots does not match up with the expected values. got: %v expected: %v",
								values,
								expected,
							))
							break
						}
					}
				}
				t2 := time.Now()

				duration := t2.Sub(t1)
				if duration.Milliseconds() > 10 {
					getSnapshotsTimeExceeded++
				}

				return fmt.Sprintf("GetSnapshots(%d, %s)", accountId, courseId), nil
			},
			func(ctx context.Context, f *FuzzResults) (string, error) {
				value := rndm.Float64()
				// round to the nearest whole percent
				value = math.Round(value*100) / 100

				accountId, err := randomAccountId(ctx, rndm)
				if err != nil {
					return "", err
				}
				courseId, err := randomCourseId(ctx, rndm, accountId)
				if err != nil {
					return "", err
				}

				makeSnapshotCount++

				t1 := time.Now()
				err = snapshotter.MakeSnapshot(ctx, accountId, courseId, float32(value))
				if err == nil {
					id := fmt.Sprintf("%d:%s", accountId, courseId)
					added[id] = append(added[id], float32(value))
				}
				t2 := time.Now()

				duration := t2.Sub(t1)
				if duration.Milliseconds() > 10 {
					makeSnapshotTimeExceeded++
				}

				return fmt.Sprintf(
					"MakeSnapshot(%d, %s, %f)",
					accountId,
					courseId,
					value,
				), nil
			},
			func(ctx context.Context, f *FuzzResults) (string, error) {
				accountId, err := qry.AddPSAccount(ctx, db.AddPSAccountParams{
					Email: "fake@email.com",
				})
				return fmt.Sprintf("AddPSAccount(%d)", accountId), err
			},
			func(ctx context.Context, f *FuzzResults) (string, error) {
				duration := timeshim.randDuration()
				timeshim.current = timeshim.current.Add(duration)
				return fmt.Sprintf(
					"IncrementTime(%s): %s",
					duration.String(),
					timeshim.current.Format(time.DateTime),
				), nil
			},
			func(ctx context.Context, f *FuzzResults) (string, error) {
				duration := timeshim.randDuration()
				timeshim.current = timeshim.current.Add(-duration)
				return fmt.Sprintf(
					"DecrementTime(%s): %s",
					duration.String(),
					timeshim.current.Format(time.DateTime),
				), nil
			},
		},
	}, nil
}
