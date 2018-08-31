package query

import "testing"

func TestGetOpName(t *testing.T) {
	if getOpName(254) != "[unknown]" {
		t.Error("unexpected output")
	}
}
