//nolint:goconst,errcheck
package config

import "testing"

func TestLayersGetters(t *testing.T) {

	mapData, err := JSONToMap([]byte(`
		{
			"monkey": "1",
			"elephant": 2,
			"zebras": {
				"zebra": ["black", "white"],
				"weird_zebra": ["black", -1]
			},
			"env": {
				"hot": true
			}
		}
    `))
	if err != nil {
		t.Fatal(err)
	}

	err = setConfig(mapData)
	if err != nil {
		t.Fatal(err)
	}

	// Test missing values

	missingString := GetAsString("missing", "fallback")
	if missingString() != "fallback" {
		t.Error("expected fallback value: fallback")
	}

	missingStringArray := GetAsStringArray("missing", []string{"fallback"})
	if len(missingStringArray()) != 1 || missingStringArray()[0] != "fallback" {
		t.Error("expected fallback value: [fallback]")
	}

	missingInt := GetAsInt("missing", -1)
	if missingInt() != -1 {
		t.Error("expected fallback value: -1")
	}

	missingBool := GetAsBool("missing", false)
	if missingBool() {
		t.Error("expected fallback value: false")
	}

	// Test value mismatch

	notString := GetAsString("elephant", "fallback")
	if notString() != "fallback" {
		t.Error("expected fallback value: fallback")
	}

	notStringArray := GetAsStringArray("elephant", []string{"fallback"})
	if len(notStringArray()) != 1 || notStringArray()[0] != "fallback" {
		t.Error("expected fallback value: [fallback]")
	}

	mixedStringArray := GetAsStringArray("zebras/weird_zebra", []string{"fallback"})
	if len(mixedStringArray()) != 1 || mixedStringArray()[0] != "fallback" {
		t.Error("expected fallback value: [fallback]")
	}

	notInt := GetAsInt("monkey", -1)
	if notInt() != -1 {
		t.Error("expected fallback value: -1")
	}

	notBool := GetAsBool("monkey", false)
	if notBool() {
		t.Error("expected fallback value: false")
	}

}

func TestLayersSetters(t *testing.T) {

	Register(&Option{
		Name:            "name",
		Key:             "monkey",
		Description:     "description",
		ExpertiseLevel:  1,
		OptType:         OptTypeString,
		DefaultValue:    "banana",
		ValidationRegex: "^(banana|water)$",
	})
	Register(&Option{
		Name:            "name",
		Key:             "zebras/zebra",
		Description:     "description",
		ExpertiseLevel:  1,
		OptType:         OptTypeStringArray,
		DefaultValue:    []string{"black", "white"},
		ValidationRegex: "^[a-z]+$",
	})
	Register(&Option{
		Name:            "name",
		Key:             "elephant",
		Description:     "description",
		ExpertiseLevel:  1,
		OptType:         OptTypeInt,
		DefaultValue:    2,
		ValidationRegex: "",
	})
	Register(&Option{
		Name:            "name",
		Key:             "hot",
		Description:     "description",
		ExpertiseLevel:  1,
		OptType:         OptTypeBool,
		DefaultValue:    true,
		ValidationRegex: "",
	})

	// correct types
	if err := SetConfigOption("monkey", "banana"); err != nil {
		t.Error(err)
	}
	if err := SetConfigOption("zebras/zebra", []string{"black", "white"}); err != nil {
		t.Error(err)
	}
	if err := SetDefaultConfigOption("elephant", 2); err != nil {
		t.Error(err)
	}
	if err := SetDefaultConfigOption("hot", true); err != nil {
		t.Error(err)
	}

	// incorrect types
	if err := SetConfigOption("monkey", []string{"black", "white"}); err == nil {
		t.Error("should fail")
	}
	if err := SetConfigOption("zebras/zebra", 2); err == nil {
		t.Error("should fail")
	}
	if err := SetDefaultConfigOption("elephant", true); err == nil {
		t.Error("should fail")
	}
	if err := SetDefaultConfigOption("hot", "banana"); err == nil {
		t.Error("should fail")
	}
	if err := SetDefaultConfigOption("hot", []byte{0}); err == nil {
		t.Error("should fail")
	}

	// validation fail
	if err := SetConfigOption("monkey", "dirt"); err == nil {
		t.Error("should fail")
	}
	if err := SetConfigOption("zebras/zebra", []string{"Element649"}); err == nil {
		t.Error("should fail")
	}

	// unregistered checking
	if err := SetConfigOption("invalid", "banana"); err == nil {
		t.Error("should fail")
	}
	if err := SetConfigOption("invalid", []string{"black", "white"}); err == nil {
		t.Error("should fail")
	}
	if err := SetConfigOption("invalid", 2); err == nil {
		t.Error("should fail")
	}
	if err := SetConfigOption("invalid", true); err == nil {
		t.Error("should fail")
	}
	if err := SetConfigOption("invalid", []byte{0}); err == nil {
		t.Error("should fail")
	}

	// delete
	if err := SetConfigOption("monkey", nil); err != nil {
		t.Error(err)
	}
	if err := SetDefaultConfigOption("elephant", nil); err != nil {
		t.Error(err)
	}
	if err := SetDefaultConfigOption("invalid_delete", nil); err != nil {
		t.Error(err)
	}

}
