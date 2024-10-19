package vcsnet

import (
	"vcassist-backend/lib/util/restyutil"
	"vcassist-backend/lib/telemetry"

	"github.com/go-resty/resty/v2"
)

var tracer = telemetry.Tracer("vcassist.lib.scrapers.vcsnet")

func SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	client = resty.New()
	restyutil.InstrumentClient(client, tracer, out)
}
