package model

// Model provides an interface for uniformally handling database records.
type Model interface {
	Key() string
	SetKey(key string)
	MoveTo(key string)
	Meta() *Meta
	SetMeta(meta *Meta)
	Marshal(format uint8) ([]byte, error)
}
