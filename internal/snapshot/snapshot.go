package snapshot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"vcassist-backend/internal/assert"
	"vcassist-backend/internal/chrono"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
)

const (
	report_db_query      = "db.query"
	report_make_snapshot = "snapshot.make-snapshot"
)

type Snapshot struct {
	db     *db.Queries
	makeTx db.MakeTx
	tel    telemetry.API
	time   chrono.TimeAPI
}

func NewSnapshot(
	db *db.Queries,
	makeTx db.MakeTx,
	time chrono.TimeAPI,
	tel telemetry.API,
) Snapshot {
	assert.NotNil(db)
	assert.NotNil(makeTx)
	assert.NotNil(tel)

	tel = telemetry.NewScopedAPI("snapshot", tel)

	return Snapshot{
		db:     db,
		makeTx: makeTx,
		time:   time,
		tel:    tel,
	}
}

func (s Snapshot) GetSnapshots(ctx context.Context, accountId int64, courseId string) (values []float32, times []time.Time, err error) {
	param := db.GetSnapshotSeriesParams{
		PowerschoolAccountID: accountId,
		CourseID:             courseId,
	}
	dbSeries, err := s.db.GetSnapshotSeries(ctx, param)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetSnapshotSeries", param)
		return nil, nil, err
	}

	for _, series := range dbSeries {
		dbSnapshots, err := s.db.GetSnapshotSeriesSnapshots(ctx, series.ID)
		if err != nil {
			s.tel.ReportBroken(report_db_query, err, "GetSnapshotSeriesSnapshots", series.ID)
			continue
		}
		// s.tel.ReportDebug("get snapshots", accountId, courseId, series.ID, dbSnapshots)
		for i, value := range dbSnapshots {
			values = append(values, float32(value))
			times = append(times, series.StartTime.Add(time.Duration(i)*24*time.Hour))
		}
	}

	return values, times, nil
}

func (s Snapshot) MakeSnapshot(ctx context.Context, accountId int64, courseId string, value float32) error {
	now := s.time.Now()
	startOfToday := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		chrono.LA(),
	)

	tx, discard, commit, err := s.makeTx()
	if err != nil {
		s.tel.ReportBroken(report_db_query, fmt.Errorf("make tx: %w", err))
		return err
	}
	defer discard()

	paramLatest := db.GetLatestSnapshotSeriesParams{
		PowerschoolAccountID: accountId,
		CourseID:             courseId,
	}
	latest, err := tx.GetLatestSnapshotSeries(ctx, paramLatest)
	if err != nil && err != sql.ErrNoRows {
		s.tel.ReportBroken(report_db_query, err, "GetLatestSnapshotSeries", paramLatest)
		return err
	}
	mostRecentNotFound := err == sql.ErrNoRows

	latestSnapCount, err := tx.GetSnapshotSeriesCount(ctx, latest.ID)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetSnapshotSeriesCount", latest.ID)
		return err
	}
	latestDate := latest.StartTime.AddDate(0, 0, int(latestSnapCount))
	timeSinceLatest := startOfToday.Sub(latestDate)

	if timeSinceLatest.Seconds() < 0 {
		err := fmt.Errorf("current date is before the most recent snapshot")
		s.tel.ReportBroken(report_make_snapshot, err, now.Format(time.DateTime), accountId, courseId, value)
		return err
	}

	targetSeriesId := latest.ID

	if timeSinceLatest.Hours() > 23 || mostRecentNotFound {
		// create new series and make it the target series

		param := db.CreateSnapshotSeriesParams{
			PowerschoolAccountID: accountId,
			CourseID:             courseId,
			StartTime:            startOfToday,
		}

		s.tel.ReportDebug(
			"create new series",
			courseId,
			accountId,
			startOfToday,
			telemetry.KV{Key: "time_since_latest", Value: timeSinceLatest.String()},
		)

		targetSeriesId, err = tx.CreateSnapshotSeries(ctx, param)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.tel.ReportBroken(report_db_query, err, "AddSnapshotSeries", param)
			return err
		}
	}

	// append to target series
	paramCreateSnap := db.CreateSnapshotParams{
		SeriesID: targetSeriesId,
		Value:    float64(value),
	}

	s.tel.ReportDebug("append to series", float64(value), targetSeriesId)
	err = tx.CreateSnapshot(ctx, paramCreateSnap)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "CreateSnapshot", paramCreateSnap)
		return err
	}

	commit()
	return nil
}
