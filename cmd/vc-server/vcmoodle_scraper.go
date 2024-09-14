package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/sqliteutil"
	"vcassist-backend/lib/timezone"
	"vcassist-backend/services/vcmoodle/db"
	"vcassist-backend/services/vcmoodle/scraper"
)

type VCMoodleScraperConfig struct {
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func createMoodleClient(username, password string) (view.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	coreClient, err := core.NewClient(ctx, core.ClientOptions{
		BaseUrl: "https://learn.vcs.net",
	})
	if err != nil {
		return view.Client{}, err
	}
	err = coreClient.LoginUsernamePassword(ctx, username, password)
	if err != nil {
		return view.Client{}, err
	}
	client, err := view.NewClient(ctx, coreClient)
	if err != nil {
		return view.Client{}, err
	}

	return client, nil
}

func vcmoodleScrapeWorker(ctx context.Context, db *sql.DB, username, password string) {
	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := timezone.Now()
			if current.Hour() != 3 {
				continue
			}

			client, err := createMoodleClient(username, password)
			if err != nil {
				slog.ErrorContext(ctx, "create moodle client", "err", err)
				continue
			}
			scraper.Scrape(ctx, db, client)
		}
	}
}

func InitVCMoodleScraper(ctx context.Context, cfg VCMoodleScraperConfig) error {
	database, err := sqliteutil.OpenDB(db.Schema, cfg.Database)
	if err != nil {
		return err
	}

	_, err = createMoodleClient(cfg.Username, cfg.Password)
	if err != nil {
		return err
	}
	go vcmoodleScrapeWorker(ctx, database, cfg.Username, cfg.Password)

	return nil
}
