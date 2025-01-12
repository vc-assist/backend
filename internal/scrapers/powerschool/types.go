package powerschool

import (
	"context"
	"time"
	"vcassist-backend/internal/components/assert"
	"vcassist-backend/internal/components/chrono"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"
)

// WeightsAPI describes all the methods that make up the external assignment weight data
// the powerschool api implementation depends on.
type WeightsAPI interface {
	// GetWeights returns the weight values for a course and its categories
	GetWeights(ctx context.Context, courseId string, categories []string) ([]float32, error)
}

type SnapshotValue struct {
	Value float32
	Time  time.Time
}

// SnapshotAPI describes all the methods that make up the external assignment snapshot data
// the powerschool api implementation depends on.
type SnapshotAPI interface {
	// GetSnapshots returns the snapshots stored for a given course on a given account.
	// The snapshots should be ordered earliest to latest.
	GetSnapshots(ctx context.Context, accountId int64, courseId string) ([]SnapshotValue, error)

	// MakeSnapshot adds a grade snapshot to a specific accountId & courseId for the current time.
	// Make sure that no more than 1 grade snapshot is created per day for the same account.
	MakeSnapshot(ctx context.Context, accountId int64, courseId string, value float32) error
}

// Powerschool implements service.PowerschoolAPI
type Powerschool struct {
	db       *db.Queries
	tel      telemetry.API
	time     chrono.TimeAPI
	weights  WeightsAPI
	snapshot SnapshotAPI
}

func NewPowerschool(
	db *db.Queries,
	tel telemetry.API,
	time chrono.TimeAPI,
	weights WeightsAPI,
	snapshot SnapshotAPI,
) Powerschool {
	assert.NotNil(db)
	assert.NotNil(tel)
	assert.NotNil(chrono)
	assert.NotNil(weights)
	assert.NotNil(snapshot)

	tel = telemetry.NewScopedAPI("apis", tel)

	return Powerschool{
		db:       db,
		tel:      tel,
		time:     time,
		weights:  weights,
		snapshot: snapshot,
	}
}
