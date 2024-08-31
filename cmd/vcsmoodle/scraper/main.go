package main

import (
	"context"
	"log/slog"
	"os"
	"time"
	"vcassist-backend/lib/configutil"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/services/vcsmoodle/db"
	"vcassist-backend/services/vcsmoodle/scraper"

	"github.com/lmittmann/tint"
)

type Config struct {
	Database configlibsql.Struct `json:"database"`
	Username string              `json:"username"`
	Password string              `json:"password"`
}

func createClient(username, password string) view.Client {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: "https://learn.vcs.net",
	})
	if err != nil {
		serviceutil.Fatal("failed to initialize core moodle client", err)
	}
	coreClient.SetRestyInstrumentOutput(restyutil.NewFilesystemOutput("<dev_state>/vcsmoodle/resty"))

	err = coreClient.LoginUsernamePassword(ctx, username, password)
	if err != nil {
		serviceutil.Fatal("failed to login to moodle", err)
	}
	client, err := view.NewClient(ctx, coreClient)
	if err != nil {
		serviceutil.Fatal("failed to initialize client", err)
	}

	return client
}

func main() {
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	}))
	slog.SetDefault(logger)

	cfg, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}
	out, err := cfg.Database.OpenDB(db.Schema)
	if err != nil {
		serviceutil.Fatal("failed to open db", err)
	}

	client := createClient(cfg.Username, cfg.Password)

	t1 := time.Now()
	scraper.Scrape(context.Background(), out, client)
	t2 := time.Now()

	slog.Info("scraping time", "seconds", t2.Sub(t1).Seconds())
}
