package powerservice

import (
	"vcassist-backend/lib/restyutil"
)

var restyInstrumentOutput restyutil.InstrumentOutput

func SetRestyInstrumentOutput(output restyutil.InstrumentOutput) {
	restyInstrumentOutput = output
}
