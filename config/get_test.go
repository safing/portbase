package config

import (
	"testing"

	"github.com/safing/portbase/log"
)

func parseAndSetConfig(jsonData string) error {
	m, err := JSONToMap([]byte(jsonData))
	if err != nil {
		return err
	}

	return setConfig(m)
}

func parseAndSetDefaultConfig(jsonData string) error {
	m, err := JSONToMap([]byte(jsonData))
	if err != nil {
		return err
	}

	return SetDefaultConfig(m)
}

func quickRegister(t *testing.T, key string, optType uint8, defaultValue interface{}) {
	err := Register(&Option{
		Name:           key,
		Key:            key,
		Description:    "test config",
		ReleaseLevel:   ReleaseLevelStable,
		ExpertiseLevel: ExpertiseLevelUser,
		OptType:        optType,
		DefaultValue:   defaultValue,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGet(t *testing.T) {
	// reset
	options = make(map[string]*Option)

	err := log.Start()
	if err != nil {
		t.Fatal(err)
	}

	quickRegister(t, "monkey", OptTypeInt, -1)
	quickRegister(t, "zebras/zebra", OptTypeStringArray, []string{"a", "b"})
	quickRegister(t, "elephant", OptTypeInt, -1)
	quickRegister(t, "hot", OptTypeBool, false)
	quickRegister(t, "cold", OptTypeBool, true)

	err = parseAndSetConfig(`
  {
    "monkey": "1",
		"zebras": {
			"zebra": ["black", "white"]
		},
    "elephant": 2,
		"hot": true,
		"cold": false
  }
  `)
	if err != nil {
		t.Fatal(err)
	}

	err = parseAndSetDefaultConfig(`
  {
    "monkey": "0",
    "snake": "0",
    "elephant": 0
  }
  `)
	if err != nil {
		t.Fatal(err)
	}

	monkey := GetAsString("monkey", "none")
	if monkey() != "1" {
		t.Errorf("monkey should be 1, is %s", monkey())
	}

	zebra := GetAsStringArray("zebras/zebra", []string{})
	if len(zebra()) != 2 || zebra()[0] != "black" || zebra()[1] != "white" {
		t.Errorf("zebra should be [\"black\", \"white\"], is %v", zebra())
	}

	elephant := GetAsInt("elephant", -1)
	if elephant() != 2 {
		t.Errorf("elephant should be 2, is %d", elephant())
	}

	hot := GetAsBool("hot", false)
	if !hot() {
		t.Errorf("hot should be true, is %v", hot())
	}

	cold := GetAsBool("cold", true)
	if cold() {
		t.Errorf("cold should be false, is %v", cold())
	}

	err = parseAndSetConfig(`
  {
    "monkey": "3"
  }
  `)
	if err != nil {
		t.Fatal(err)
	}

	if monkey() != "3" {
		t.Errorf("monkey should be 0, is %s", monkey())
	}

	if elephant() != 0 {
		t.Errorf("elephant should be 0, is %d", elephant())
	}

	zebra()
	hot()

	// concurrent
	GetAsString("monkey", "none")()
	GetAsStringArray("zebras/zebra", []string{})()
	GetAsInt("elephant", -1)()
	GetAsBool("hot", false)()

}

func TestReleaseLevel(t *testing.T) {
	// reset
	options = make(map[string]*Option)
	registerReleaseLevelOption()

	// setup
	subsystemOption := &Option{
		Name:           "test subsystem",
		Key:            "subsystem/test",
		Description:    "test config",
		ReleaseLevel:   ReleaseLevelStable,
		ExpertiseLevel: ExpertiseLevelUser,
		OptType:        OptTypeBool,
		DefaultValue:   false,
	}
	err := Register(subsystemOption)
	if err != nil {
		t.Fatal(err)
	}
	err = SetConfigOption("subsystem/test", true)
	if err != nil {
		t.Fatal(err)
	}
	testSubsystem := GetAsBool("subsystem/test", false)

	// test option level stable
	subsystemOption.ReleaseLevel = ReleaseLevelStable
	err = SetConfigOption(releaseLevelKey, ReleaseLevelStable)
	if err != nil {
		t.Fatal(err)
	}
	if !testSubsystem() {
		t.Error("should be active")
	}
	err = SetConfigOption(releaseLevelKey, ReleaseLevelBeta)
	if err != nil {
		t.Fatal(err)
	}
	if !testSubsystem() {
		t.Error("should be active")
	}
	err = SetConfigOption(releaseLevelKey, ReleaseLevelExperimental)
	if err != nil {
		t.Fatal(err)
	}
	if !testSubsystem() {
		t.Error("should be active")
	}

	// test option level beta
	subsystemOption.ReleaseLevel = ReleaseLevelBeta
	err = SetConfigOption(releaseLevelKey, ReleaseLevelStable)
	if err != nil {
		t.Fatal(err)
	}
	if testSubsystem() {
		t.Errorf("should be inactive: opt=%s system=%s", subsystemOption.ReleaseLevel, releaseLevel)
	}
	err = SetConfigOption(releaseLevelKey, ReleaseLevelBeta)
	if err != nil {
		t.Fatal(err)
	}
	if !testSubsystem() {
		t.Error("should be active")
	}
	err = SetConfigOption(releaseLevelKey, ReleaseLevelExperimental)
	if err != nil {
		t.Fatal(err)
	}
	if !testSubsystem() {
		t.Error("should be active")
	}

	// test option level experimental
	subsystemOption.ReleaseLevel = ReleaseLevelExperimental
	err = SetConfigOption(releaseLevelKey, ReleaseLevelStable)
	if err != nil {
		t.Fatal(err)
	}
	if testSubsystem() {
		t.Error("should be inactive")
	}
	err = SetConfigOption(releaseLevelKey, ReleaseLevelBeta)
	if err != nil {
		t.Fatal(err)
	}
	if testSubsystem() {
		t.Error("should be inactive")
	}
	err = SetConfigOption(releaseLevelKey, ReleaseLevelExperimental)
	if err != nil {
		t.Fatal(err)
	}
	if !testSubsystem() {
		t.Error("should be active")
	}
}

func BenchmarkGetAsStringCached(b *testing.B) {
	// reset
	options = make(map[string]*Option)

	// Setup
	err := parseAndSetConfig(`
  {
    "monkey": "banana"
  }
  `)
	if err != nil {
		b.Fatal(err)
	}
	monkey := GetAsString("monkey", "no banana")

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		monkey()
	}
}

func BenchmarkGetAsStringRefetch(b *testing.B) {
	// Setup
	err := parseAndSetConfig(`
  {
    "monkey": "banana"
  }
  `)
	if err != nil {
		b.Fatal(err)
	}

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		findStringValue("monkey", "no banana")
	}
}

func BenchmarkGetAsIntCached(b *testing.B) {
	// Setup
	err := parseAndSetConfig(`
  {
    "monkey": 1
  }
  `)
	if err != nil {
		b.Fatal(err)
	}
	monkey := GetAsInt("monkey", -1)

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		monkey()
	}
}

func BenchmarkGetAsIntRefetch(b *testing.B) {
	// Setup
	err := parseAndSetConfig(`
  {
    "monkey": 1
  }
  `)
	if err != nil {
		b.Fatal(err)
	}

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		findIntValue("monkey", 1)
	}
}
