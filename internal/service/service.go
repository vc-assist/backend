package service

import (
	"crypto/rand"
	"encoding/base64"
	"vcassist-backend/internal/assert"
	"vcassist-backend/internal/chrono"
	"vcassist-backend/internal/db"
	"vcassist-backend/internal/telemetry"
)

const (
	report_db_query              = "db.query"
	report_rand_token_generation = "rand.token-generation"
)

// RandomAPI is an abstraction over any code that potentially generates random values.
// This makes mocking/simulation testing much easier.
//
// note: fault injection point
type RandomAPI interface {
	GenerateToken() (string, error)
}

type defaultRandomAPI struct{}

func (defaultRandomAPI) GenerateToken() (string, error) {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(nonce), nil
}

type coreAPIs struct {
	db     *db.Queries
	makeTx db.MakeTx
	rand   RandomAPI
	tel    telemetry.API
}

// NewCoreAPIs initializes a collection of common APIs all services need to run.
func NewCoreAPIs(
	db *db.Queries,
	makeTx db.MakeTx,
	tel telemetry.API,
	options ...CoreAPIsOption,
) coreAPIs {
	assert.NotNil(db)
	assert.NotNil(makeTx)
	assert.NotNil(makeTx)

	cfg := coreAPIsConfig{}
	for _, opt := range options {
		opt(&cfg)
	}

	apis := coreAPIs{
		db:     db,
		makeTx: makeTx,
		rand:   defaultRandomAPI{},
		tel:    telemetry.NewScopedAPI("service", tel),
	}
	if cfg.rand != nil {
		apis.rand = cfg.rand
	}

	return apis
}

type coreAPIsConfig struct {
	rand RandomAPI
}

type CoreAPIsOption func(cfg *coreAPIsConfig)

func WithCustomRandomAPI(rand RandomAPI) CoreAPIsOption {
	return func(cfg *coreAPIsConfig) {
		cfg.rand = rand
	}
}

// MoodleService implements `vcassist.moodle.v1.MoodleService`
type MoodleService struct {
	coreAPIs

	api    MoodleAPI
	ctxKey any
}

// PowerschoolService implements `vcassist.powerschool.v1.PowerschoolService`
type PowerschoolService struct {
	coreAPIs

	api    PowerschoolAPI
	ctxKey any
}

// PublicService `vcassist.public.v1.PublicService`
type PublicService struct {
	coreAPIs

	api  PublicAPI
	time chrono.TimeAPI
}

// NewPublicService creates a [PublicService]
func NewPublicService(coreAPIs coreAPIs, time chrono.TimeAPI) PublicService {
	assert.NotNil(chrono)

	return PublicService{
		coreAPIs: coreAPIs,
		time:     time,
	}
}

// NewMoodleService creates a [MoodleService]
func NewMoodleService(coreAPIs coreAPIs, api MoodleAPI, ctxKey any) MoodleService {
	assert.NotNil(api)
	assert.NotNil(ctxKey)

	return MoodleService{
		coreAPIs: coreAPIs,
		api:      api,
		ctxKey:   ctxKey,
	}
}

// NewPowerschoolService creates a [PowerschoolService]
func NewPowerschoolService(
	coreAPIs coreAPIs,
	api PowerschoolAPI,
	ctxKey any,
) PowerschoolService {
	assert.NotNil(api)
	assert.NotNil(ctxKey)

	return PowerschoolService{
		coreAPIs: coreAPIs,
		api:      api,
		ctxKey:   ctxKey,
	}
}
