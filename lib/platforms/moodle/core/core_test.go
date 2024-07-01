package core

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/telemetry"

	_ "embed"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

//go:embed moodle_login_page_test.html
var moodleLoginPageTest []byte

func TestGetMoodleConfig(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(moodleLoginPageTest))
	if err != nil {
		t.Fatal(err)
	}
	sesskey := getSesskey(context.Background(), doc)
	require.Equal(t, sesskey, "w11kOTXYpH")
}

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
	cleanup := telemetry.SetupForTesting(t, "test:moodle/core")
	defer cleanup()

	ctx, span := tracer.Start(context.Background(), "TestClient")
	defer span.End()

	config := getTestConfig(t)
	client, err := NewClient(ctx, ClientOptions{
		BaseUrl:  config.BaseUrl,
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = client.LoginUsernamePassword(ctx, config.Username, config.Password)
	if err != nil {
		t.Fatal(err)
	}
}
