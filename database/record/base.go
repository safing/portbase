package record

import (
	"errors"
	"fmt"

	"github.com/Safing/portbase/container"
	"github.com/Safing/portbase/formats/dsd"
)

// Base provides a quick way to comply with the Model interface.
type Base struct {
	dbName string
	dbKey  string
	meta   *Meta
}

// Key returns the key of the database record.
func (b *Base) Key() string {
	return fmt.Sprintf("%s:%s", b.dbName, b.dbKey)
}

// DatabaseName returns the name of the database.
func (b *Base) DatabaseName() string {
	return b.dbName
}

// DatabaseKey returns the database key of the database record.
func (b *Base) DatabaseKey() string {
	return b.dbKey
}

// SetKey sets the key on the database record, it should only be called after loading the record. Use MoveTo to save the record with another key.
func (b *Base) SetKey(key string) {
	b.dbName, b.dbKey = ParseKey(key)
}

// MoveTo sets a new key for the record and resets all metadata, except for the secret and crownjewel status.
func (b *Base) MoveTo(key string) {
	b.SetKey(key)
	b.meta.Reset()
}

// Meta returns the metadata object for this record.
func (b *Base) Meta() *Meta {
	return b.meta
}

// SetMeta sets the metadata on the database record, it should only be called after loading the record. Use MoveTo to save the record with another key.
func (b *Base) SetMeta(meta *Meta) {
	b.meta = meta
}

// Marshal marshals the object, without the database key or metadata
func (b *Base) Marshal(format uint8) ([]byte, error) {
	dumped, err := dsd.Dump(b, format)
	if err != nil {
		return nil, err
	}
	return dumped, nil
}

// MarshalRecord packs the object, including metadata, into a byte array for saving in a database.
func (b *Base) MarshalRecord() ([]byte, error) {
	if b.Meta() == nil {
		return nil, errors.New("missing meta")
	}

	// version
	c := container.New([]byte{1})

	// meta
	metaSection, err := b.meta.GenCodeMarshal(nil)
	if err != nil {
		return nil, err
	}
	c.AppendAsBlock(metaSection)

	// data
	dataSection, err := b.Marshal(dsd.JSON)
	if err != nil {
		return nil, err
	}
	c.Append(dataSection)

	return c.CompileData(), nil
}
