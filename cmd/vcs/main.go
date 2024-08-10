package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	"vcassist-backend/lib/configuration"
	configlibsql "vcassist-backend/lib/configuration/libsql"
	"vcassist-backend/lib/osutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/proto/vcassist/services/linker/v1/linkerv1connect"
	"vcassist-backend/proto/vcassist/services/studentdata/v1/studentdatav1connect"
	"vcassist-backend/services/auth/verifier"
	"vcassist-backend/services/gradesnapshots"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/linker"
	"vcassist-backend/services/powerservice"
	"vcassist-backend/services/vcs"
	"vcassist-backend/services/vcsmoodle"

	"connectrpc.com/connect"
	"github.com/dgraph-io/badger/v4"
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
	config, err := configuration.ReadConfig[Config]("config.json5")
	if err != nil {
		fatalerr("failed to read config", err)
	}

	var db *sql.DB
	db, err = config.Database.GradeSnapshot.OpenDB()
	if err != nil {
		fatalerr("failed to open gradesnapshot database", err)
	}
	gradesnapshotService := gradesnapshots.NewService(db)

	db, err = config.Database.Keychain.OpenDB()
	if err != nil {
		fatalerr("failed to open keychain database", err)
	}
	keychainService := keychain.NewService(context.Background(), db)

	db, err = config.Database.Powerservice.OpenDB()
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

	moodleCache, err := badger.Open(badger.DefaultOptions("moodle-cache.db"))
	if err != nil {
		fatalerr("failed to open moodle KV cache", err)
	}
	vcsmoodleService := vcsmoodle.NewService(moodleCache, keychainService)

	db, err = config.Database.Linker.OpenDB()
	if err != nil {
		fatalerr("failed to open linker database", err)
	}
	linkerService := linker.NewService(db)

	db, err = config.Database.Self.OpenDB()
	if err != nil {
		fatalerr("failed to open self DB", err)
	}
	service := vcs.NewService(
		db,
		powerserviceService,
		vcsmoodleService,
		linkerService,
		gradesnapshotService,
		time.Duration(config.MaxDataCacheSeconds)*time.Second,
	)

	db, err = config.Database.Auth.OpenDB()
	if err != nil {
		fatalerr("failed to open auth DB", err)
	}
	verify := verifier.NewVerifier(db)
	authInterceptor := verifier.NewAuthInterceptor(verify)

	linkerMux := http.NewServeMux()
	linkerMux.Handle(linkerv1connect.NewLinkerServiceHandler(linkerService))

	studentDataMux := http.NewServeMux()
	studentDataMux.Handle(studentdatav1connect.NewStudentDataServiceHandler(
		service,
		connect.WithInterceptors(authInterceptor),
	))

	ctx := osutil.SignalContext()

	telemetry.SetupFromEnv(ctx, "cmd/vcs")

	startHttpServer(8222, linkerMux)
	startHttpServer(9111, studentDataMux)

	<-ctx.Done()
}
