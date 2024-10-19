package core

import (
	"context"
	"testing"
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/util/configutil"

	_ "embed"
)

type TestConfig struct {
	BaseUrl  string `json:"base_url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func TestClient(t *testing.T) {
	cleanup := telemetry.SetupForTesting("test:scrapers/moodle/core")
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
