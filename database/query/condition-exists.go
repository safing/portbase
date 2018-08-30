package query

import (
	"errors"
	"fmt"
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

func (c *existsCondition) complies(f Fetcher) bool {
	return f.Exists(c.key)
}

func (c *existsCondition) check() error {
	if c.operator == errorPresent {
		return errors.New(c.key)
	}
	return nil
}

func (c *existsCondition) string() string {
	return fmt.Sprintf("%s %s", c.key, getOpName(c.operator))
}
