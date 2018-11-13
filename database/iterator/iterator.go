package iterator

import (
	"github.com/Safing/portbase/database/record"
)

// Iterator defines the iterator structure.
type Iterator struct {
	Next chan record.Record
	Done chan struct{}
	Err  error
}

// New creates a new Iterator.
func New() *Iterator {
	return &Iterator{
		Next: make(chan record.Record, 10),
		Done: make(chan struct{}),
	}
}

func (it *Iterator) Finish(err error) {
	close(it.Next)
	close(it.Done)
	it.Err = err
}
