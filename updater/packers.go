package updater

import (
	"compress/gzip"
	"io"
)

// UnpackGZIP unpacks a GZIP compressed reader r
// and returns a new reader. It's suitable to be
// used with registry.GetPackedFile.
func UnpackGZIP(r io.Reader) (io.Reader, error) {
	return gzip.NewReader(r)
}
