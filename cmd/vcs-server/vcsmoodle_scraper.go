package main

import (
	"context"
	"database/sql"
	"time"
	configlibsql "vcassist-backend/lib/configutil/libsql"
	"vcassist-backend/lib/scrapers/moodle/core"
	"vcassist-backend/lib/scrapers/moodle/view"
	"vcassist-backend/lib/timezone"
	"vcassist-backend/services/vcsmoodle/db"
	"vcassist-backend/services/vcsmoodle/scraper"
)

type VcsmoodleScraperConfig struct {
	Database configlibsql.Struct `json:"database"`
	Username string              `json:"username"`
	Password string              `json:"password"`
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

func vcsmoodleScrapeWorker(ctx context.Context, db *sql.DB, client view.Client) {
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
			scraper.Scrape(ctx, db, client)
		}
	}
}

func InitVcsmoodleScraper(ctx context.Context, cfg VcsmoodleScraperConfig) error {
	database, err := cfg.Database.OpenDB(db.Schema)
	if err != nil {
		return err
	}
	client, err := createMoodleClient(cfg.Username, cfg.Password)
	if err != nil {
		return err
	}
	go vcsmoodleScrapeWorker(ctx, database, client)
	return nil
}
