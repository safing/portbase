package iterator

import (
	"github.com/Safing/portbase/database/model"
)

// Iterator defines the iterator structure.
type Iterator struct {
	Next  chan model.Model
	Error error
}

// New creates a new Iterator.
func New() *Iterator {
	return &Iterator{
		Next: make(chan model.Model, 10),
	}
}
