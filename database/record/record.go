package record

// Record provides an interface for uniformally handling database records.
type Record interface {
	Key() string          // test:config
	DatabaseName() string // test
	DatabaseKey() string  // config

	SetKey(key string) // test:config
	MoveTo(key string) // test:config
	Meta() *Meta
	SetMeta(meta *Meta)

	Marshal(r Record, format uint8) ([]byte, error)
	MarshalRecord(r Record) ([]byte, error)

	Lock()
	Unlock()

	IsWrapped() bool
}
