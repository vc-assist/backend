package apis

import (
	"context"
	"time"
	"vcassist-backend/internal/assert"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
)

// WeightsAPI describes all the methods that make up the external assignment weight data
// the powerschool api implementation depends on.
type WeightsAPI interface {
	// GetWeights returns the weight values for a course and its categories
	GetWeights(ctx context.Context, courseId string, categories []string) ([]float32, error)
}

type Snapshot struct {
	Value float32
	Time  time.Time
}

// SnapshotAPI describes all the methods that make up the external assignment snapshot data
// the powerschool api implementation depends on.
type SnapshotAPI interface {
	// GetSnapshots returns the snapshots stored for a given course on a given account.
	// The snapshots should be ordered earliest to latest.
	GetSnapshots(ctx context.Context, accountId int64, courseId string) ([]Snapshot, error)

	// MakeSnapshot adds a grade snapshot to a specific accountId & courseId for the current time.
	// Make sure that no more than 1 grade snapshot is created per day for the same account.
	MakeSnapshot(ctx context.Context, accountId int64, courseId string, value float32) error
}

// PowerschoolImpl implements service.PowerschoolAPI
type PowerschoolImpl struct {
	db       *db.Queries
	tel      telemetry.API
	weights  WeightsAPI
	snapshot SnapshotAPI
}

func NewPowerschoolImpl(
	db *db.Queries,
	tel telemetry.API,
	weights WeightsAPI,
	snapshot SnapshotAPI,
) PowerschoolImpl {
	assert.NotNil(db)
	assert.NotNil(tel)
	assert.NotNil(weights)
	assert.NotNil(snapshot)

	tel = telemetry.NewScopedAPI("apis", tel)

	return PowerschoolImpl{
		db:       db,
		tel:      tel,
		weights:  weights,
		snapshot: snapshot,
	}
}
