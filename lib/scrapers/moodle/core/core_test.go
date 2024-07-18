package core

import (
	"context"
	"testing"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/telemetry"

	_ "embed"
)

func TestClient(t *testing.T) {
	cleanup := telemetry.SetupForTesting(t, "test:scrapers/moodle/core")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestClient")
	defer span.End()

	config, err := devenv.GetStateConfig[devenv.MoodleTestConfig]("moodle_config.json5")
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(ctx, ClientOptions{
		BaseUrl: config.BaseUrl,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = client.LoginUsernamePassword(ctx, config.Username, config.Password)
	if err != nil {
		t.Fatal(err)
	}
}
