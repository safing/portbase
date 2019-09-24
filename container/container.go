package container

import (
	"errors"

	"github.com/safing/portbase/formats/varint"
)

// Container is []byte sclie on steroids, allowing for quick data appending, prepending and fetching as well as transparent error transportation. (Error transportation requires use of varints for data)
type Container struct {
	compartments [][]byte
	offset       int
	err          error
}

// Data Handling

// NewContainer is DEPRECATED, please use New(), it's the same thing.
func NewContainer(data ...[]byte) *Container {
	return &Container{
		compartments: data,
	}
}

// New creates a new container with an optional initial []byte slice. Data will NOT be copied.
func New(data ...[]byte) *Container {
	return &Container{
		compartments: data,
	}
}

// Prepend prepends data. Data will NOT be copied.
func (c *Container) Prepend(data []byte) {
	if c.offset < 1 {
		c.renewCompartments()
	}
	c.offset--
	c.compartments[c.offset] = data
}

// Append appends the given data. Data will NOT be copied.
func (c *Container) Append(data []byte) {
	c.compartments = append(c.compartments, data)
}

// AppendNumber appends a number (varint encoded).
func (c *Container) AppendNumber(n uint64) {
	c.compartments = append(c.compartments, varint.Pack64(n))
}

// AppendAsBlock appends the length of the data and the data itself. Data will NOT be copied.
func (c *Container) AppendAsBlock(data []byte) {
	c.AppendNumber(uint64(len(data)))
	c.Append(data)
}

// Length returns the full length of all bytes held by the container.
func (c *Container) Length() (length int) {
	for i := c.offset; i < len(c.compartments); i++ {
		length += len(c.compartments[i])
	}
	return
}

// Replace replaces all held data with a new data slice. Data will NOT be copied.
func (c *Container) Replace(data []byte) {
	c.compartments = [][]byte{data}
}

// CompileData concatenates all bytes held by the container and returns it as one single []byte slice. Data will NOT be copied and is NOT consumed.
func (c *Container) CompileData() []byte {
	if len(c.compartments) != 1 {
		newBuf := make([]byte, c.Length())
		copyBuf := newBuf
		for i := c.offset; i < len(c.compartments); i++ {
			copy(copyBuf, c.compartments[i])
			copyBuf = copyBuf[len(c.compartments[i]):]
		}
		c.compartments = [][]byte{newBuf}
		c.offset = 0
	}
	return c.compartments[0]
}

// Get returns the given amount of bytes. Data MAY be copied and IS consumed.
func (c *Container) Get(n int) ([]byte, error) {
	buf := c.gather(n)
	if len(buf) < n {
		return nil, errors.New("container: not enough data to return")
	}
	c.skip(len(buf))
	return buf, nil
}

// GetMax returns as much as possible, but the given amount of bytes at maximum. Data MAY be copied and IS consumed.
func (c *Container) GetMax(n int) []byte {
	buf := c.gather(n)
	c.skip(len(buf))
	return buf
}

// WriteToSlice copies data to the give slice until it is full, or the container is empty. It returns the bytes written and if the container is now empty. Data IS copied and IS consumed.
func (c *Container) WriteToSlice(slice []byte) (n int, containerEmptied bool) {
	for i := c.offset; i < len(c.compartments); i++ {
		copy(slice, c.compartments[i])
		if len(slice) < len(c.compartments[i]) {
			// only part was copied
			n += len(slice)
			c.compartments[i] = c.compartments[i][len(slice):]
			c.checkOffset()
			return n, false
		}
		// all was copied
		n += len(c.compartments[i])
		slice = slice[len(c.compartments[i]):]
		c.compartments[i] = nil
		c.offset = i + 1
	}
	c.checkOffset()
	return n, true
}

func (c *Container) clean() {
	if c.offset > 100 {
		c.renewCompartments()
	}
}

func (c *Container) renewCompartments() {
	baseLength := len(c.compartments) - c.offset + 5
	newCompartments := make([][]byte, baseLength, baseLength+5)
	copy(newCompartments[5:], c.compartments[c.offset:])
	c.compartments = newCompartments
	c.offset = 4
}

func (c *Container) carbonCopy() *Container {
	new := &Container{
		compartments: make([][]byte, len(c.compartments)),
		offset:       c.offset,
		err:          c.err,
	}
	for i := 0; i < len(c.compartments); i++ {
		new.compartments[i] = c.compartments[i]
	}
	// TODO: investigate why copy fails to correctly duplicate [][]byte
	// copy(new.compartments, c.compartments)
	return new
}

func (c *Container) checkOffset() {
	if c.offset >= len(c.compartments) {
		c.offset = len(c.compartments) / 2
	}
}

// Error Handling

// SetError sets an error.
func (c *Container) SetError(err error) {
	c.err = err
	c.Replace(append([]byte{0x00}, []byte(err.Error())...))
}

// CheckError checks if there is an error in the data. If so, it will parse the error and delete the data.
func (c *Container) CheckError() {
	if len(c.compartments[c.offset]) > 0 && c.compartments[c.offset][0] == 0x00 {
		c.compartments[c.offset] = c.compartments[c.offset][1:]
		c.err = errors.New(string(c.CompileData()))
		c.compartments = nil
	}
}

// HasError returns wether or not the container is holding an error.
func (c *Container) HasError() bool {
	return c.err != nil
}

// Error returns the error.
func (c *Container) Error() error {
	return c.err
}

// ErrString returns the error as a string.
func (c *Container) ErrString() string {
	return c.err.Error()
}

// Block Handling

// PrependLength prepends the current full length of all bytes in the container.
func (c *Container) PrependLength() {
	c.Prepend(varint.Pack64(uint64(c.Length())))
}

func (c *Container) gather(n int) []byte {
	// check if first slice holds enough data
	if len(c.compartments[c.offset]) >= n {
		return c.compartments[c.offset][:n]
	}
	// start gathering data
	slice := make([]byte, n)
	copySlice := slice
	n = 0
	for i := c.offset; i < len(c.compartments); i++ {
		copy(copySlice, c.compartments[i])
		if len(copySlice) <= len(c.compartments[i]) {
			n += len(copySlice)
			return slice[:n]
		}
		n += len(c.compartments[i])
		copySlice = copySlice[len(c.compartments[i]):]
	}
	return slice[:n]
}

func (c *Container) skip(n int) {
	for i := c.offset; i < len(c.compartments); i++ {
		if len(c.compartments[i]) <= n {
			n -= len(c.compartments[i])
			c.offset = i + 1
			c.compartments[i] = nil
			if n == 0 {
				c.checkOffset()
				return
			}
		} else {
			c.compartments[i] = c.compartments[i][n:]
			c.checkOffset()
			return
		}
	}
	c.checkOffset()
}

// GetNextBlock returns the next block of data defined by a varint (note: data will MAY be copied and IS consumed).
func (c *Container) GetNextBlock() ([]byte, error) {
	blockSize, err := c.GetNextN64()
	if err != nil {
		return nil, err
	}
	return c.Get(int(blockSize))
}

// GetNextN8 parses and returns a varint of type uint8.
func (c *Container) GetNextN8() (uint8, error) {
	buf := c.gather(2)
	num, n, err := varint.Unpack8(buf)
	if err != nil {
		return 0, err
	}
	c.skip(n)
	return num, nil
}

// GetNextN16 parses and returns a varint of type uint16.
func (c *Container) GetNextN16() (uint16, error) {
	buf := c.gather(3)
	num, n, err := varint.Unpack16(buf)
	if err != nil {
		return 0, err
	}
	c.skip(n)
	return num, nil
}

// GetNextN32 parses and returns a varint of type uint32.
func (c *Container) GetNextN32() (uint32, error) {
	buf := c.gather(5)
	num, n, err := varint.Unpack32(buf)
	if err != nil {
		return 0, err
	}
	c.skip(n)
	return num, nil
}

// GetNextN64 parses and returns a varint of type uint64.
func (c *Container) GetNextN64() (uint64, error) {
	buf := c.gather(9)
	num, n, err := varint.Unpack64(buf)
	if err != nil {
		return 0, err
	}
	c.skip(n)
	return num, nil
}
