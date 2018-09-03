package model

import (
	"bytes"
	"testing"
	"time"

	"github.com/Safing/portbase/container"
	"github.com/Safing/portbase/database/model/model"
	"github.com/Safing/portbase/formats/dsd"
	xdr2 "github.com/davecgh/go-xdr/xdr2"
)

var (
	testMeta = &Meta{
		Created:   time.Now().Unix(),
		Modified:  time.Now().Unix(),
		Expires:   time.Now().Unix(),
		Deleted:   time.Now().Unix(),
		Secret:    true,
		Cronjewel: true,
	}
)

func BenchmarkAllocateBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = make([]byte, 33)
	}
}

func BenchmarkAllocateStruct1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var new Meta
		_ = new
	}
}

func BenchmarkAllocateStruct2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Meta{}
	}
}

func BenchmarkMetaSerializeCustom(b *testing.B) {

	// Start benchmark
	for i := 0; i < b.N; i++ {
		c := container.New()
		c.AppendNumber(uint64(testMeta.Created))
		c.AppendNumber(uint64(testMeta.Modified))
		c.AppendNumber(uint64(testMeta.Expires))
		c.AppendNumber(uint64(testMeta.Deleted))
		switch {
		case testMeta.Secret && testMeta.Cronjewel:
			c.AppendNumber(3)
		case testMeta.Secret:
			c.AppendNumber(1)
		case testMeta.Cronjewel:
			c.AppendNumber(2)
		default:
			c.AppendNumber(0)
		}
	}

}

func BenchmarkMetaUnserializeCustom(b *testing.B) {

	// Setup
	c := container.New()
	c.AppendNumber(uint64(testMeta.Created))
	c.AppendNumber(uint64(testMeta.Modified))
	c.AppendNumber(uint64(testMeta.Expires))
	c.AppendNumber(uint64(testMeta.Deleted))
	switch {
	case testMeta.Secret && testMeta.Cronjewel:
		c.AppendNumber(3)
	case testMeta.Secret:
		c.AppendNumber(1)
	case testMeta.Cronjewel:
		c.AppendNumber(2)
	default:
		c.AppendNumber(0)
	}
	encodedData := c.CompileData()

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		var newMeta Meta
		var err error
		var num uint64
		c := container.New(encodedData)
		num, err = c.GetNextN64()
		newMeta.Created = int64(num)
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}
		num, err = c.GetNextN64()
		newMeta.Modified = int64(num)
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}
		num, err = c.GetNextN64()
		newMeta.Expires = int64(num)
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}
		num, err = c.GetNextN64()
		newMeta.Deleted = int64(num)
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}

		flags, err := c.GetNextN8()
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}

		switch flags {
		case 3:
			newMeta.Secret = true
			newMeta.Cronjewel = true
		case 2:
			newMeta.Cronjewel = true
		case 1:
			newMeta.Secret = true
		case 0:
		default:
			b.Errorf("invalid flag value: %d", flags)
			return
		}
	}

}

func BenchmarkMetaSerializeWithXDR2(b *testing.B) {

	// Setup
	var w bytes.Buffer

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		w.Reset()
		_, err := xdr2.Marshal(&w, testMeta)
		if err != nil {
			b.Errorf("failed to serialize with xdr2: %s", err)
			return
		}
	}

}

func BenchmarkMetaUnserializeWithXDR2(b *testing.B) {

	// Setup
	var w bytes.Buffer
	_, err := xdr2.Marshal(&w, testMeta)
	if err != nil {
		b.Errorf("failed to serialize with xdr2: %s", err)
	}
	encodedData := w.Bytes()

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		var newMeta Meta
		_, err := xdr2.Unmarshal(bytes.NewReader(encodedData), &newMeta)
		if err != nil {
			b.Errorf("failed to unserialize with xdr2: %s", err)
			return
		}
	}

}

func BenchmarkMetaSerializeWithColfer(b *testing.B) {

	testColf := &model.Course{
		Created:   time.Now().Unix(),
		Modified:  time.Now().Unix(),
		Expires:   time.Now().Unix(),
		Deleted:   time.Now().Unix(),
		Secret:    true,
		Cronjewel: true,
	}

	// Setup
	for i := 0; i < b.N; i++ {
		_, err := testColf.MarshalBinary()
		if err != nil {
			b.Errorf("failed to serialize with colfer: %s", err)
			return
		}
	}

}

func BenchmarkMetaUnserializeWithColfer(b *testing.B) {

	testColf := &model.Course{
		Created:   time.Now().Unix(),
		Modified:  time.Now().Unix(),
		Expires:   time.Now().Unix(),
		Deleted:   time.Now().Unix(),
		Secret:    true,
		Cronjewel: true,
	}
	encodedData, err := testColf.MarshalBinary()
	if err != nil {
		b.Errorf("failed to serialize with colfer: %s", err)
		return
	}

	// Setup
	for i := 0; i < b.N; i++ {
		var testUnColf model.Course
		err := testUnColf.UnmarshalBinary(encodedData)
		if err != nil {
			b.Errorf("failed to unserialize with colfer: %s", err)
			return
		}
	}

}

func BenchmarkMetaSerializeWithCodegen(b *testing.B) {

	for i := 0; i < b.N; i++ {
		buf := make([]byte, 34)
		_, err := testMeta.Marshal(buf)
		if err != nil {
			b.Errorf("failed to serialize with codegen: %s", err)
			return
		}
	}

}

func BenchmarkMetaUnserializeWithCodegen(b *testing.B) {

	// Setup
	buf := make([]byte, 34)
	encodedData, err := testMeta.Marshal(buf)
	if err != nil {
		b.Errorf("failed to serialize with codegen: %s", err)
		return
	}

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		var newMeta Meta
		_, err := newMeta.Unmarshal(encodedData)
		if err != nil {
			b.Errorf("failed to unserialize with codegen: %s", err)
			return
		}
	}

}

func BenchmarkMetaSerializeWithDSDJSON(b *testing.B) {

	for i := 0; i < b.N; i++ {
		_, err := dsd.Dump(testMeta, dsd.JSON)
		if err != nil {
			b.Errorf("failed to serialize with DSD/JSON: %s", err)
			return
		}
	}

}

func BenchmarkMetaUnserializeWithDSDJSON(b *testing.B) {

	// Setup
	encodedData, err := dsd.Dump(testMeta, dsd.JSON)
	if err != nil {
		b.Errorf("failed to serialize with DSD/JSON: %s", err)
		return
	}

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		var newMeta Meta
		_, err := dsd.Load(encodedData, &newMeta)
		if err != nil {
			b.Errorf("failed to unserialize with DSD/JSON: %s", err)
			return
		}
	}

}
