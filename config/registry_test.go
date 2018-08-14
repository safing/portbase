package config

import (
	"testing"
)

func TestRegistry(t *testing.T) {

	if err := Register(&Option{
		Name:            "name",
		Key:             "key",
		Description:     "description",
		ExpertiseLevel:  1,
		OptType:         OptTypeString,
		DefaultValue:    "default",
		ValidationRegex: "^(banana|water)$",
	}); err != nil {
		t.Error(err)
	}

	if err := Register(&Option{
		Name:            "name",
		Key:             "key",
		Description:     "description",
		ExpertiseLevel:  1,
		OptType:         0,
		DefaultValue:    "default",
		ValidationRegex: "^[A-Z][a-z]+$",
	}); err == nil {
		t.Error("should fail")
	}

	if err := Register(&Option{
		Name:            "name",
		Key:             "key",
		Description:     "description",
		ExpertiseLevel:  1,
		OptType:         OptTypeString,
		DefaultValue:    "default",
		ValidationRegex: "[",
	}); err == nil {
		t.Error("should fail")
	}

}
