package snapshot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"vcassist-backend/internal/components/assert"
	"vcassist-backend/internal/components/chrono"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"
)

const (
	report_db_query = "db.query"
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

	paramLatest := db.GetMostRecentSnapshotSeriesParams{
		PowerschoolAccountID: accountId,
		CourseID:             courseId,
	}
	latest, err := tx.GetMostRecentSnapshotSeries(ctx, paramLatest)
	notFound := err == sql.ErrNoRows
	if err != nil && !notFound {
		s.tel.ReportBroken(report_db_query, err, "GetMostRecentSnapshotSeries", paramLatest)
		return err
	}

	latestSnapCount, err := tx.GetSnapshotSeriesCount(ctx, latest.ID)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetSnapshotSeriesCount", latest.ID)
		return err
	}
	latestDate := latest.StartTime.AddDate(0, 0, int(latestSnapCount))
	timeSinceLatest := now.Sub(latestDate)

	if timeSinceLatest.Seconds() < 0 {
		s.tel.ReportDebug("skipped negative insert", now.Format(time.DateTime), accountId, courseId)
		return fmt.Errorf("current date is before the most recent snapshot")
	}

	targetSeriesId := latest.ID
	if timeSinceLatest >= time.Hour*24 || notFound {
		param := db.AddSnapshotSeriesParams{
			PowerschoolAccountID: accountId,
			CourseID:             courseId,
			StartTime:            startOfToday,
		}

		targetSeriesId, err = tx.AddSnapshotSeries(ctx, param)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.tel.ReportBroken(report_db_query, err, "AddSnapshotSeries", param)
			return err
		}
	}

	// s.tel.ReportDebug("make snapshot", accountId, courseId, targetSeriesId, float64(value))
	paramCreateSnap := db.CreateSnapshotParams{
		SeriesID: targetSeriesId,
		Value:    float64(value),
	}
	err = tx.CreateSnapshot(ctx, paramCreateSnap)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "CreateSnapshot", paramCreateSnap)
		return err
	}

	commit()
	return nil
}
