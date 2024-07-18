package main

import (
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"
	"vcassist-backend/lib/configuration"
	configlibsql "vcassist-backend/lib/configuration/libsql"
	"vcassist-backend/proto/vcassist/services/studentdata/v1/studentdatav1connect"
	"vcassist-backend/services/gradesnapshots"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/linker"
	"vcassist-backend/services/powerservice"
	"vcassist-backend/services/vcs"
	"vcassist-backend/services/vcsmoodle"

	"github.com/dgraph-io/badger/v4"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func fatalerr(message string, err error) {
	slog.Error(message, "err", err.Error())
	os.Exit(1)
}

type DatabaseConfig struct {
	Keychain      configlibsql.Struct `json:"keychain"`
	GradeSnapshot configlibsql.Struct `json:"grade_snapshot"`
	Linker        configlibsql.Struct `json:"linker"`
	Powerservice  configlibsql.Struct `json:"powerservice"`
	Self          configlibsql.Struct `json:"self"`
}

type Config struct {
	Database DatabaseConfig `json:"database"`
}

func main() {
	config, err := configuration.ReadConfig[Config]("config.json")
	if err != nil {
		fatalerr("failed to read config", err)
	}

	var db *sql.DB
	db, err = config.Database.GradeSnapshot.OpenDB()
	if err != nil {
		fatalerr("failed to open gradesnapshot database", err)
	}
	gradesnapshotService := gradesnapshots.NewService(db)

	db, err = config.Database.Linker.OpenDB()
	if err != nil {
		fatalerr("failed to open linker database", err)
	}
	linkerService := linker.NewService(db)

	db, err = config.Database.Keychain.OpenDB()
	if err != nil {
		fatalerr("failed to open keychain database", err)
	}
	keychainService := keychain.NewService(db)

	db, err = config.Database.Powerservice.OpenDB()
	if err != nil {
		fatalerr("failed to open powerschoold database", err)
	}
	powerschooldService := powerservice.NewService(
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
		log.Fatal(err)
	}
	vcsmoodleService := vcsmoodle.NewService(moodleCache, keychainService)

	db, err = config.Database.Self.OpenDB()
	if err != nil {
		log.Fatal(err)
	}
	service := vcs.NewService(
		db,
		powerschooldService,
		vcsmoodleService,
		linkerService,
		gradesnapshotService,
	)

	mux := http.NewServeMux()
	mux.Handle(studentdatav1connect.NewStudentDataServiceHandler(service))

	slog.Info("listening to gRPC...", "port", 9111)
	err = http.ListenAndServe(
		"127.0.0.1:8111",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		fatalerr("failed to listen on port 9111", err)
	}
}
