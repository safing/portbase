package dsd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	ErrMissingBody        = errors.New("dsd: missing http body")
	ErrMissingContentType = errors.New("dsd: missing http content type")
)

const (
	httpHeaderContentType = "Content-Type"
)

func LoadFromHTTPRequest(r *http.Request, t interface{}) (format uint8, err error) {
	if r.Body == nil {
		return 0, ErrMissingBody
	}
	defer r.Body.Close()

	return loadFromHTTP(r.Body, r.Header.Get(httpHeaderContentType), t)
}

func LoadFromHTTPResponse(resp *http.Response, t interface{}) (format uint8, err error) {
	if resp.Body == nil {
		return 0, ErrMissingBody
	}
	defer resp.Body.Close()

	return loadFromHTTP(resp.Body, resp.Header.Get(httpHeaderContentType), t)
}

func loadFromHTTP(body io.Reader, mimeType string, t interface{}) (format uint8, err error) {
	// Read full body.
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return 0, fmt.Errorf("dsd: failed to read http body: %w", err)
	}

	// Get mime type from header, then check, clean and verify it.
	if mimeType == "" {
		return 0, ErrMissingContentType
	}
	if strings.Contains(mimeType, ";") {
		mimeType = strings.SplitN(mimeType, ";", 2)[0]
	}
	format, ok := MimeTypeToFormat[mimeType]
	if !ok {
		return 0, ErrIncompatibleFormat
	}

	// Parse data..
	return format, LoadAsFormat(data, format, t)
}

func RequestHTTPResponseFormat(r *http.Request, format uint8) (mimeType string, err error) {
	// Get mime type.
	mimeType, ok := FormatToMimeType[format]
	if !ok {
		return "", ErrIncompatibleFormat
	}

	// Request response format.
	r.Header.Set("Accept", mimeType)

	return mimeType, nil
}

func DumpToHTTPRequest(r *http.Request, t interface{}, format uint8) error {
	mimeType, err := RequestHTTPResponseFormat(r, format)
	if err != nil {
		return err
	}

	// Serialize data.
	data, err := Dump(t, format)
	if err != nil {
		return fmt.Errorf("dsd: failed to serialize: %w", err)
	}

	// Set body.
	r.Header.Set("Content-Type", mimeType)
	r.Body = ioutil.NopCloser(bytes.NewReader(data))

	return nil
}

func DumpToHTTPResponse(w http.ResponseWriter, r *http.Request, t interface{}, fallbackFormat uint8) error {
	// Get format from Accept header.
	format, ok := MimeTypeToFormat[r.Header.Get("Accept")]
	if !ok {
		format = fallbackFormat
	}
	mimeType, ok := FormatToMimeType[format]
	if !ok {
		return ErrIncompatibleFormat
	}

	// Serialize data.
	data, err := Dump(t, format)
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
