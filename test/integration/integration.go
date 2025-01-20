package integration

import "vcassist-backend/internal/components/telemetry"

var tel telemetry.API

func SetTelemetry(api telemetry.API) {
	tel = api
}
