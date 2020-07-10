package config

import (
	"fmt"
	"math"
)

type valueCache struct {
	stringVal      string
	stringArrayVal []string
	intVal         int64
	boolVal        bool
}

func (vc *valueCache) getData(opt *Option) interface{} {
	switch opt.OptType {
	case OptTypeBool:
		return vc.boolVal
	case OptTypeInt:
		return vc.intVal
	case OptTypeString:
		return vc.stringVal
	case OptTypeStringArray:
		return vc.stringArrayVal
	default:
		return nil
	}
}

func validateValue(option *Option, value interface{}) (*valueCache, error) { //nolint:gocyclo
	switch v := value.(type) {
	case string:
		if option.OptType != OptTypeString {
			return nil, newInvalidValueError(option.Key, fmt.Sprintf("%T", v), "expected type string")
		}
		if option.compiledRegex != nil {
			if !option.compiledRegex.MatchString(v) {
				return nil, newInvalidValueError(option.Key, v, "validation regex failed")
			}
		}
		return &valueCache{stringVal: v}, nil
	case []interface{}:
		vConverted := make([]string, len(v))
		for pos, entry := range v {
			s, ok := entry.(string)
			if !ok {
				return nil, newInvalidValueError(option.Key, fmt.Sprintf("element %+v ad index %d", entry, pos), "not a string")
			}
			vConverted[pos] = s
		}
		// continue to next case
		return validateValue(option, vConverted)
	case []string:
		if option.OptType != OptTypeStringArray {
			return nil, newInvalidValueError(option.Key, fmt.Sprintf("%T", v), "expected type []string")
		}
		if option.compiledRegex != nil {
			for pos, entry := range v {
				if !option.compiledRegex.MatchString(entry) {
					return nil, newInvalidValueError(option.Key, fmt.Sprintf("element %s ad index %d", entry, pos), "validation regex failed")
				}
			}
		}
		return &valueCache{stringArrayVal: v}, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, float32, float64:
		// uint64 is omitted, as it does not fit in a int64
		if option.OptType != OptTypeInt {
			return nil, newInvalidValueError(option.Key, fmt.Sprintf("%T", v), "expected type int")
		}
		if option.compiledRegex != nil {
			// we need to use %v here so we handle float and int correctly.
			if !option.compiledRegex.MatchString(fmt.Sprintf("%v", v)) {
				return nil, newInvalidValueError(option.Key, v, "validation regex failed")
			}
		}
		switch v := value.(type) {
		case int:
			return &valueCache{intVal: int64(v)}, nil
		case int8:
			return &valueCache{intVal: int64(v)}, nil
		case int16:
			return &valueCache{intVal: int64(v)}, nil
		case int32:
			return &valueCache{intVal: int64(v)}, nil
		case int64:
			return &valueCache{intVal: v}, nil
		case uint:
			return &valueCache{intVal: int64(v)}, nil
		case uint8:
			return &valueCache{intVal: int64(v)}, nil
		case uint16:
			return &valueCache{intVal: int64(v)}, nil
		case uint32:
			return &valueCache{intVal: int64(v)}, nil
		case float32:
			// convert if float has no decimals
			if math.Remainder(float64(v), 1) == 0 {
				return &valueCache{intVal: int64(v)}, nil
			}
			return nil, newInvalidValueError(option.Key, v, "failed to convert float32 to int64")
		case float64:
			// convert if float has no decimals
			if math.Remainder(v, 1) == 0 {
				return &valueCache{intVal: int64(v)}, nil
			}
			return nil, newInvalidValueError(option.Key, v, "failed to convert float64 to int64")
		default:
			return nil, ErrUnsupportedType
		}
	case bool:
		if option.OptType != OptTypeBool {
			return nil, newInvalidValueError(option.Key, fmt.Sprintf("%T", v), "expected type bool")
		}
		return &valueCache{boolVal: v}, nil
	default:
		return nil, newInvalidValueError(option.Key, fmt.Sprintf("%T", v), "invalid value")
	}
}
