package vcsis

import (
	"vcassist-backend/lib/util/restyutil"
)

var restyInstrumentOutput restyutil.InstrumentOutput

func SetRestyInstrumentOutput(output restyutil.InstrumentOutput) {
	restyInstrumentOutput = output
}
