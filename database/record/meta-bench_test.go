package record

// Benchmark:
// BenchmarkAllocateBytes-8                	2000000000	         0.76 ns/op
// BenchmarkAllocateStruct1-8              	2000000000	         0.76 ns/op
// BenchmarkAllocateStruct2-8              	2000000000	         0.79 ns/op
// BenchmarkMetaSerializeContainer-8       	 1000000	      1703 ns/op
// BenchmarkMetaUnserializeContainer-8     	 2000000	       950 ns/op
// BenchmarkMetaSerializeVarInt-8          	 3000000	       457 ns/op
// BenchmarkMetaUnserializeVarInt-8        	20000000	        62.9 ns/op
// BenchmarkMetaSerializeWithXDR2-8        	 1000000	      2360 ns/op
// BenchmarkMetaUnserializeWithXDR2-8      	  500000	      3189 ns/op
// BenchmarkMetaSerializeWithColfer-8      	10000000	       237 ns/op
// BenchmarkMetaUnserializeWithColfer-8    	20000000	        51.7 ns/op
// BenchmarkMetaSerializeWithCodegen-8     	50000000	        23.7 ns/op
// BenchmarkMetaUnserializeWithCodegen-8   	100000000	        18.9 ns/op
// BenchmarkMetaSerializeWithDSDJSON-8     	 1000000	      2398 ns/op
// BenchmarkMetaUnserializeWithDSDJSON-8   	  300000	      6264 ns/op

import (
	"testing"
	"time"

	"github.com/Safing/portbase/container"
	"github.com/Safing/portbase/formats/dsd"
	"github.com/Safing/portbase/formats/varint"
	// Colfer
	// "github.com/Safing/portbase/database/model/model"
	// XDR
	// xdr2 "github.com/davecgh/go-xdr/xdr2"
)

var (
	testMeta = &Meta{
		created:   time.Now().Unix(),
		modified:  time.Now().Unix(),
		expires:   time.Now().Unix(),
		deleted:   time.Now().Unix(),
		secret:    true,
		cronjewel: true,
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

func BenchmarkMetaSerializeContainer(b *testing.B) {

	// Start benchmark
	for i := 0; i < b.N; i++ {
		c := container.New()
		c.AppendNumber(uint64(testMeta.created))
		c.AppendNumber(uint64(testMeta.modified))
		c.AppendNumber(uint64(testMeta.expires))
		c.AppendNumber(uint64(testMeta.deleted))
		switch {
		case testMeta.secret && testMeta.cronjewel:
			c.AppendNumber(3)
		case testMeta.secret:
			c.AppendNumber(1)
		case testMeta.cronjewel:
			c.AppendNumber(2)
		default:
			c.AppendNumber(0)
		}
	}

}

func BenchmarkMetaUnserializeContainer(b *testing.B) {

	// Setup
	c := container.New()
	c.AppendNumber(uint64(testMeta.created))
	c.AppendNumber(uint64(testMeta.modified))
	c.AppendNumber(uint64(testMeta.expires))
	c.AppendNumber(uint64(testMeta.deleted))
	switch {
	case testMeta.secret && testMeta.cronjewel:
		c.AppendNumber(3)
	case testMeta.secret:
		c.AppendNumber(1)
	case testMeta.cronjewel:
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
		newMeta.created = int64(num)
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}
		num, err = c.GetNextN64()
		newMeta.modified = int64(num)
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}
		num, err = c.GetNextN64()
		newMeta.expires = int64(num)
		if err != nil {
			b.Errorf("could not decode: %s", err)
			return
		}
		num, err = c.GetNextN64()
		newMeta.deleted = int64(num)
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
			newMeta.secret = true
			newMeta.cronjewel = true
		case 2:
			newMeta.cronjewel = true
		case 1:
			newMeta.secret = true
		case 0:
		default:
			b.Errorf("invalid flag value: %d", flags)
			return
		}
	}

}

func BenchmarkMetaSerializeVarInt(b *testing.B) {

	// Start benchmark
	for i := 0; i < b.N; i++ {
		encoded := make([]byte, 33)
		offset := 0
		data := varint.Pack64(uint64(testMeta.created))
		for _, part := range data {
			encoded[offset] = part
			offset++
		}
		data = varint.Pack64(uint64(testMeta.modified))
		for _, part := range data {
			encoded[offset] = part
			offset++
		}
		data = varint.Pack64(uint64(testMeta.expires))
		for _, part := range data {
			encoded[offset] = part
			offset++
		}
		data = varint.Pack64(uint64(testMeta.deleted))
		for _, part := range data {
			encoded[offset] = part
			offset++
		}

		switch {
		case testMeta.secret && testMeta.cronjewel:
			encoded[offset] = 3
		case testMeta.secret:
			encoded[offset] = 1
		case testMeta.cronjewel:
			encoded[offset] = 2
		default:
			encoded[offset] = 0
		}
		offset++
	}

}

func BenchmarkMetaUnserializeVarInt(b *testing.B) {

	// Setup
	encoded := make([]byte, 33)
	offset := 0
	data := varint.Pack64(uint64(testMeta.created))
	for _, part := range data {
		encoded[offset] = part
		offset++
	}
	data = varint.Pack64(uint64(testMeta.modified))
	for _, part := range data {
		encoded[offset] = part
		offset++
	}
	data = varint.Pack64(uint64(testMeta.expires))
	for _, part := range data {
		encoded[offset] = part
		offset++
	}
	data = varint.Pack64(uint64(testMeta.deleted))
	for _, part := range data {
		encoded[offset] = part
		offset++
	}

	switch {
	case testMeta.secret && testMeta.cronjewel:
		encoded[offset] = 3
	case testMeta.secret:
		encoded[offset] = 1
	case testMeta.cronjewel:
		encoded[offset] = 2
	default:
		encoded[offset] = 0
	}
	offset++
	encodedData := encoded[:offset]

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		var newMeta Meta
		offset = 0

		num, n, err := varint.Unpack64(encodedData)
		if err != nil {
			b.Error(err)
			return
		}
		testMeta.created = int64(num)
		offset += n

		num, n, err = varint.Unpack64(encodedData[offset:])
		if err != nil {
			b.Error(err)
			return
		}
		testMeta.modified = int64(num)
		offset += n

		num, n, err = varint.Unpack64(encodedData[offset:])
		if err != nil {
			b.Error(err)
			return
		}
		testMeta.expires = int64(num)
		offset += n

		num, n, err = varint.Unpack64(encodedData[offset:])
		if err != nil {
			b.Error(err)
			return
		}
		testMeta.deleted = int64(num)
		offset += n

		switch encodedData[offset] {
		case 3:
			newMeta.secret = true
			newMeta.cronjewel = true
		case 2:
			newMeta.cronjewel = true
		case 1:
			newMeta.secret = true
		case 0:
		default:
			b.Errorf("invalid flag value: %d", encodedData[offset])
			return
		}
	}

}

// func BenchmarkMetaSerializeWithXDR2(b *testing.B) {
//
// 	// Setup
// 	var w bytes.Buffer
//
// 	// Reset timer for precise results
// 	b.ResetTimer()
//
// 	// Start benchmark
// 	for i := 0; i < b.N; i++ {
// 		w.Reset()
// 		_, err := xdr2.Marshal(&w, testMeta)
// 		if err != nil {
// 			b.Errorf("failed to serialize with xdr2: %s", err)
// 			return
// 		}
// 	}
//
// }

// func BenchmarkMetaUnserializeWithXDR2(b *testing.B) {
//
// 	// Setup
// 	var w bytes.Buffer
// 	_, err := xdr2.Marshal(&w, testMeta)
// 	if err != nil {
// 		b.Errorf("failed to serialize with xdr2: %s", err)
// 	}
// 	encodedData := w.Bytes()
//
// 	// Reset timer for precise results
// 	b.ResetTimer()
//
// 	// Start benchmark
// 	for i := 0; i < b.N; i++ {
// 		var newMeta Meta
// 		_, err := xdr2.Unmarshal(bytes.NewReader(encodedData), &newMeta)
// 		if err != nil {
// 			b.Errorf("failed to unserialize with xdr2: %s", err)
// 			return
// 		}
// 	}
//
// }

// func BenchmarkMetaSerializeWithColfer(b *testing.B) {
//
// 	testColf := &model.Course{
// 		Created:   time.Now().Unix(),
// 		Modified:  time.Now().Unix(),
// 		Expires:   time.Now().Unix(),
// 		Deleted:   time.Now().Unix(),
// 		Secret:    true,
// 		Cronjewel: true,
// 	}
//
// 	// Setup
// 	for i := 0; i < b.N; i++ {
// 		_, err := testColf.MarshalBinary()
// 		if err != nil {
// 			b.Errorf("failed to serialize with colfer: %s", err)
// 			return
// 		}
// 	}
//
// }

// func BenchmarkMetaUnserializeWithColfer(b *testing.B) {
//
// 	testColf := &model.Course{
// 		Created:   time.Now().Unix(),
// 		Modified:  time.Now().Unix(),
// 		Expires:   time.Now().Unix(),
// 		Deleted:   time.Now().Unix(),
// 		Secret:    true,
// 		Cronjewel: true,
// 	}
// 	encodedData, err := testColf.MarshalBinary()
// 	if err != nil {
// 		b.Errorf("failed to serialize with colfer: %s", err)
// 		return
// 	}
//
// 	// Setup
// 	for i := 0; i < b.N; i++ {
// 		var testUnColf model.Course
// 		err := testUnColf.UnmarshalBinary(encodedData)
// 		if err != nil {
// 			b.Errorf("failed to unserialize with colfer: %s", err)
// 			return
// 		}
// 	}
//
// }

func BenchmarkMetaSerializeWithCodegen(b *testing.B) {

	for i := 0; i < b.N; i++ {
		_, err := testMeta.GenCodeMarshal(nil)
		if err != nil {
			b.Errorf("failed to serialize with codegen: %s", err)
			return
		}
	}

}

func BenchmarkMetaUnserializeWithCodegen(b *testing.B) {

	// Setup
	encodedData, err := testMeta.GenCodeMarshal(nil)
	if err != nil {
		b.Errorf("failed to serialize with codegen: %s", err)
		return
	}

	// Reset timer for precise results
	b.ResetTimer()

	// Start benchmark
	for i := 0; i < b.N; i++ {
		var newMeta Meta
		_, err := newMeta.GenCodeUnmarshal(encodedData)
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
