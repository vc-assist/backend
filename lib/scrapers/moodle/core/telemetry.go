package core

import (
	"vcassist-backend/lib/restyutil"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const library_name = "vcassist.lib.scrapers.moodle.core"

var tracer = otel.Tracer(library_name)
var restyInstrumentOutput restyutil.InstrumentOutput

func SetTracerProvider(provider trace.TracerProvider) {
	tracer = provider.Tracer(library_name)
}

func SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	restyInstrumentOutput = out
}
