package core

import (
	"vcassist-backend/lib/restyutil"
	"vcassist-backend/lib/telemetry"
)

var tracer = telemetry.Tracer("vcassist.lib.scrapers.moodle.core")
var restyInstrumentOutput restyutil.InstrumentOutput

func SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	restyInstrumentOutput = out
}
