package config

import "testing"

func TestLayers(t *testing.T) {

	err := SetConfig("{invalid json")
	if err == nil {
		t.Fatal("expected error")
	}

	err = SetDefaultConfig("{invalid json")
	if err == nil {
		t.Fatal("expected error")
	}

	err = SetConfig(`
    {
      "monkey": "banana",
      "elephant": 3
    }
    `)
	if err != nil {
		t.Fatal(err)
	}

	// Test missing values

	missingString := GetAsString("missing", "fallback")
	if missingString() != "fallback" {
		t.Fatal("expected fallback value: fallback")
	}

	missingInt := GetAsInt("missing", -1)
	if missingInt() != -1 {
		t.Fatal("expected fallback value: -1")
	}

	// Test value mismatch

	notString := GetAsString("elephant", "fallback")
	if notString() != "fallback" {
		t.Fatal("expected fallback value: fallback")
	}

	notInt := GetAsInt("monkey", -1)
	if notInt() != -1 {
		t.Fatal("expected fallback value: -1")
	}

}
