package dsd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

// HTTP Related Errors.
var (
	ErrMissingBody        = errors.New("dsd: missing http body")
	ErrMissingContentType = errors.New("dsd: missing http content type")
)

const (
	httpHeaderContentType = "Content-Type"
)

// LoadFromHTTPRequest loads the data from the body into the given interface.
func LoadFromHTTPRequest(r *http.Request, t interface{}) (format uint8, err error) {
	return loadFromHTTP(r.Body, r.Header.Get(httpHeaderContentType), t)
}

// LoadFromHTTPResponse loads the data from the body into the given interface.
// Closing the body is left to the caller.
func LoadFromHTTPResponse(resp *http.Response, t interface{}) (format uint8, err error) {
	return loadFromHTTP(resp.Body, resp.Header.Get(httpHeaderContentType), t)
}

func loadFromHTTP(body io.Reader, mimeType string, t interface{}) (format uint8, err error) {
	// Read full body.
	data, err := io.ReadAll(body)
	if err != nil {
		return 0, fmt.Errorf("dsd: failed to read http body: %w", err)
	}

	// Get mime type from header, then check, clean and verify it.
	if mimeType == "" {
		return 0, ErrMissingContentType
	}
	mimeType, _, err = mime.ParseMediaType(mimeType)
	if err != nil {
		return 0, fmt.Errorf("dsd: failed to parse content type: %w", err)
	}
	format, ok := MimeTypeToFormat[mimeType]
	if !ok {
		return 0, ErrIncompatibleFormat
	}

	// Parse data..
	return format, LoadAsFormat(data, format, t)
}

// RequestHTTPResponseFormat sets the Accept header to the given format.
func RequestHTTPResponseFormat(r *http.Request, format uint8) (mimeType string, err error) {
	// Get mime type.
	mimeType, ok := FormatToMimeType[format]
	if !ok {
		return "", ErrIncompatibleFormat
	}
	// Omit charset.
	mimeType, _, err = mime.ParseMediaType(mimeType)
	if err != nil {
		return "", fmt.Errorf("dsd: failed to parse content type: %w", err)
	}

	// Request response format.
	r.Header.Set("Accept", mimeType)

	return mimeType, nil
}

// DumpToHTTPRequest dumps the given data to the HTTP request using the given
// format. It also sets the Accept header to the same format.
func DumpToHTTPRequest(r *http.Request, t interface{}, format uint8) error {
	mimeType, err := RequestHTTPResponseFormat(r, format)
	if err != nil {
		return err
	}

	// Serialize data.
	data, err := dumpWithoutIdentifier(t, format, "")
	if err != nil {
		return fmt.Errorf("dsd: failed to serialize: %w", err)
	}

	// Set body.
	r.Header.Set("Content-Type", mimeType)
	r.Body = io.NopCloser(bytes.NewReader(data))

	return nil
}

// DumpToHTTPResponse dumpts the given data to the HTTP response, using the
// format defined in the request's Accept header.
func DumpToHTTPResponse(w http.ResponseWriter, r *http.Request, t interface{}) error {
	// Get format from Accept header.
	// TODO: Improve parsing of Accept header.
	mimeType := r.Header.Get("Accept")
	format, ok := MimeTypeToFormat[mimeType]
	if !ok {
		return ErrIncompatibleFormat
	}

	// Serialize data.
	data, err := dumpWithoutIdentifier(t, format, "")
	if err != nil {
		return fmt.Errorf("dsd: failed to serialize: %w", err)
	}

	// Write data to response
	w.Header().Set("Content-Type", mimeType)
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("dsd: failed to write response: %w", err)
	}
	return nil
}

// Format and MimeType mappings.
var (
	FormatToMimeType = map[uint8]string{
		JSON:    "application/json; charset=utf-8",
		CBOR:    "application/cbor",
		MsgPack: "application/msgpack",
	}
	MimeTypeToFormat = map[string]uint8{
		"application/json":    JSON,
		"application/cbor":    CBOR,
		"application/msgpack": MsgPack,
	}
)
