package apis

import (
	"vcassist-backend/internal/assert"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
)

// MoodleImpl implements service.MoodleAPI
type MoodleImpl struct {
	db        *db.Queries
	makeTx    MakeTx
	tel       telemetry.API
	adminUser string
	adminPass string
}

func NewMoodleImpl(
	db *db.Queries,
	makeTx MakeTx,
	tel telemetry.API,
	adminUser, adminPass string,
) MoodleImpl {
	assert.NotNil(db)
	assert.NotNil(makeTx)
	assert.NotNil(tel)
	assert.NotEmptyStr(adminUser)
	assert.NotEmptyStr(adminPass)

	tel = telemetry.NewScopedAPI("apis", tel)

	return MoodleImpl{db: db, makeTx: makeTx, tel: tel}
}
