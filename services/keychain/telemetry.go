package keychain

import "vcassist-backend/lib/util/restyutil"

var restyInstrumentOutput restyutil.InstrumentOutput

func SetRestyInstrumentOutput(out restyutil.InstrumentOutput) {
	restyInstrumentOutput = out
}
