package iterator

import (
	"github.com/Safing/portbase/database/record"
)

// Iterator defines the iterator structure.
type Iterator struct {
	Next  chan record.Record
	Done  chan struct{}
	Error error
}

// New creates a new Iterator.
func New() *Iterator {
	return &Iterator{
		Next: make(chan record.Record, 10),
		Done: make(chan struct{}),
	}
}
