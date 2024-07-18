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

	config, err := devenv.GetStateConfig[TestConfig]("moodle/core.json5")
	if err != nil {
		t.Skip("skipping moodle/core test because there is no valid test config at .dev/state/moodle/core.json5")
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
