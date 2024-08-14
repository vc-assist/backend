package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/osutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/proto/vcassist/services/studentdata/v1/studentdatav1connect"
	authdb "vcassist-backend/services/auth/db"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/gradesnapshots"
	gradesnapshotsdb "vcassist-backend/services/gradesnapshots/db"
	"vcassist-backend/services/keychain"
	keychaindb "vcassist-backend/services/keychain/db"
	"vcassist-backend/services/linker"
	linkerdb "vcassist-backend/services/linker/db"
	"vcassist-backend/services/powerservice"
	powerservicedb "vcassist-backend/services/powerservice/db"
	"vcassist-backend/services/vcs"
	vcsdb "vcassist-backend/services/vcs/db"
	"vcassist-backend/services/vcsmoodle"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func fatalerr(message string, err error) {
	slog.Error(message, "err", err.Error())
	os.Exit(1)
}

type DatabaseConfig struct {
	Auth          configlibsql.Struct `json:"auth"`
	Keychain      configlibsql.Struct `json:"keychain"`
	GradeSnapshot configlibsql.Struct `json:"grade_snapshot"`
	Linker        configlibsql.Struct `json:"linker"`
	Powerservice  configlibsql.Struct `json:"powerservice"`
	Self          configlibsql.Struct `json:"self"`
}

type Config struct {
	Database DatabaseConfig `json:"database"`
	// the maximum duration to cache the student data for
	MaxDataCacheSeconds int `json:"max_data_cache_seconds"`
	// the path to a JSON file that explicitly defines grade weights for various courses
	WeightsFile string `json:"weights_file"`
	// an access token that must be provided in the `Authorization` header in the format
	// of `Authorization=Bearer <access token>`
	// if this value not specified, authorization checks will not be performed
	LinkerAccessToken string `json:"linker_access_token"`
}

func startHttpServer(port int, mux *http.ServeMux) {
	go func() {
		slog.Info("listening to gRPC...", "port", port)
		err := http.ListenAndServe(
			fmt.Sprintf("0.0.0.0:%d", port),
			h2c.NewHandler(mux, &http2.Server{}),
		)
		if err != nil {
			fatalerr(fmt.Sprintf("failed to listen on port %d", port), err)
		}
	}()
}

func main() {
	config, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		fatalerr("failed to read config", err)
	}

	var db *sql.DB
	db, err = config.Database.GradeSnapshot.OpenDB(gradesnapshotsdb.Schema)
	if err != nil {
		fatalerr("failed to open gradesnapshot database", err)
	}
	gradesnapshotService := gradesnapshots.NewService(db)

	db, err = config.Database.Keychain.OpenDB(keychaindb.Schema)
	if err != nil {
		fatalerr("failed to open keychain database", err)
	}
	keychainService := keychain.NewService(context.Background(), db)

	db, err = config.Database.Powerservice.OpenDB(powerservicedb.Schema)
	if err != nil {
		fatalerr("failed to open powerschoold database", err)
	}
	powerserviceService := powerservice.NewService(
		db,
		keychainService,
		"https://vcsnet.powerschool.com",
		powerservice.OAuthConfig{
			BaseLoginUrl: "https://accounts.google.com/o/oauth2/v2/auth",
			RefreshUrl:   "https://oauth2.googleapis.com/token",
			ClientId:     "162669419438-egansm7coo8n7h301o7042kad9t9uao9.apps.googleusercontent.com",
		},
	)

	vcsmoodleService := vcsmoodle.NewService(keychainService)

	db, err = config.Database.Linker.OpenDB(linkerdb.Schema)
	if err != nil {
		fatalerr("failed to open linker database", err)
	}
	linkerService := linker.NewService(db)

	db, err = config.Database.Self.OpenDB(vcsdb.Schema)
	if err != nil {
		fatalerr("failed to open self DB", err)
	}

	var weights vcs.WeightData
	if config.WeightsFile != "" {
		weightsFile, err := os.ReadFile(config.WeightsFile)
		if err != nil {
			fatalerr("failed to read weights file", err)
		}
		err = json.Unmarshal(weightsFile, &weights)
		if err != nil {
			fatalerr("failed to parse weights file", err)
		}
	}

	service := vcs.NewService(
		db,
		vcs.Config{
			Gradesnapshots:       gradesnapshotService,
			Powerschool:          powerserviceService,
			Moodle:               vcsmoodleService,
			Linker:               linkerService,
			MaxDataCacheDuration: time.Duration(config.MaxDataCacheSeconds) * time.Second,
			Weights:              weights,
		},
	)

	db, err = config.Database.Auth.OpenDB(authdb.Schema)
	if err != nil {
		fatalerr("failed to open auth DB", err)
	}
	verify := verifier.NewVerifier(db)
	authInterceptor := verifier.NewAuthInterceptor(verify)

	linkerMux := http.NewServeMux()
	linkerMux.Handle(linkerv1connect.NewLinkerServiceHandler(
		linkerService,
		connect.WithInterceptors(
			linkerInterceptor(config.LinkerAccessToken),
		),
	))

	studentDataMux := http.NewServeMux()
	studentDataMux.Handle(studentdatav1connect.NewStudentDataServiceHandler(
		service,
		connect.WithInterceptors(authInterceptor),
	))

	ctx := osutil.SignalContext()

	telemetry.SetupFromEnv(ctx, "cmd/vcs")
	telemetry.InstrumentPerfStats(ctx)

	startHttpServer(8222, linkerMux)
	startHttpServer(9111, studentDataMux)

	<-ctx.Done()
}

func linkerInterceptor(accessToken string) connect.Interceptor {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if accessToken == "" {
				return next(ctx, req)
			}
			token := strings.Split(req.Header().Get("Authorization"), " ")
			if len(token) != 2 || token[1] != accessToken {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("Unauthorized"))
			}
			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}
