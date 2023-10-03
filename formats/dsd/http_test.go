package dsd

import (
	"mime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMimeTypes(t *testing.T) {
	t.Parallel()

	// Test static maps.
	for _, mimeType := range FormatToMimeType {
		cleaned, _, err := mime.ParseMediaType(mimeType)
		assert.NoError(t, err, "mime type must be parse-able")
		assert.Equal(t, mimeType, cleaned, "mime type should be clean in map already")
	}
	for mimeType := range MimeTypeToFormat {
		cleaned, _, err := mime.ParseMediaType(mimeType)
		assert.NoError(t, err, "mime type must be parse-able")
		assert.Equal(t, mimeType, cleaned, "mime type should be clean in map already")
	}

	// Test assumptions.
	for mimeType, mimeTypeCleaned := range map[string]string{
		"application/xml, image/webp":       "xml",
		"application/xml;q=0.9, image/webp": "xml",
		"*":                                 "*",
		"*/*":                               "*",
		"text/yAMl":                         "yaml",
	} {
		cleaned := extractMimeType(mimeType)
		assert.Equal(t, mimeTypeCleaned, cleaned, "assumption for %q should hold", mimeType)
	}
}
