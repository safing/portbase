package utils

import (
	"bytes"
	"testing"
)

var (
	stringTestSlice  = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	stringTestSlice2 = []string{"a", "x", "x", "x", "x", "x", "x", "x", "x", "j"}
	stringTestSlice3 = []string{"a", "x"}
	byteTestSlice    = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
)

func TestStringInSlice(t *testing.T) {
	if !StringInSlice("a", stringTestSlice) {
		t.Fatal("string reported not in slice (1), but it is")
	}
	if !StringInSlice("d", stringTestSlice) {
		t.Fatal("string reported not in slice (2), but it is")
	}
	if !StringInSlice("j", stringTestSlice) {
		t.Fatal("string reported not in slice (3), but it is")
	}

	if StringInSlice("0", stringTestSlice) {
		t.Fatal("string reported in slice (1), but is not")
	}
	if StringInSlice("x", stringTestSlice) {
		t.Fatal("string reported in slice (2), but is not")
	}
	if StringInSlice("k", stringTestSlice) {
		t.Fatal("string reported in slice (3), but is not")
	}
}

func TestRemoveFromStringSlice(t *testing.T) {
	test1 := DuplicateStrings(stringTestSlice)
	test1 = RemoveFromStringSlice(test1, "b")
	if StringInSlice("b", test1) {
		t.Fatal("string reported in slice, but was removed")
	}
	if len(test1) != len(stringTestSlice)-1 {
		t.Fatalf("new string slice length not as expected: is %d, should be %d\nnew slice is %v", len(test1), len(stringTestSlice)-1, test1)
	}
}

func TestDuplicateStrings(t *testing.T) {
	a := DuplicateStrings(stringTestSlice)
	if !StringSliceEqual(a, stringTestSlice) {
		t.Fatal("copied string slice is not equal")
	}
	a[0] = "x"
	if StringSliceEqual(a, stringTestSlice) {
		t.Fatal("copied string slice is not a real copy")
	}
}

func TestStringSliceEqual(t *testing.T) {
	if !StringSliceEqual(stringTestSlice, stringTestSlice) {
		t.Fatal("strings are equal, but are reported as not")
	}
	if StringSliceEqual(stringTestSlice, stringTestSlice2) {
		t.Fatal("strings are not equal (1), but are reported as equal")
	}
	if StringSliceEqual(stringTestSlice, stringTestSlice3) {
		t.Fatal("strings are not equal (1), but are reported as equal")
	}
}

func TestDuplicateBytes(t *testing.T) {
	a := DuplicateBytes(byteTestSlice)
	if !bytes.Equal(a, byteTestSlice) {
		t.Fatal("copied bytes slice is not equal")
	}
	a[0] = 0xff
	if bytes.Equal(a, byteTestSlice) {
		t.Fatal("copied bytes slice is not a real copy")
	}
}
