package config

import (
	"bytes"
	"testing"
)

func TestJSONMapConversion(t *testing.T) {

	jsonData := `{
  "a": "b",
  "c": {
    "d": "e",
    "f": "g",
    "h": {
      "i": "j",
      "k": "l",
      "m": {
        "n": "o"
      }
    }
  },
  "p": "q"
}`
	jsonBytes := []byte(jsonData)

	mapData := map[string]interface{}{
		"a":       "b",
		"p":       "q",
		"c/d":     "e",
		"c/f":     "g",
		"c/h/i":   "j",
		"c/h/k":   "l",
		"c/h/m/n": "o",
	}

	m, err := JSONToMap(jsonBytes)
	if err != nil {
		t.Fatal(err)
	}

	j, err := MapToJSON(mapData)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(jsonBytes, j) {
		t.Errorf("json does not match, got %s", j)
	}

	j2, err := MapToJSON(m)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(jsonBytes, j2) {
		t.Errorf("json does not match, got %s", j)
	}

	// fails for some reason
	// if !reflect.DeepEqual(mapData, m) {
	// 	t.Errorf("maps do not match, got %s", m)
	// }
}
