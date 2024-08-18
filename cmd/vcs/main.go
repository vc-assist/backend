package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/gradesnapshots/v1/gradesnapshotsv1connect"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/proto/vcassist/services/powerservice/v1/powerservicev1connect"
	"vcassist-backend/proto/vcassist/services/studentdata/v1/studentdatav1connect"
	"vcassist-backend/proto/vcassist/services/vcsmoodle/v1/vcsmoodlev1connect"
	authdb "vcassist-backend/services/auth/db"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/vcs"
	vcsdb "vcassist-backend/services/vcs/db"

	"connectrpc.com/connect"
)

type NoAuthConfig struct {
	BaseUrl string `json:"base_url"`
}

type AccessTokenConfig struct {
	BaseUrl string `json:"base_url"`
	// an access token that must be provided in the `Authorization` header in the format
	// of `Authorization=Bearer <access token>` for certain services
	// if this value not specified, authorization will be skipped
	AccessToken string `json:"access_token"`
}

type ServicesConfig struct {
	Keychain       NoAuthConfig      `json:"keychain"`
	GradeSnapshots NoAuthConfig      `json:"gradesnapshots"`
	Linker         AccessTokenConfig `json:"linker"`
	Powerservice   NoAuthConfig      `json:"powerservice"`
	Vcsmoodle      NoAuthConfig      `json:"vcsmoodle"`
}

type Config struct {
	Database     configlibsql.Struct `json:"database"`
	AuthDatabase configlibsql.Struct `json:"auth_database"`
	Services     ServicesConfig      `json:"services"`

	// the maximum duration to cache the student data for
	MaxDataCacheSeconds int `json:"max_data_cache_seconds"`
	// the path to a JSON file that explicitly defines grade weights for various courses
	WeightsFile string `json:"weights_file"`
}

func main() {
	ctx := serviceutil.SignalContext()

	telemetry.SetupFromEnv(ctx, "studentdata")
	telemetry.InstrumentPerfStats(ctx)

	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}

	powerservice := powerservicev1connect.NewPowerschoolServiceClient(
		http.DefaultClient,
		config.Services.Powerservice.BaseUrl,
	)
	vcsmoodle := vcsmoodlev1connect.NewMoodleServiceClient(
		http.DefaultClient,
		config.Services.Vcsmoodle.BaseUrl,
	)
	linker := linkerv1connect.NewLinkerServiceClient(
		http.DefaultClient,
		config.Services.Linker.BaseUrl,
		connect.WithInterceptors(
			serviceutil.ProvideAccessTokenInterceptor(config.Services.Linker.AccessToken),
		),
	)
	gradesnapshots := gradesnapshotsv1connect.NewGradeSnapshotsServiceClient(
		http.DefaultClient,
		config.Services.GradeSnapshots.BaseUrl,
	)

	db, err := config.Database.OpenDB(vcsdb.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open self DB", err)
	}

	var weights vcs.WeightData
	if config.WeightsFile != "" {
		weightsFile, err := os.ReadFile(config.WeightsFile)
		if err != nil {
			serviceutil.Fatal("failed to read weights file", err)
		}
		err = json.Unmarshal(weightsFile, &weights)
		if err != nil {
			serviceutil.Fatal("failed to parse weights file", err)
		}
	}

	service := vcs.NewService(
		db,
		vcs.Options{
			Gradesnapshots:       gradesnapshots,
			Powerschool:          powerservice,
			Moodle:               vcsmoodle,
			Linker:               linker,
			MaxDataCacheDuration: time.Duration(config.MaxDataCacheSeconds) * time.Second,
			Weights:              weights,
		},
	)

	authDb, err := config.AuthDatabase.OpenDB(authdb.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open auth DB", err)
	}
	verify := verifier.NewVerifier(authDb)
	authInterceptor := verifier.NewAuthInterceptor(verify)

	studentDataMux := http.NewServeMux()
	studentDataMux.Handle(studentdatav1connect.NewStudentDataServiceHandler(
		service,
		connect.WithInterceptors(authInterceptor),
	))

	go serviceutil.StartHttpServer(9111, studentDataMux)

	<-ctx.Done()
}
