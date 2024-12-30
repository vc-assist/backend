package impl

import (
	"vcassist-backend/internal/components/assert"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"
)

// Moodle implements service.MoodleAPI
type Moodle struct {
	db        *db.Queries
	makeTx    MakeTx
	tel       telemetry.API
	adminUser string
	adminPass string
}

func NewMoodle(
	db *db.Queries,
	makeTx MakeTx,
	tel telemetry.API,
	adminUser, adminPass string,
) Moodle {
	assert.NotNil(db)
	assert.NotNil(makeTx)
	assert.NotNil(tel)
	assert.NotEmptyStr(adminUser)
	assert.NotEmptyStr(adminPass)

	tel = telemetry.NewScopedAPI("apis", tel)

	return Moodle{db: db, makeTx: makeTx, tel: tel}
}
