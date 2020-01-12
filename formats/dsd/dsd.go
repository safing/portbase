package dsd

// dynamic structured data
// check here for some benchmarks: https://github.com/alecthomas/go_serialization_benchmarks

import (
	"encoding/json"
	"errors"
	"fmt"

	// "github.com/pkg/bson"

	"github.com/safing/portbase/formats/varint"
)

// define types
const (
	AUTO = 0
	NONE = 1

	// special
	LIST = 76 // L

	// serialization
	STRING  = 83 // S
	BYTES   = 88 // X
	JSON    = 74 // J
	BSON    = 66 // B
	GenCode = 71 // G

	// compression
	GZIP = 90 // Z
)

// define errors
var errNoMoreSpace = errors.New("dsd: no more space left after reading dsd type")
var errNotImplemented = errors.New("dsd: this type is not yet implemented")

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
			return nil, fmt.Errorf("dsd: failed to unpack json data: %s", data)
		}
		return t, nil
	case BSON:
		return nil, errNotImplemented
	// 	err := bson.Unmarshal(data[read:], t)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return t, nil
	case GenCode:
		genCodeStruct, ok := t.(GenCodeCompatible)
		if !ok {
			return nil, errors.New("dsd: gencode is not supported by the given data structure")
		}
		_, err := genCodeStruct.GenCodeUnmarshal(data)
		if err != nil {
			return nil, fmt.Errorf("dsd: failed to unpack gencode data: %s", err)
		}
		return t, nil
	default:
		return nil, fmt.Errorf("dsd: tried to load unknown type %d, data: %v", format, data)
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
	case BSON:
		return nil, errNotImplemented
	// 	data, err = bson.Marshal(t)
	// 	if err != nil {
	// 		return nil, err
	// 	}
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
