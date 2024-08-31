package powerservice

import (
	"vcassist-backend/lib/restyutil"
)

// const library_name = "vcassist.services.powerservice_test"

// var tracer = otel.Tracer(library_name)
var restyInstrumentOutput restyutil.InstrumentOutput

// func SetTraceProvider(provider trace.TracerProvider) {
// 	tracer = provider.Tracer(library_name)
// }

func SetRestyInstrumentOutput(output restyutil.InstrumentOutput) {
	restyInstrumentOutput = output
}
