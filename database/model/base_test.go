package model

import "testing"

func TestBaseModel(t *testing.T) {

	// check model interface compliance
	var m Model
	b := &TestModel{}
	m = b
	_ = m

}
