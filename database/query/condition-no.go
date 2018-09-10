package query

import (
	"github.com/Safing/portbase/database/accessor"
)

type noCond struct {
}

func (c *noCond) complies(acc accessor.Accessor) bool {
	return true
}

func (c *noCond) check() (err error) {
	return nil
}

func (c *noCond) string() string {
	return ""
}
