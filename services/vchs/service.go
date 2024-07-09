package vchs

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("services/vchs")

type Service struct {
}
