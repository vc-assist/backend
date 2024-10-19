package core

import (
	"vcassist-backend/lib/telemetry"
	"vcassist-backend/lib/util/restyutil"
)

var tracer = telemetry.Tracer("vcassist.lib.scrapers.moodle.core")
var restyInstrumentOutput restyutil.InstrumentOutput

func SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	restyInstrumentOutput = out
}
