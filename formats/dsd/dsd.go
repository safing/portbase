// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

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
	AUTO    = 0
	STRING  = 83 // S
	BYTES   = 88 // X
	JSON    = 74 // J
	BSON    = 66 // B
	GenCode = 71 // G (reserved)
)

// define errors
var errNoMoreSpace = errors.New("dsd: no more space left after reading dsd type")
var errUnknownType = errors.New("dsd: tried to unpack unknown type")
var errNotImplemented = errors.New("dsd: this type is not yet implemented")

func Load(data []byte, t interface{}) (interface{}, error) {
	if len(data) < 2 {
		return nil, errNoMoreSpace
	}

	format, read, err := varint.Unpack8(data)
	if err != nil {
		return nil, err
	}
	if len(data) <= read {
		return nil, errNoMoreSpace
	}

	return LoadAsFormat(data[read:], format, t)
}

func LoadAsFormat(data []byte, format uint8, t interface{}) (interface{}, error) {
	switch format {
	case STRING:
		return string(data), nil
	case BYTES:
		return data, nil
	case JSON:
		err := json.Unmarshal(data, t)
		if err != nil {
			return nil, err
		}
		return t, nil
		// case BSON:
		// 	err := bson.Unmarshal(data[read:], t)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	return t, nil
		// case MSGP:
		//   err := t.UnmarshalMsg(data[read:])
		//   if err != nil {
		//     return nil, err
		//   }
		//   return t, nil
	default:
		return nil, errors.New(fmt.Sprintf("dsd: tried to load unknown type %d, data: %v", format, data))
	}
}

func Dump(t interface{}, format uint8) ([]byte, error) {

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

	f := varint.Pack8(uint8(format))
	var data []byte
	var err error
	switch format {
	case STRING:
		data = []byte(t.(string))
	case BYTES:
		data = t.([]byte)
	case JSON:
		// TODO: use SetEscapeHTML(false)
		data, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
	// case BSON:
	// 	data, err = bson.Marshal(t)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// case MSGP:
	//   data, err := t.MarshalMsg(nil)
	//   if err != nil {
	//     return nil, err
	//   }
	default:
		return nil, errors.New(fmt.Sprintf("dsd: tried to dump unknown type %d", format))
	}

	r := append(f, data...)
	// log.Tracef("packing %v to %s", t, string(r))
	// return nil, errors.New(fmt.Sprintf("dsd: dumped bytes are: %v", r))
	return r, nil

}
