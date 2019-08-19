package record

import (
	"bytes"
	"errors"
	"testing"

	"github.com/safing/portbase/container"
)

func TestWrapper(t *testing.T) {

	// check model interface compliance
	var m Record
	w := &Wrapper{}
	m = w
	_ = m

	// create test data
	testData := []byte(`{"a": "b"}`)
	encodedTestData := []byte(`J{"a": "b"}`)

	// test wrapper
	wrapper, err := NewWrapper("test:a", &Meta{}, JSON, testData)
	if err != nil {
		t.Fatal(err)
	}
	if wrapper.Format != JSON {
		t.Error("format mismatch")
	}
	if !bytes.Equal(testData, wrapper.Data) {
		t.Error("data mismatch")
	}

	encoded, err := wrapper.Marshal(wrapper, JSON)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encodedTestData, encoded) {
		t.Error("marshal mismatch")
	}

	wrapper.SetMeta(&Meta{})
	wrapper.meta.Update()
	raw, err := wrapper.MarshalRecord(wrapper)
	if err != nil {
		t.Fatal(err)
	}

	wrapper2, err := NewRawWrapper("test", "a", raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(testData, wrapper2.Data) {
		t.Error("marshal mismatch")
	}

	// test new format
	oldRaw, err := oldWrapperMarshalRecord(wrapper, wrapper)
	if err != nil {
		t.Fatal(err)
	}

	wrapper3, err := NewRawWrapper("test", "a", oldRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(testData, wrapper3.Data) {
		t.Error("marshal mismatch")
	}
}

func oldWrapperMarshalRecord(w *Wrapper, r Record) ([]byte, error) {
	if w.Meta() == nil {
		return nil, errors.New("missing meta")
	}

	// version
	c := container.New([]byte{1})

	// meta
	metaSection, err := w.meta.GenCodeMarshal(nil)
	if err != nil {
		return nil, err
	}
	c.AppendAsBlock(metaSection)

	// data
	dataSection, err := w.Marshal(r, JSON)
	if err != nil {
		return nil, err
	}
	c.Append(dataSection)

	return c.CompileData(), nil
}
