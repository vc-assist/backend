package vcsnet

import (
	"vcassist-backend/lib/restyutil"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const library_name = "vcassist.lib.scrapers.vcsnet"

var tracer = otel.Tracer(library_name)

func SetTracerProvider(provider trace.TracerProvider) {
	tracer = provider.Tracer(library_name)
}

func SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	client = resty.New()
	restyutil.InstrumentClient(client, tracer, out)
}
