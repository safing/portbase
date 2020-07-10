package query

import (
	"fmt"

	"github.com/safing/portbase/database/accessor"
)

type existsCondition struct {
	key      string
	operator uint8
}

func newExistsCondition(key string, operator uint8) *existsCondition {
	return &existsCondition{
		key:      key,
		operator: operator,
	}
}

func (c *existsCondition) complies(acc accessor.Accessor) bool {
	return acc.Exists(c.key)
}

func (c *existsCondition) check() error {
	if c.operator == errorPresent {
		return conditionKeyError(c.key)
	}
	return nil
}

func (c *existsCondition) string() string {
	return fmt.Sprintf("%s %s", escapeString(c.key), getOpName(c.operator))
}
