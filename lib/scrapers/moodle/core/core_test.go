package core

import (
	"context"
	"encoding/json"
	"testing"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/telemetry"

	_ "embed"
)

func getTestConfig(t testing.TB) devenv.MoodleTestConfig {
	contents, err := devenv.GetStateFile("moodle_config.json")
	if err != nil {
		t.Fatal(err)
	}

	var cached devenv.MoodleTestConfig
	err = json.Unmarshal(contents, &cached)
	if err != nil {
		t.Fatal(err)
	}
	return cached
}

func TestClient(t *testing.T) {
	cleanup := telemetry.SetupForTesting(t, "test:scrapers/moodle/core")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestClient")
	defer span.End()

	config := getTestConfig(t)
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
