package record

import (
	"github.com/safing/portbase/formats/dsd"
)

// Reimport DSD storage types
const (
	AUTO    = dsd.AUTO
	STRING  = dsd.STRING  // S
	BYTES   = dsd.BYTES   // X
	JSON    = dsd.JSON    // J
	BSON    = dsd.BSON    // B
	GenCode = dsd.GenCode // G
)
