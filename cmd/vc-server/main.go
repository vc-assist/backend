package main

import (
	"flag"
	"net/http"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/serviceutil"
)

type Config struct {
	Auth            AuthConfig            `json:"auth"`
	Keychain        KeychainConfig        `json:"keychain"`
	Linker          LinkerConfig          `json:"linker"`
	VCSis           VCSisConfig           `json:"vcsis"`
	VCMoodleScraper VCMoodleScraperConfig `json:"vcmoodle_scraper"`
	VCMoodleServer  VCMoodleServerConfig  `json:"vcmoodle_server"`
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose logging/instrumentation.")
	flag.Parse()

	ctx := serviceutil.SignalContext()

	InitTelemetry(ctx, *verbose)

	cfg, err := configutil.ReadConfig[Config]("config.json5")
	if err != nil {
		serviceutil.Fatal("read config", err)
	}

	mux := http.NewServeMux()

	verify, err := InitAuth(mux, cfg.Auth)
	if err != nil {
		serviceutil.Fatal("init auth", err)
	}
	linker, err := InitLinker(mux, cfg.Linker)
	if err != nil {
		serviceutil.Fatal("init linker", err)
	}
	keychain, err := InitKeychain(ctx, cfg.Keychain)
	if err != nil {
		serviceutil.Fatal("init keychain", err)
	}

	err = InitVCMoodleScraper(ctx, cfg.VCMoodleScraper)
	if err != nil {
		serviceutil.Fatal("init vcmoodle scraper", err)
	}
	err = InitVCMoodleServer(mux, verify, cfg.VCMoodleServer, keychain)
	if err != nil {
		serviceutil.Fatal("init vcmoodle server", err)
	}
	err = InitVCSis(mux, verify, cfg.VCSis, keychain, linker)
	if err != nil {
		serviceutil.Fatal("init vcsis", err)
	}

	go serviceutil.StartHttpServer(8000, mux)
	<-ctx.Done()
}
