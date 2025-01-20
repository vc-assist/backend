package fuzzing

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
	testutil "vcassist-backend/test/util"
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

type snapshotTarget struct {
	tel         telemetry.API
	rndm        *rand.Rand
	qry         *db.Queries
	maketx      db.MakeTx
	snapshotter snapshot.Snapshot
	timeshim    *timeShim

	added                    map[string][]float32
	getSnapshotsTimeExceeded int
	getSnapshotsCount        int
	makeSnapshotTimeExceeded int
	makeSnapshotCount        int

	idAction func(*rand.Rand) int
}

type SnapshotProvider struct{}

func (SnapshotProvider) CreateTarget(tel telemetry.API, rndm *rand.Rand) (target Target, err error) {
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

	return &snapshotTarget{
		tel:         tel,
		rndm:        rndm,
		qry:         qry,
		maketx:      maketx,
		snapshotter: snapshotter,
		timeshim:    timeshim,

		added: map[string][]float32{},
		// 0: id matches random existing row
		// 1: new id
		idAction: testutil.RandomSwitch(4, 1),
	}, nil
}

func (t *snapshotTarget) StepGetSnapshots(ctx context.Context, res *Results) error {
	accountId, err := t.randomAccountId(ctx)
	if err != nil {
		return err
	}
	courseId, err := t.randomCourseId(ctx, accountId)
	if err != nil {
		return err
	}

	t.tel.ReportDebug("? snapshots", accountId, courseId)

	t.getSnapshotsCount++

	t1 := time.Now()
	values, _, err := t.snapshotter.GetSnapshots(ctx, accountId, courseId)
	if err == nil {
		expected := t.added[fmt.Sprintf("%d:%s", accountId, courseId)]
		for i := range values {
			if values[i] != expected[i] {
				res.Fail(fmt.Errorf(
					"getsnapshots.correct-values: return values of GetSnapshots does not match up with the expected values. got: %v expected: %v",
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
		t.getSnapshotsTimeExceeded++
	}
	return nil
}

func (t *snapshotTarget) StepMakeSnapshot(ctx context.Context, f *Results) error {
	value := t.rndm.Float64()
	// round to the nearest whole percent
	value = math.Round(value*100) / 100

	accountId, err := t.randomAccountId(ctx)
	if err != nil {
		return err
	}
	courseId, err := t.randomCourseId(ctx, accountId)
	if err != nil {
		return err
	}

	t.tel.ReportDebug("+ snapshot", accountId, courseId, value)

	t.makeSnapshotCount++

	t1 := time.Now()
	err = t.snapshotter.MakeSnapshot(ctx, accountId, courseId, float32(value))
	if err == nil {
		id := fmt.Sprintf("%d:%s", accountId, courseId)
		t.added[id] = append(t.added[id], float32(value))
	}
	t2 := time.Now()

	duration := t2.Sub(t1)
	if duration.Milliseconds() > 10 {
		t.makeSnapshotTimeExceeded++
	}

	return nil
}

func (t *snapshotTarget) StepAddPSAccount(ctx context.Context, f *Results) error {
	accountId, err := t.qry.AddPSAccount(ctx, db.AddPSAccountParams{
		Email: "fake@email.com",
	})
	if err != nil {
		return err
	}
	t.tel.ReportDebug("+ powerschool account", accountId)
	return nil
}

func (t *snapshotTarget) StepAddTime(ctx context.Context, f *Results) error {
	duration := t.timeshim.randDuration()
	t.timeshim.current = t.timeshim.current.Add(duration)
	t.tel.ReportDebug(
		"+ time",
		telemetry.KV{
			Key:   "current_time",
			Value: t.timeshim.current.Format(time.DateTime),
		},
		telemetry.KV{
			Key:   "added",
			Value: duration.String(),
		},
	)
	return nil
}

func (t *snapshotTarget) StepSubTime(ctx context.Context, f *Results) error {
	duration := t.timeshim.randDuration()
	t.timeshim.current = t.timeshim.current.Add(-duration)
	t.tel.ReportDebug(
		"- time",
		telemetry.KV{
			Key:   "current_time",
			Value: t.timeshim.current.Format(time.DateTime),
		},
		telemetry.KV{
			Key:   "subtracted",
			Value: duration.String(),
		},
	)
	return nil
}

func (t *snapshotTarget) OnEnd(ctx context.Context, f *Results) {
	if float64(t.getSnapshotsTimeExceeded)/float64(t.getSnapshotsCount) > 0.05 {
		f.Fail(fmt.Errorf(
			"getsnapshots.time: more than 5%% of GetSnapshots took too longer than 10ms",
		))
	}

	if float64(t.makeSnapshotTimeExceeded)/float64(t.makeSnapshotCount) > 0.05 {
		f.Fail(fmt.Errorf(
			"makesnapshot.time: more than 5%% of MakeSnapshot took too longer than 10ms",
		))
	}
}

func (t *snapshotTarget) randomAccountId(ctx context.Context) (int64, error) {
	switch t.idAction(t.rndm) {
	case 0:
		accounts, err := t.qry.GetAllPSAccounts(ctx)
		if err != nil {
			return 0, err
		}

		if len(accounts) == 0 {
			id, err := t.qry.AddPSAccount(ctx, db.AddPSAccountParams{
				Email: "fake@email.com",
			})
			return id, err
		}

		i := t.rndm.Intn(len(accounts))
		return accounts[i].ID, nil
	default:
		return t.rndm.Int63(), nil
	}
}

func (t *snapshotTarget) randomCourseId(ctx context.Context, accountId int64) (string, error) {
	switch t.idAction(t.rndm) {
	case 0:
		ids, err := t.qry.GetSnapshotSeriesCourseIds(ctx, accountId)
		if err != nil {
			return "", err
		}

		if len(ids) == 0 {
			return testutil.RandomString(t.rndm, 10), nil
		}

		i := t.rndm.Intn(len(ids))
		return ids[i], nil
	default:
		return testutil.RandomString(t.rndm, 10), nil
	}
}
