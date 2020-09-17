package runtime

import "github.com/safing/portbase/database/record"

// singleRecordReader is a convenience type for read-only exposing
// a single record.Record. Note that users must lock the whole record
// themself before performing any manipulation on the record.
type singleRecordReader struct {
	record.Record
}

// ProvideRecord returns a ValueProvider the exposes read-only
// access to r. Users of ProvideRecord need to ensure the lock
// the whole record before performing modifications on it.
//
// Example:
//
//		type MyValue struct {
//			record.Base
//			Value string
//		}
//		r := new(MyValue)
//		pushUpdate, _ := runtime.Register("my/key", ProvideRecord(r))
//		r.Lock()
//		r.Value = "foobar"
//		r.Unlock()
//		pushUpdate(r)
//
func ProvideRecord(r record.Record) ValueProvider {
	return &singleRecordReader{r}
}

// Set implements ValueProvider.Set and returns ErrReadOnly.
func (sr *singleRecordReader) Set(_ record.Record) (record.Record, error) {
	return nil, ErrReadOnly
}

// Get implements ValueProvider.Get and returns the wrapped record.Record
// but only if keyOrPrefix exactly matches the records database key.
func (sr *singleRecordReader) Get(keyOrPrefix string) ([]record.Record, error) {
	if keyOrPrefix != sr.Record.Key() {
		return nil, nil
	}
	return []record.Record{sr.Record}, nil
}
