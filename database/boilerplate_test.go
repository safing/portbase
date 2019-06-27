package database

import (
	"fmt"
	"sync"

	"github.com/safing/portbase/database/record"
)

type Example struct {
	record.Base
	sync.Mutex

	Name  string
	Score int
}

var (
	exampleDB = NewInterface(nil)
)

// GetExample gets an Example from the database.
func GetExample(key string) (*Example, error) {
	r, err := exampleDB.Get(key)
	if err != nil {
		return nil, err
	}

	// unwrap
	if r.IsWrapped() {
		// only allocate a new struct, if we need it
		new := &Example{}
		err = record.Unwrap(r, new)
		if err != nil {
			return nil, err
		}
		return new, nil
	}

	// or adjust type
	new, ok := r.(*Example)
	if !ok {
		return nil, fmt.Errorf("record not of type *Example, but %T", r)
	}
	return new, nil
}

func (e *Example) Save() error {
	return exampleDB.Put(e)
}

func (e *Example) SaveAs(key string) error {
	e.SetKey(key)
	return exampleDB.PutNew(e)
}

func NewExample(key, name string, score int) *Example {
	new := &Example{
		Name:  name,
		Score: score,
	}
	new.SetKey(key)
	return new
}
