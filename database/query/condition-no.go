package query

type noCond struct {
}

func (c *noCond) complies(f Fetcher) bool {
	return true
}

func (c *noCond) check() (err error) {
	return nil
}

func (c *noCond) string() string {
	return ""
}
