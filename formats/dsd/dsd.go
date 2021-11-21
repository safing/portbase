package dsd

// dynamic structured data
// check here for some benchmarks: https://github.com/alecthomas/go_serialization_benchmarks

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/safing/portbase/formats/varint"
	"github.com/safing/portbase/utils"
)

// Types.
const (
	AUTO = 0
	NONE = 1

	// Special types.
	LIST = 76 // L

	// Serialization types.
	CBOR    = 67 // C
	GenCode = 71 // G
	JSON    = 74 // J
	STRING  = 83 // S
	BYTES   = 88 // X

	// Compression types.
	GZIP = 90 // Z
)

// Errors.
var (
	errNoMoreSpace    = errors.New("dsd: no more space left after reading dsd type")
	errNotImplemented = errors.New("dsd: this type is not yet implemented")
)

// Load loads an dsd structured data blob into the given interface.
func Load(data []byte, t interface{}) (interface{}, error) {
	format, read, err := varint.Unpack8(data)
	if err != nil {
		return nil, err
	}
	if len(data) <= read {
		return nil, errNoMoreSpace
	}

	switch format {
	case GZIP:
		return DecompressAndLoad(data[read:], format, t)
	default:
		return LoadAsFormat(data[read:], format, t)
	}
}

// LoadAsFormat loads a data blob into the interface using the specified format.
func LoadAsFormat(data []byte, format uint8, t interface{}) (interface{}, error) {
	switch format {
	case STRING:
		return string(data), nil
	case BYTES:
		return data, nil
	case JSON:
		err := json.Unmarshal(data, t)
		if err != nil {
			return nil, fmt.Errorf("dsd: failed to unpack json: %s, data: %s", err, utils.SafeFirst16Bytes(data))
		}
		return t, nil
	case CBOR:
		err := cbor.Unmarshal(data, t)
		if err != nil {
			return nil, fmt.Errorf("dsd: failed to unpack cbor: %s, data: %s", err, utils.SafeFirst16Bytes(data))
		}
		return t, nil
	case GenCode:
		genCodeStruct, ok := t.(GenCodeCompatible)
		if !ok {
			return nil, errors.New("dsd: gencode is not supported by the given data structure")
		}
		_, err := genCodeStruct.GenCodeUnmarshal(data)
		if err != nil {
			return nil, fmt.Errorf("dsd: failed to unpack gencode: %s, data: %s", err, utils.SafeFirst16Bytes(data))
		}
		return t, nil
	default:
		return nil, fmt.Errorf("dsd: tried to load unknown type %d, data: %s", format, utils.SafeFirst16Bytes(data))
	}
}

// Dump stores the interface as a dsd formatted data structure.
func Dump(t interface{}, format uint8) ([]byte, error) {
	return DumpIndent(t, format, "")
}

// DumpIndent stores the interface as a dsd formatted data structure with indentation, if available.
func DumpIndent(t interface{}, format uint8, indent string) ([]byte, error) {
	if format == AUTO {
		switch t.(type) {
		case string:
			format = STRING
		case []byte:
			format = BYTES
		default:
			format = JSON
		}
	}

	f := varint.Pack8(format)
	var data []byte
	var err error
	switch format {
	case STRING:
		data = []byte(t.(string))
	case BYTES:
		data = t.([]byte)
	case JSON:
		// TODO: use SetEscapeHTML(false)
		if indent != "" {
			data, err = json.MarshalIndent(t, "", indent)
		} else {
			data, err = json.Marshal(t)
		}
		if err != nil {
			return nil, err
		}
	case CBOR:
		data, err = cbor.Marshal(t)
		if err != nil {
			return nil, err
		}
	case GenCode:
		genCodeStruct, ok := t.(GenCodeCompatible)
		if !ok {
			return nil, errors.New("dsd: gencode is not supported by the given data structure")
		}
		data, err = genCodeStruct.GenCodeMarshal(nil)
		if err != nil {
			return nil, fmt.Errorf("dsd: failed to pack gencode struct: %s", err)
		}
	default:
		return nil, fmt.Errorf("dsd: tried to dump with unknown format %d", format)
	}

	r := append(f, data...)
	return r, nil
}
