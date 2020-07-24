package varint

// PrependLength prepends the varint encoded length of the byte slice to itself.
func PrependLength(data []byte) []byte {
	return append(Pack64(uint64(len(data))), data...)
}

// GetNextBlock extract the integer from the beginning of the given byte slice and returns the remaining bytes, the extracted integer, and whether there was an error.
func GetNextBlock(data []byte) ([]byte, int, error) {
	l, n, err := Unpack64(data)
	if err != nil {
		return nil, 0, err
	}
	length := int(l)
	totalLength := length + n
	if totalLength > len(data) {
		return nil, 0, errTooSmall
	}
	return data[n:totalLength], totalLength, nil
}
