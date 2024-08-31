package edit

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const library_name = "vcassist.lib.scrapers.moodle.edit"

var tracer = otel.Tracer(library_name)

func SetTracerProvider(provider trace.TracerProvider) {
	tracer = provider.Tracer(library_name)
}
