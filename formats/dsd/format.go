package dsd

import "errors"

var (
	ErrIncompatibleFormat = errors.New("dsd: format is incompatible with operation")
	ErrIsRaw              = errors.New("dsd: given data is in raw format")
	ErrNoMoreSpace        = errors.New("dsd: no more space left after reading dsd type")
	ErrUnknownFormat      = errors.New("dsd: format is unknown")
)

type SerializationFormat uint8

const (
	AUTO    SerializationFormat = 0
	RAW     SerializationFormat = 1
	CBOR    SerializationFormat = 67 // C
	GenCode SerializationFormat = 71 // G
	JSON    SerializationFormat = 74 // J
	MsgPack SerializationFormat = 77 // M
)

type CompressionFormat uint8

const (
	AutoCompress CompressionFormat = 0
	GZIP         CompressionFormat = 90 // Z
)

type SpecialFormat uint8

const (
	LIST SpecialFormat = 76 // L
)

var (
	DefaultSerializationFormat = JSON
	DefaultCompressionFormat   = GZIP
)

// ValidateSerializationFormat validates if the format is for serialization,
// and returns the validated format as well as the result of the validation.
// If called on the AUTO format, it returns the default serialization format.
func (format SerializationFormat) ValidateSerializationFormat() (validated SerializationFormat, ok bool) {
	switch format {
	case AUTO:
		return DefaultSerializationFormat, true
	case RAW:
		return format, true
	case CBOR:
		return format, true
	case GenCode:
		return format, true
	case JSON:
		return format, true
	case MsgPack:
		return format, true
	default:
		return 0, false
	}
}

// ValidateCompressionFormat validates if the format is for compression,
// and returns the validated format as well as the result of the validation.
// If called on the AUTO format, it returns the default compression format.
func (format CompressionFormat) ValidateCompressionFormat() (validated CompressionFormat, ok bool) {
	switch format {
	case AutoCompress:
		return DefaultCompressionFormat, true
	case GZIP:
		return format, true
	default:
		return 0, false
	}
}
