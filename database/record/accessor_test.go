package record

import (
	"encoding/json"
	"testing"
)

type TestStruct struct {
	S    string
	I    int
	I8   int8
	I16  int16
	I32  int32
	I64  int64
	UI   uint
	UI8  uint8
	UI16 uint16
	UI32 uint32
	UI64 uint64
	F32  float32
	F64  float64
	B    bool
}

var (
	testStruct = &TestStruct{
		S:    "banana",
		I:    42,
		I8:   42,
		I16:  42,
		I32:  42,
		I64:  42,
		UI:   42,
		UI8:  42,
		UI16: 42,
		UI32: 42,
		UI64: 42,
		F32:  42.42,
		F64:  42.42,
		B:    true,
	}
	testJSONBytes, _ = json.Marshal(testStruct)
	testJSON         = string(testJSONBytes)
)

func testGetString(t *testing.T, acc Accessor, key string, shouldSucceed bool, expectedValue string) {
	v, ok := acc.GetString(key)
	switch {
	case !ok && shouldSucceed:
		t.Errorf("%s failed to get string with key %s", acc.Type(), key)
	case ok && !shouldSucceed:
		t.Errorf("%s should have failed to get string with key %s, it returned %v", acc.Type(), key, v)
	}
	if v != expectedValue {
		t.Errorf("%s returned an unexpected value: wanted %v, got %v", acc.Type(), expectedValue, v)
	}
}

func testGetInt(t *testing.T, acc Accessor, key string, shouldSucceed bool, expectedValue int64) {
	v, ok := acc.GetInt(key)
	switch {
	case !ok && shouldSucceed:
		t.Errorf("%s failed to get int with key %s", acc.Type(), key)
	case ok && !shouldSucceed:
		t.Errorf("%s should have failed to get int with key %s, it returned %v", acc.Type(), key, v)
	}
	if v != expectedValue {
		t.Errorf("%s returned an unexpected value: wanted %v, got %v", acc.Type(), expectedValue, v)
	}
}

func testGetFloat(t *testing.T, acc Accessor, key string, shouldSucceed bool, expectedValue float64) {
	v, ok := acc.GetFloat(key)
	switch {
	case !ok && shouldSucceed:
		t.Errorf("%s failed to get float with key %s", acc.Type(), key)
	case ok && !shouldSucceed:
		t.Errorf("%s should have failed to get float with key %s, it returned %v", acc.Type(), key, v)
	}
	if int64(v) != int64(expectedValue) {
		t.Errorf("%s returned an unexpected value: wanted %v, got %v", acc.Type(), expectedValue, v)
	}
}

func testGetBool(t *testing.T, acc Accessor, key string, shouldSucceed bool, expectedValue bool) {
	v, ok := acc.GetBool(key)
	switch {
	case !ok && shouldSucceed:
		t.Errorf("%s failed to get bool with key %s", acc.Type(), key)
	case ok && !shouldSucceed:
		t.Errorf("%s should have failed to get bool with key %s, it returned %v", acc.Type(), key, v)
	}
	if v != expectedValue {
		t.Errorf("%s returned an unexpected value: wanted %v, got %v", acc.Type(), expectedValue, v)
	}
}

func testSet(t *testing.T, acc Accessor, key string, shouldSucceed bool, valueToSet interface{}) {
	err := acc.Set(key, valueToSet)
	switch {
	case err != nil && shouldSucceed:
		t.Errorf("%s failed to set %s to %+v: %s", acc.Type(), key, valueToSet, err)
	case err == nil && !shouldSucceed:
		t.Errorf("%s should have failed to set %s to %+v", acc.Type(), key, valueToSet)
	}
}

func TestAccessor(t *testing.T) {

	// Test interface compliance
	accs := []Accessor{
		NewJSONAccessor(&testJSON),
		NewJSONBytesAccessor(&testJSONBytes),
		NewStructAccessor(testStruct),
	}

	// get
	for _, acc := range accs {
		testGetString(t, acc, "S", true, "banana")
		testGetInt(t, acc, "I", true, 42)
		testGetInt(t, acc, "I8", true, 42)
		testGetInt(t, acc, "I16", true, 42)
		testGetInt(t, acc, "I32", true, 42)
		testGetInt(t, acc, "I64", true, 42)
		testGetInt(t, acc, "UI", true, 42)
		testGetInt(t, acc, "UI8", true, 42)
		testGetInt(t, acc, "UI16", true, 42)
		testGetInt(t, acc, "UI32", true, 42)
		testGetInt(t, acc, "UI64", true, 42)
		testGetFloat(t, acc, "F32", true, 42.42)
		testGetFloat(t, acc, "F64", true, 42.42)
		testGetBool(t, acc, "B", true, true)
	}

	// set
	for _, acc := range accs {
		testSet(t, acc, "S", true, "coconut")
		testSet(t, acc, "I", true, uint32(44))
		testSet(t, acc, "I8", true, uint64(44))
		testSet(t, acc, "I16", true, uint8(44))
		testSet(t, acc, "I32", true, uint16(44))
		testSet(t, acc, "I64", true, 44)
		testSet(t, acc, "UI", true, 44)
		testSet(t, acc, "UI8", true, int64(44))
		testSet(t, acc, "UI16", true, int32(44))
		testSet(t, acc, "UI32", true, int8(44))
		testSet(t, acc, "UI64", true, int16(44))
		testSet(t, acc, "F32", true, 44.44)
		testSet(t, acc, "F64", true, 44.44)
		testSet(t, acc, "B", true, false)
	}

	// get again
	for _, acc := range accs {
		testGetString(t, acc, "S", true, "coconut")
		testGetInt(t, acc, "I", true, 44)
		testGetInt(t, acc, "I8", true, 44)
		testGetInt(t, acc, "I16", true, 44)
		testGetInt(t, acc, "I32", true, 44)
		testGetInt(t, acc, "I64", true, 44)
		testGetInt(t, acc, "UI", true, 44)
		testGetInt(t, acc, "UI8", true, 44)
		testGetInt(t, acc, "UI16", true, 44)
		testGetInt(t, acc, "UI32", true, 44)
		testGetInt(t, acc, "UI64", true, 44)
		testGetFloat(t, acc, "F32", true, 44.44)
		testGetFloat(t, acc, "F64", true, 44.44)
		testGetBool(t, acc, "B", true, false)
	}

	// failures
	for _, acc := range accs {
		testGetString(t, acc, "S", false, 1)
		testGetInt(t, acc, "I", false, 44)
		testGetInt(t, acc, "I8", false, 512)
		testGetInt(t, acc, "I16", false, 1000000)
		testGetInt(t, acc, "I32", false, 44)
		testGetInt(t, acc, "I64", false, "44")
		testGetInt(t, acc, "UI", false, 44)
		testGetInt(t, acc, "UI8", false, 44)
		testGetInt(t, acc, "UI16", false, 44)
		testGetInt(t, acc, "UI32", false, 44)
		testGetInt(t, acc, "UI64", false, 44)
		testGetFloat(t, acc, "F32", false, 44.44)
		testGetFloat(t, acc, "F64", false, 44.44)
		testGetBool(t, acc, "B", false, false)
	}
}
