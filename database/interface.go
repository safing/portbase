package database

type Interface struct {
  local bool
  internal bool
}

func NewInterface(local bool, internal bool) *Interface {
  return &Interface{
    local: local,
    internal: internal,
  }
}

func (i *Interface) Get(string key) (model.Model, error) {
  return nil, nil
}
