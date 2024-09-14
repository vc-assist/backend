package core

import (
	"context"
	"testing"
	"vcassist-backend/lib/configutil"
	"vcassist-backend/lib/telemetry"

	_ "embed"
)

func TestClient(t *testing.T) {
	cleanup := telemetry.SetupForTesting("test:scrapers/moodle/core")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestClient")
	defer span.End()

	config, err := configutil.ReadConfig[TestConfig](".dev/test_moodle/config.json5")
	if err != nil {
		t.Fatal("failed to read test config at .dev/test_moodle/config.json5")
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
