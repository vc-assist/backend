package core

import (
	"vcassist-backend/lib/util/restyutil"
	"vcassist-backend/lib/telemetry"
)

var tracer = telemetry.Tracer("vcassist.lib.scrapers.moodle.core")
var restyInstrumentOutput restyutil.InstrumentOutput

func SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	restyInstrumentOutput = out
}
