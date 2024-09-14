package main

import (
	"context"
	"vcassist-backend/cmd/vcmoodle-cli/commands"
	"vcassist-backend/lib/telemetry"
)

func main() {
	telemetry.SetupFromEnv(context.Background(), "vcmoodle-cli")
	telemetry.InitSlog(true)
	commands.ExecuteContext(context.Background())
}
