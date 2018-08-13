package container

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Safing/safing-core/utils"
)

var (
	testData         = []byte("The quick brown fox jumps over the lazy dog")
	testDataSplitted = [][]byte{
		[]byte("T"),
		[]byte("he"),
		[]byte(" qu"),
		[]byte("ick "),
		[]byte("brown"),
		[]byte(" fox j"),
		[]byte("umps ov"),
		[]byte("er the l"),
		[]byte("azy dog"),
	}
)

func TestContainerDataHandling(t *testing.T) {

	c1 := NewContainer(utils.DuplicateBytes(testData))
	c1c := c1.carbonCopy()

	c2 := NewContainer()
	for i := 0; i < len(testData); i++ {
		oneByte := make([]byte, 1)
		c1c.WriteToSlice(oneByte)
		c2.Append(oneByte)
	}
	c2c := c2.carbonCopy()

	c3 := NewContainer()
	for i := len(c2c.compartments) - 1; i >= c2c.offset; i-- {
		c3.Prepend(c2c.compartments[i])
	}
	c3c := c3.carbonCopy()

	d4 := make([]byte, len(testData)*2)
	n, _ := c3c.WriteToSlice(d4)
	d4 = d4[:n]
	c3c = c3.carbonCopy()

	d5 := make([]byte, len(testData))
	for i := 0; i < len(testData); i++ {
		c3c.WriteToSlice(d5[i : i+1])
	}

	c6 := NewContainer()
	c6.Replace(testData)

	c7 := NewContainer(testDataSplitted[0])
	for i := 1; i < len(testDataSplitted); i++ {
		c7.Append(testDataSplitted[i])
	}

	c8 := NewContainer(testDataSplitted...)
	for i := 0; i < 110; i++ {
		c8.Prepend(nil)
	}
	c8.Clean()

	compareMany(t, testData, c1.CompileData(), c2.CompileData(), c3.CompileData(), d4, d5, c6.CompileData(), c7.CompileData(), c8.CompileData())
}

func compareMany(t *testing.T, reference []byte, other ...[]byte) {
	for i, cmp := range other {
		if !bytes.Equal(reference, cmp) {
			t.Errorf("sample %d does not match reference: sample is '%s'", i+1, string(cmp))
		}
	}
}

func TestContainerErrorHandling(t *testing.T) {

	c1 := NewContainer(nil)

	if c1.HasError() {
		t.Error("should not have error")
	}

	c1.SetError(errors.New("test error"))

	if !c1.HasError() {
		t.Error("should have error")
	}

	c2 := NewContainer(append([]byte{0}, []byte("test error")...))

	if c2.HasError() {
		t.Error("should not have error")
	}

	c2.CheckError()

	if !c2.HasError() {
		t.Error("should have error")
	}

	if c2.Error().Error() != "test error" {
		t.Errorf("error message mismatch, was %s", c2.Error())
	}

}

func TestContainerBlockHandling(t *testing.T) {

	c1 := NewContainer(utils.DuplicateBytes(testData))
	c1.PrependLength()
	c1.AppendAsBlock(testData)
	c1c := c1.carbonCopy()

	c2 := NewContainer(nil)
	for i := 0; i < c1.Length(); i++ {
		oneByte := make([]byte, 1)
		c1c.WriteToSlice(oneByte)
		c2.Append(oneByte)
	}

	c3 := NewContainer(testDataSplitted[0])
	for i := 1; i < len(testDataSplitted); i++ {
		c3.Append(testDataSplitted[i])
	}
	c3.PrependLength()

	d1, err := c1.GetNextBlock()
	if err != nil {
		t.Errorf("GetNextBlock failed: %s", err)
	}
	d2, err := c1.GetNextBlock()
	if err != nil {
		t.Errorf("GetNextBlock failed: %s", err)
	}
	d3, err := c2.GetNextBlock()
	if err != nil {
		t.Errorf("GetNextBlock failed: %s", err)
	}
	d4, err := c2.GetNextBlock()
	if err != nil {
		t.Errorf("GetNextBlock failed: %s", err)
	}
	d5, err := c3.GetNextBlock()
	if err != nil {
		t.Errorf("GetNextBlock failed: %s", err)
	}

	compareMany(t, testData, d1, d2, d3, d4, d5)
}

func TestContainerMisc(t *testing.T) {
	c1 := NewContainer()
	d1 := c1.CompileData()
	if len(d1) > 0 {
		t.Fatalf("empty container should not hold any data")
	}
}
