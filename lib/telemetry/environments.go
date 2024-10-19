package telemetry

import (
	"context"
	"vcassist-backend/lib/util/configutil"
)

var setupTestEnvironments = map[string]bool{}

// sets up telemetry in a testing environment, ensuring that it isn't
// set up more than once
func SetupForTesting(serviceName string) func() {
	_, setupAlready := setupTestEnvironments[serviceName]
	if setupAlready {
		return func() {}
	}

	InitSlog(true)
	err := SetupFromEnv(context.Background(), serviceName)
	if err != nil {
		panic(err)
	}

	return func() {
		err = Shutdown(context.Background())
		if err != nil {
			panic(err)
		}
	}
}

// searches up the filesystem from the cwd to find a file
// called telemetry.json5, once found it will then use it
// as a config to setup telemetry
func SetupFromEnv(ctx context.Context, serviceName string) error {
	config, err := configutil.ReadRecursively[config]("telemetry.json5")
	if err != nil {
		return err
	}
	return Setup(ctx, serviceName, config)
}
