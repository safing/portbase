package database

// Interface provides a method to access the database with attached options.
type Interface struct {}

// Options holds options that may be set for an Interface instance.
type Options struct {
  Local bool
  Internal bool
  AlwaysMakeSecret bool
  AlwaysMakeCrownjewel bool
}

// NewInterface returns a new Interface to the database.
func NewInterface(opts *Options) *Interface {
  return &Interface{
    local: local,
    internal: internal,
  }
}

func (i *Interface) Get(key string) (record.Record, error) {

  controller

  return nil, nil
}
