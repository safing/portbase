package record

import "testing"

func TestBaseRecord(t *testing.T) {

	// check model interface compliance
	var m Record
	b := &TestRecord{}
	m = b
	_ = m

}
