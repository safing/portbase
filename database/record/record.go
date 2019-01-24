package record

import (
	"github.com/Safing/portbase/database/accessor"
)

// Record provides an interface for uniformally handling database records.
type Record interface {
	Key() string // test:config
	KeyIsSet() bool
	DatabaseName() string // test
	DatabaseKey() string  // config

	SetKey(key string) // test:config
	MoveTo(key string) // test:config
	Meta() *Meta
	SetMeta(meta *Meta)

	Marshal(self Record, format uint8) ([]byte, error)
	MarshalRecord(self Record) ([]byte, error)
	GetAccessor(self Record) accessor.Accessor

	Lock()
	Unlock()

	IsWrapped() bool
}
