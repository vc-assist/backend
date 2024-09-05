package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"runtime"
	"time"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/services/vcmoodle/scraper"

	"net/http"
	_ "net/http/pprof"

	"github.com/pkg/profile"
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

func startProfiling() {
	go func() {
		var memStats runtime.MemStats
		ticker := time.NewTicker(time.Second)
		for {
			<-ticker.C
			runtime.ReadMemStats(&memStats)
			memoryUsageMb := int64(memStats.Alloc / 1_000_000)
			slog.Debug("memory usage", "megabytes", memoryUsageMb)
		}
	}()

	defer profile.Start(profile.MemProfile).Stop()
	go func() {
		http.ListenAndServe(":8080", nil)
	}()
}

func main() {
	profile := flag.Bool("profile", false, "enable memory profiling")

	if *profile {
		startProfiling()
	}

	telemetry.SetupFromEnv(context.Background(), "vcmoodle_test")
	telemetry.InitSlog(true)

	cfg, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("failed to read config", err)
	}

	path, err := devenv.ResolvePath("<dev_state>/vcmoodle.db")
	if err != nil {
		serviceutil.Fatal("failed to resolve db path", err)
	}
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
