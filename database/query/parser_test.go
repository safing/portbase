package query

import (
	"testing"
)

func TestMakeTree(t *testing.T) {
	text1 := `(age > 100 and experience <= "99") or (age < 10 and motivation > 50) or name matches Mr.\"X or name matches y`
	snippets, err := extractSnippets(text1)
	if err != nil {
		t.Errorf("failed to make tree: %s", err)
	} else {
		for _, el := range snippets {
			t.Errorf("%+v", el)
		}
	}
	// t.Error(spew.Sprintf("%v", treeElement))
}
