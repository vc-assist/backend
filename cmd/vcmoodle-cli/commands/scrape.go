package commands

import (
	"context"
	"log/slog"
	"time"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/lib/sqliteutil"
	"vcassist-backend/services/vcmoodle/db"
	"vcassist-backend/services/vcmoodle/scraper"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var scrapeDb *string

func init() {
	scrapeDb = scrapeCmd.Flags().String("db", "results.db", "The database to write scrape results to.")
	rootCmd.AddCommand(scrapeCmd)
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
	core.SetRestyInstrumentOutput(restyutil.NewFilesystemOutput(".dev/resty/scraper"))

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

var scrapeCmd = &cobra.Command{
	Use:   "scrape [--db <path/to/output.db>]",
	Short: "Scrapes moodle according to a config and writes to a database.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := configutil.ReadConfig[Config]("config.json5")
		if err != nil {
			serviceutil.Fatal("failed to read config", err)
		}

		slog.Info("scraping using user", "username", cfg.Username)
		client := createClient(cfg.Username, cfg.Password)

		out, err := sqliteutil.OpenDB(db.Schema, *scrapeDb)
		if err != nil {
			serviceutil.Fatal("failed to open db", err)
		}
		defer out.Close()

		t1 := time.Now()
		scraper.Scrape(context.Background(), out, client)
		t2 := time.Now()

		slog.Info("scraping time", "seconds", t2.Sub(t1).Seconds())
	},
}
