package main

import (
	"flag"
	"net/http"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/serviceutil"
)

type Config struct {
	Auth             AuthConfig             `json:"auth"`
	Keychain         KeychainConfig         `json:"keychain"`
	Linker           LinkerConfig           `json:"linker"`
	GradeSnapshots   GradeSnapshotsConfig   `json:"gradesnapshots"`
	Powerservice     PowerserviceConfig     `json:"powerservice"`
	VcsmoodleScraper VcsmoodleScraperConfig `json:"vcsmoodle_scraper"`
	VcsmoodleServer  VcsmoodleServerConfig  `json:"vcsmoodle_server"`
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

	err = InitAuth(mux, cfg.Auth)
	if err != nil {
		serviceutil.Fatal("init auth", err)
	}
	err = InitGradeSnapshots(mux, cfg.GradeSnapshots)
	if err != nil {
		serviceutil.Fatal("init gradesnapshots", err)
	}
	err = InitLinker(mux, cfg.Linker)
	if err != nil {
		serviceutil.Fatal("init linker", err)
	}
	keychain, err := InitKeychain(ctx, mux, cfg.Keychain)
	if err != nil {
		serviceutil.Fatal("init keychain", err)
	}

	err = InitVcsmoodleScraper(ctx, cfg.VcsmoodleScraper)
	if err != nil {
		serviceutil.Fatal("init vcsmoodle scraper", err)
	}
	err = InitVcsmoodleServer(mux, cfg.VcsmoodleServer, keychain)
	if err != nil {
		serviceutil.Fatal("init vcsmoodle server", err)
	}
	InitPowerservice(mux, cfg.Powerservice, keychain)

	go serviceutil.StartHttpServer(8000, mux)
	<-ctx.Done()
}
