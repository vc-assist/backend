package main

import (
	"flag"
	"net/http"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/sqliteutil"
	"vcassist-backend/services/keychain"
	"vcassist-backend/services/linker/db"
)

type KeychainConfig struct {
	Database string `json:"database"`
}

type Config struct {
	Keychain        KeychainConfig        `json:"keychain"`
	Linker          LinkerConfig          `json:"linker"`
	VCSis           VCSisConfig           `json:"vcsis"`
	VCMoodleScraper VCMoodleScraperConfig `json:"vcmoodle_scraper"`
	VCMoodleServer  VCMoodleServerConfig  `json:"vcmoodle_server"`
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose logging/instrumentation.")
	initialScrape := flag.Bool("scrape", false, "Trigger scraping immediately on run.")
	flag.Parse()

	ctx := serviceutil.SignalContext()

	InitTelemetry(ctx, *verbose)

	cfg, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("read config", err)
	}

	mux := http.NewServeMux()

	linker, err := InitLinker(mux, cfg.Linker)
	if err != nil {
		serviceutil.Fatal("init linker", err)
	}
	db, err := sqliteutil.OpenDB(db.Schema, cfg.Keychain.Database)
	if err != nil {
		serviceutil.Fatal("init keychain", err)
	}
	keychainService := keychain.NewService(ctx, db)

	err = InitVCMoodleScraper(ctx, cfg.VCMoodleScraper, initialScrape)
	if err != nil {
		serviceutil.Fatal("init vcmoodle scraper", err)
	}
	err = InitVCMoodleServer(mux, keychain.NewAuthInterceptor(db), cfg.VCMoodleServer, keychainService)
	if err != nil {
		serviceutil.Fatal("init vcmoodle server", err)
	}
	err = InitVCSis(mux, cfg.VCSis, keychain.NewAuthInterceptor(db), keychainService, linker)
	if err != nil {
		serviceutil.Fatal("init vcsis", err)
	}

	go serviceutil.StartHttpServer(8000, mux)
	<-ctx.Done()
}
