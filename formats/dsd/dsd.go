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

// Load loads an dsd structured data blob into the given interface.
func Load(data []byte, t interface{}) (format SerializationFormat, err error) {
	formatID, read, err := loadFormat(data)
	if err != nil {
		return 0, err
	}

	format, ok := SerializationFormat(formatID).ValidateSerializationFormat()
	if ok {
		return format, LoadAsFormat(data[read:], format, t)
	}
	return DecompressAndLoad(data[read:], CompressionFormat(format), t)
}

// LoadAsFormat loads a data blob into the interface using the specified format.
func LoadAsFormat(data []byte, format SerializationFormat, t interface{}) (err error) {
	switch format {
	case JSON:
		err = json.Unmarshal(data, t)
		if err != nil {
			return fmt.Errorf("dsd: failed to unpack json: %w, data: %s", err, utils.SafeFirst16Bytes(data))
		}
		return nil
	case CBOR:
		err = cbor.Unmarshal(data, t)
		if err != nil {
			return fmt.Errorf("dsd: failed to unpack cbor: %w, data: %s", err, utils.SafeFirst16Bytes(data))
		}
		return nil
	case GenCode:
		genCodeStruct, ok := t.(GenCodeCompatible)
		if !ok {
			return errors.New("dsd: gencode is not supported by the given data structure")
		}
		_, err = genCodeStruct.GenCodeUnmarshal(data)
		if err != nil {
			return fmt.Errorf("dsd: failed to unpack gencode: %w, data: %s", err, utils.SafeFirst16Bytes(data))
		}
		return nil
	default:
		return ErrIncompatibleFormat
	}
}

func loadFormat(data []byte) (format uint8, read int, err error) {
	format, read, err = varint.Unpack8(data)
	if err != nil {
		return 0, 0, err
	}
	if len(data) <= read {
		return 0, 0, ErrNoMoreSpace
	}

	return format, read, nil
}

// Dump stores the interface as a dsd formatted data structure.
func Dump(t interface{}, format SerializationFormat) ([]byte, error) {
	return DumpIndent(t, format, "")
}

// DumpIndent stores the interface as a dsd formatted data structure with indentation, if available.
func DumpIndent(t interface{}, format SerializationFormat, indent string) ([]byte, error) {
	format, ok := format.ValidateSerializationFormat()
	if !ok {
		return nil, ErrIncompatibleFormat
	}

	f := varint.Pack8(uint8(format))
	var data []byte
	var err error
	switch format {
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
			return nil, fmt.Errorf("dsd: failed to pack gencode struct: %w", err)
		}
	default:
		return nil, ErrIncompatibleFormat
	}

	// TODO: Find a better way to do this.
	f = append(f, data...)
	return f, nil
}
