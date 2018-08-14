package config

import (
	"testing"
)

func TestGet(t *testing.T) {

	err := SetConfig(`
  {
    "monkey": "1",
		"zebra": ["black", "white"],
    "elephant": 2,
		"hot": true,
		"cold": false
  }
  `)
	if err != nil {
		t.Fatal(err)
	}

	err = SetDefaultConfig(`
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
		t.Fatalf("monkey should be 1, is %s", monkey())
	}

	zebra := GetAsStringArray("zebra", []string{})
	if len(zebra()) != 2 || zebra()[0] != "black" || zebra()[1] != "white" {
		t.Fatalf("zebra should be [\"black\", \"white\"], is %v", zebra())
	}

	elephant := GetAsInt("elephant", -1)
	if elephant() != 2 {
		t.Fatalf("elephant should be 2, is %d", elephant())
	}

	hot := GetAsBool("hot", false)
	if !hot() {
		t.Fatalf("hot should be true, is %v", hot())
	}

	cold := GetAsBool("cold", true)
	if cold() {
		t.Fatalf("cold should be false, is %v", cold())
	}

	err = SetConfig(`
  {
    "monkey": "3"
  }
  `)
	if err != nil {
		t.Fatal(err)
	}

	if monkey() != "3" {
		t.Fatalf("monkey should be 0, is %s", monkey())
	}

	if elephant() != 0 {
		t.Fatalf("elephant should be 0, is %d", elephant())
	}

	zebra()
	hot()

}

func BenchmarkGetAsStringCached(b *testing.B) {
	// Setup
	err := SetConfig(`
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
	err := SetConfig(`
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
	err := SetConfig(`
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
	err := SetConfig(`
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
