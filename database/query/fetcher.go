package query

import (
	"github.com/tidwall/gjson"
)

const (
	emptyString = ""
)

// Fetcher provides an interface to supply the query matcher a method to retrieve values from an object.
type Fetcher interface {
	GetString(key string) (value string, ok bool)
	GetInt(key string) (value int64, ok bool)
	GetFloat(key string) (value float64, ok bool)
	GetBool(key string) (value bool, ok bool)
	Exists(key string) bool
}

// JSONFetcher is a json string with get functions.
type JSONFetcher struct {
	json string
}

// NewJSONFetcher adds the Fetcher interface to a JSON string.
func NewJSONFetcher(json string) *JSONFetcher {
	return &JSONFetcher{
		json: json,
	}
}

// GetString returns the string found by the given json key and whether it could be successfully extracted.
func (jf *JSONFetcher) GetString(key string) (value string, ok bool) {
	result := gjson.Get(jf.json, key)
	if !result.Exists() || result.Type != gjson.String {
		return emptyString, false
	}
	return result.String(), true
}

// GetInt returns the int found by the given json key and whether it could be successfully extracted.
func (jf *JSONFetcher) GetInt(key string) (value int64, ok bool) {
	result := gjson.Get(jf.json, key)
	if !result.Exists() || result.Type != gjson.Number {
		return 0, false
	}
	return result.Int(), true
}

// GetFloat returns the float found by the given json key and whether it could be successfully extracted.
func (jf *JSONFetcher) GetFloat(key string) (value float64, ok bool) {
	result := gjson.Get(jf.json, key)
	if !result.Exists() || result.Type != gjson.Number {
		return 0, false
	}
	return result.Float(), true
}

// GetBool returns the bool found by the given json key and whether it could be successfully extracted.
func (jf *JSONFetcher) GetBool(key string) (value bool, ok bool) {
	result := gjson.Get(jf.json, key)
	switch {
	case !result.Exists():
		return false, false
	case result.Type == gjson.True:
		return true, true
	case result.Type == gjson.False:
		return false, true
	default:
		return false, false
	}
}

// Exists returns the whether the given key exists.
func (jf *JSONFetcher) Exists(key string) bool {
	result := gjson.Get(jf.json, key)
	return result.Exists()
}
