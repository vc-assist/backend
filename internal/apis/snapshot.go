package apis

import (
	"context"
	"database/sql"
	"time"
	"vcassist-backend/internal/assert"
	"vcassist-backend/internal/chrono"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
)

type SnapshotImpl struct {
	db     *db.Queries
	makeTx MakeTx
	tel    telemetry.API
	chrono chrono.API
}

func NewSnapshotImpl(db *db.Queries, makeTx MakeTx, tel telemetry.API) SnapshotImpl {
	assert.NotNil(db)
	assert.NotNil(makeTx)
	assert.NotNil(tel)

	return SnapshotImpl{db: db, makeTx: makeTx, tel: tel}
}

func (s SnapshotImpl) GetSnapshots(ctx context.Context, accountId int64, courseId string) ([]Snapshot, error) {
	param := db.GetSnapshotSeriesParams{
		PowerschoolAccountID: accountId,
		CourseID:             courseId,
	}
	dbSeries, err := s.db.GetSnapshotSeries(ctx, param)
	if err != nil {
		s.tel.ReportBroken(report_db_query, err, "GetSnapshotSeries", param)
		return nil, err
	}

	var snapshots []Snapshot
	for _, series := range dbSeries {
		dbSnapshots, err := s.db.GetSnapshotSeriesSnapshots(ctx, series.ID)
		if err != nil {
			s.tel.ReportBroken(report_db_query, err, "GetSnapshotSeriesSnapshots", series.ID)
			continue
		}
		for _, value := range dbSnapshots {
			snapshots = append(snapshots, Snapshot{
				Value: float32(value),
				Time:  series.StartTime,
			})
		}
	}

	return snapshots, nil
}

func (s SnapshotImpl) MakeSnapshot(ctx context.Context, accountId int64, courseId string, value float32) error {
	now := s.chrono.Now()
	startOfToday := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		s.chrono.Location(),
	)

	tx, discard, commit := s.makeTx()
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

	if timeSinceLatest >= time.Hour*24 || notFound {
		param := db.AddSnapshotSeriesParams{
			PowerschoolAccountID: accountId,
			CourseID:             courseId,
			StartTime:            startOfToday,
		}

		_, err = tx.AddSnapshotSeries(ctx, param)
		if err != nil {
			s.tel.ReportBroken(report_db_query, err, "AddSnapshotSeries", param)
			return err
		}

		commit()
		return nil
	}

	paramCreateSnap := db.CreateSnapshotParams{
		SeriesID: latest.ID,
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
