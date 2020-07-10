package dsd

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"github.com/safing/portbase/formats/varint"
)

// DumpAndCompress stores the interface as a dsd formatted data structure and compresses the resulting data.
func DumpAndCompress(t interface{}, format, compression uint8) ([]byte, error) {
	data, err := Dump(t, format)
	if err != nil {
		return nil, err
	}

	// handle special cases
	switch compression {
	case NONE:
		return data, nil
	case AUTO:
		compression = GZIP
	}

	// prepare writer
	packetFormat := varint.Pack8(compression)
	buf := bytes.NewBuffer(nil)
	buf.Write(packetFormat)

	// compress
	switch compression {
	case GZIP:
		// create gzip writer
		gzipWriter, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
		if err != nil {
			return nil, err
		}

		// write data
		_, err = gzipWriter.Write(data) // no need to check the number of bytes written
		if err != nil {
			return nil, err
		}

		// flush and write gzip footer
		err = gzipWriter.Close()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%d: %w", format, errUnsupportedFormat)
	}

	return buf.Bytes(), nil
}

// DecompressAndLoad decompresses the data using the specified compression format and then loads the resulting data blob into the interface.
func DecompressAndLoad(data []byte, format uint8, t interface{}) (interface{}, error) {
	// prepare reader
	buf := bytes.NewBuffer(nil)

	// decompress
	switch format {
	case GZIP:
		// create gzip reader
		gzipReader, err := gzip.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return nil, err
		}

		// read uncompressed data
		_, err = buf.ReadFrom(gzipReader)
		if err != nil {
			return nil, err
		}

		// flush and verify gzip footer
		err = gzipReader.Close()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%d: %w", format, errUnsupportedFormat)
	}

	// assign decompressed data
	data = buf.Bytes()

	// get format
	format, read, err := varint.Unpack8(data)
	if err != nil {
		return nil, err
	}
	if len(data) <= read {
		return nil, errNoMoreSpace
	}

	return LoadAsFormat(data[read:], format, t)
}
