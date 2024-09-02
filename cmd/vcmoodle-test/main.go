package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/vcmoodle/scraper"

	_ "modernc.org/sqlite"
)

type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
	core.SetRestyInstrumentOutput(restyutil.NewFilesystemOutput("<dev_state>/vcmoodle/resty"))

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
	telemetry.InitSlog(true)

	cfg, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}

	path, err := devenv.ResolvePath("<dev_state>/vcmoodle_test.db")
	if err != nil {
		serviceutil.Fatal("failed to resolve db path", err)
	}
	os.Remove(path)
	out, err := sql.Open("sqlite", path)
	if err != nil {
		serviceutil.Fatal("failed to open db", err)
	}
	defer out.Close()

	client := createClient(cfg.Username, cfg.Password)

	t1 := time.Now()
	scraper.Scrape(context.Background(), out, client)
	t2 := time.Now()

	slog.Info("scraping time", "seconds", t2.Sub(t1).Seconds())
}
